package observability

import (
	"context"
	"errors"
	"fmt"
	"time"

	observabilityv1 "OpenCNC_config_service/common/structures/logging"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/proto"
)

const (
	defaultTopicLogs    = "opencnc.logs"
	defaultTopicEvents  = "opencnc.events"
	defaultTopicMetrics = "opencnc.metrics"
	defaultTopicAudit   = "opencnc.audit"
	defaultTopicHealth  = "opencnc.health"

	defaultBatchTimeout = 10 * time.Millisecond
)

var (
	ErrNilEnvelope           = errors.New("event envelope is nil")
	ErrMissingSchemaVersion  = errors.New("schema_version is required")
	ErrMissingEventID        = errors.New("event_id is required")
	ErrMissingOccurredAt     = errors.New("occurred_at is required")
	ErrInvalidOccurredAt     = errors.New("occurred_at is invalid")
	ErrMissingSource         = errors.New("source is required")
	ErrMissingSourceService  = errors.New("source.service is required")
	ErrMissingTrace          = errors.New("trace is required")
	ErrMissingCorrelationID  = errors.New("trace.correlation_id is required")
	ErrMissingKind           = errors.New("kind is required")
	ErrMissingSeverity       = errors.New("severity is required")
	ErrMissingPayload        = errors.New("payload is required")
	ErrUnsupportedKind       = errors.New("unsupported event kind")
	ErrPayloadKindMismatch   = errors.New("payload does not match event kind")
	ErrMissingKafkaBrokers   = errors.New("brokers are required when Kafka publishing is enabled")
	ErrMissingTopicForKind   = errors.New("topic mapping for kind is missing")
	ErrMissingWriterForTopic = errors.New("kafka writer for topic is not configured")
	ErrSchemaVersionMismatch = errors.New("schema_version does not match required version")
)

type Config struct {
	Enabled               bool
	KafkaEnabled          bool
	FailOpen              bool
	Brokers               []string
	TopicMap              map[observabilityv1.EventKind]string
	RequiredSchemaVersion string
	WriterBatchTimeout    time.Duration
	WriterBalancer        kafka.Balancer
}

type Producer struct {
	cfg     Config
	writers map[string]*kafka.Writer
}

func DefaultTopicMap() map[observabilityv1.EventKind]string {
	return map[observabilityv1.EventKind]string{
		observabilityv1.EventKind_EVENT_KIND_LOG:    defaultTopicLogs,
		observabilityv1.EventKind_EVENT_KIND_DOMAIN: defaultTopicEvents,
		observabilityv1.EventKind_EVENT_KIND_METRIC: defaultTopicMetrics,
		observabilityv1.EventKind_EVENT_KIND_AUDIT:  defaultTopicAudit,
		observabilityv1.EventKind_EVENT_KIND_HEALTH: defaultTopicHealth,
	}
}

func NewProducer(cfg Config) (*Producer, error) {
	normalized := cfg
	if normalized.TopicMap == nil {
		normalized.TopicMap = DefaultTopicMap()
	} else {
		base := DefaultTopicMap()
		for kind, topic := range normalized.TopicMap {
			base[kind] = topic
		}
		normalized.TopicMap = base
	}

	if normalized.WriterBatchTimeout <= 0 {
		normalized.WriterBatchTimeout = defaultBatchTimeout
	}

	producer := &Producer{
		cfg:     normalized,
		writers: map[string]*kafka.Writer{},
	}

	if !producer.Enabled() {
		return producer, nil
	}

	if len(normalized.Brokers) == 0 {
		return nil, ErrMissingKafkaBrokers
	}

	balancer := normalized.WriterBalancer
	if balancer == nil {
		balancer = &kafka.Hash{}
	}

	for _, topic := range uniqueTopics(normalized.TopicMap) {
		producer.writers[topic] = &kafka.Writer{
			Addr:         kafka.TCP(normalized.Brokers...),
			Topic:        topic,
			BatchTimeout: normalized.WriterBatchTimeout,
			Balancer:     balancer,
			RequiredAcks: kafka.RequireOne,
		}
	}

	return producer, nil
}

func (p *Producer) Enabled() bool {
	return p != nil && p.cfg.Enabled && p.cfg.KafkaEnabled
}

func (p *Producer) Publish(ctx context.Context, event *observabilityv1.EventEnvelope) error {
	if p == nil || !p.Enabled() {
		return nil
	}

	if err := validateEnvelope(event, p.cfg.RequiredSchemaVersion); err != nil {
		return err
	}

	topic, err := topicForKind(event.GetKind(), p.cfg.TopicMap)
	if err != nil {
		return err
	}

	writer, ok := p.writers[topic]
	if !ok {
		return fmt.Errorf("%w: %s", ErrMissingWriterForTopic, topic)
	}

	payload, err := proto.Marshal(event)
	//payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(partitionKey(event)),
		Value: payload,
		Time:  event.GetOccurredAt().AsTime(),
	}

	if err := writer.WriteMessages(ctx, msg); err != nil {
		if p.cfg.FailOpen {
			return nil
		}
		return err
	}

	return nil
}

func (p *Producer) Close() error {
	if p == nil {
		return nil
	}

	var joined error
	for topic, writer := range p.writers {
		if writer == nil {
			continue
		}
		if err := writer.Close(); err != nil {
			joined = errors.Join(joined, fmt.Errorf("close writer %s: %w", topic, err))
		}
	}

	return joined
}

func validateEnvelope(event *observabilityv1.EventEnvelope, requiredSchemaVersion string) error {
	if event == nil {
		return ErrNilEnvelope
	}

	if event.GetSchemaVersion() == "" {
		return ErrMissingSchemaVersion
	}

	if requiredSchemaVersion != "" && event.GetSchemaVersion() != requiredSchemaVersion {
		return fmt.Errorf("%w: got %q want %q", ErrSchemaVersionMismatch, event.GetSchemaVersion(), requiredSchemaVersion)
	}

	if event.GetEventId() == "" {
		return ErrMissingEventID
	}

	if event.GetOccurredAt() == nil {
		return ErrMissingOccurredAt
	}

	if err := event.GetOccurredAt().CheckValid(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidOccurredAt, err)
	}

	if event.GetSource() == nil {
		return ErrMissingSource
	}

	if event.GetSource().GetService() == "" {
		return ErrMissingSourceService
	}

	if event.GetTrace() == nil {
		return ErrMissingTrace
	}

	if event.GetTrace().GetCorrelationId() == "" {
		return ErrMissingCorrelationID
	}

	if event.GetKind() == observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED {
		return ErrMissingKind
	}

	if event.GetSeverity() == observabilityv1.Severity_SEVERITY_UNSPECIFIED {
		return ErrMissingSeverity
	}

	payloadKind, err := payloadKind(event.GetPayload())
	if err != nil {
		return err
	}

	if payloadKind != event.GetKind() {
		return fmt.Errorf("%w: payload=%s kind=%s", ErrPayloadKindMismatch, payloadKind.String(), event.GetKind().String())
	}

	return nil
}

func payloadKind(payload any) (observabilityv1.EventKind, error) {
	switch p := payload.(type) {
	case nil:
		return observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED, ErrMissingPayload
	case *observabilityv1.EventEnvelope_Log:
		if p.Log == nil {
			return observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED, ErrMissingPayload
		}
		return observabilityv1.EventKind_EVENT_KIND_LOG, nil
	case *observabilityv1.EventEnvelope_Domain:
		if p.Domain == nil {
			return observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED, ErrMissingPayload
		}
		return observabilityv1.EventKind_EVENT_KIND_DOMAIN, nil
	case *observabilityv1.EventEnvelope_Metric:
		if p.Metric == nil {
			return observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED, ErrMissingPayload
		}
		return observabilityv1.EventKind_EVENT_KIND_METRIC, nil
	case *observabilityv1.EventEnvelope_Audit:
		if p.Audit == nil {
			return observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED, ErrMissingPayload
		}
		return observabilityv1.EventKind_EVENT_KIND_AUDIT, nil
	case *observabilityv1.EventEnvelope_Health:
		if p.Health == nil {
			return observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED, ErrMissingPayload
		}
		return observabilityv1.EventKind_EVENT_KIND_HEALTH, nil
	default:
		return observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED, ErrMissingPayload
	}
}

func topicForKind(kind observabilityv1.EventKind, topicMap map[observabilityv1.EventKind]string) (string, error) {
	if kind == observabilityv1.EventKind_EVENT_KIND_UNSPECIFIED {
		return "", ErrUnsupportedKind
	}

	topic := topicMap[kind]
	if topic == "" {
		return "", fmt.Errorf("%w: %s", ErrMissingTopicForKind, kind.String())
	}

	return topic, nil
}

func partitionKey(event *observabilityv1.EventEnvelope) string {
	if event == nil {
		return ""
	}

	if correlationID := event.GetTrace().GetCorrelationId(); correlationID != "" {
		return correlationID
	}

	return event.GetSource().GetService()
}

func uniqueTopics(topicMap map[observabilityv1.EventKind]string) []string {
	seen := map[string]struct{}{}
	topics := make([]string, 0, len(topicMap))
	for _, topic := range topicMap {
		if topic == "" {
			continue
		}
		if _, exists := seen[topic]; exists {
			continue
		}
		seen[topic] = struct{}{}
		topics = append(topics, topic)
	}
	return topics
}
