package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	observabilityv1 "OpenCNC_config_service/common/structures/logging"

	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultSchemaVersion = "1.0.0"

type Client struct {
	producer      *Producer
	service       string
	schemaVersion string
	cmdMirror     bool
	FailOpen      bool
}

// NewFromEnv builds a simplified observability client from OBS_* environment variables.
// It supports automatic fallback to stdout when publishing is unavailable or fails.
func NewFromEnv(service string) (*Client, error) {
	obsEnabled := parseEnvBool("OBS_ENABLED", false)
	kafkaEnabled := parseEnvBool("OBS_KAFKA_ENABLED", false)
	failOpen := parseEnvBool("OBS_FAIL_OPEN", true)
	brokers := parseCSVEnv("OBS_BROKERS")
	cmdMirror := parseEnvBool("OBS_CMD_MIRROR", true)

	p, err := NewProducer(Config{
		Enabled:               obsEnabled,
		KafkaEnabled:          kafkaEnabled,
		Brokers:               brokers,
		RequiredSchemaVersion: defaultSchemaVersion,
	})
	if err != nil {
		// Producer can't be created, still returns a usable client
		// and use stdout mirroring as fallback
		if failOpen {
			writeToCMD("producer not available, falling back to stdout", nil)
			return &Client{
				producer:      nil,
				service:       service,
				schemaVersion: defaultSchemaVersion,
				cmdMirror:     true,
				FailOpen:      failOpen,
			}, nil
		}
		return nil, err
	}

	return &Client{
		producer:      p,
		service:       service,
		schemaVersion: defaultSchemaVersion,
		cmdMirror:     cmdMirror,
		FailOpen:      failOpen,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.producer == nil {
		return nil
	}
	return c.producer.Close()
}

func (c *Client) Health(ctx context.Context, check string, status observabilityv1.HealthStatus, severity observabilityv1.Severity, message string) error {
	if c == nil {
		return nil
	}

	event := c.newBaseEnvelope(observabilityv1.EventKind_EVENT_KIND_HEALTH, severity, "")
	event.Payload = &observabilityv1.EventEnvelope_Health{
		Health: &observabilityv1.HealthEvent{
			Check:   check,
			Status:  status,
			Message: message,
		},
	}

	return c.publishEvent(ctx, event)
}

func (c *Client) Log(ctx context.Context, severity observabilityv1.Severity, loggerName string, message string) error {
	if c == nil {
		return nil
	}

	event := c.newBaseEnvelope(observabilityv1.EventKind_EVENT_KIND_LOG, severity, "")
	event.Payload = &observabilityv1.EventEnvelope_Log{
		Log: &observabilityv1.LogEvent{
			Logger:  loggerName,
			Message: message,
		},
	}

	return c.publishEvent(ctx, event)
}

func (c *Client) Event(ctx context.Context, severity observabilityv1.Severity, domain string, action string, result observabilityv1.DomainResult, subjectType string, subjectID string, summary string) error {
	if c == nil {
		return nil
	}

	event := c.newBaseEnvelope(observabilityv1.EventKind_EVENT_KIND_DOMAIN, severity, "")
	event.Payload = &observabilityv1.EventEnvelope_Domain{
		Domain: &observabilityv1.DomainEvent{
			Domain:      domain,
			Action:      action,
			Result:      result,
			SubjectType: subjectType,
			SubjectId:   subjectID,
			Summary:     summary,
		},
	}

	return c.publishEvent(ctx, event)
}

func (c *Client) Audit(ctx context.Context, severity observabilityv1.Severity, actor string, action string, targetType string, targetID string, result observabilityv1.AuditResult, reason string) error {
	if c == nil {
		return nil
	}

	event := c.newBaseEnvelope(observabilityv1.EventKind_EVENT_KIND_AUDIT, severity, "")
	event.Payload = &observabilityv1.EventEnvelope_Audit{
		Audit: &observabilityv1.AuditEvent{
			Actor:      actor,
			Action:     action,
			TargetType: targetType,
			TargetId:   targetID,
			Result:     result,
			Reason:     reason,
		},
	}

	return c.publishEvent(ctx, event)
}

func (c *Client) Metric(ctx context.Context, severity observabilityv1.Severity, name string, metricType observabilityv1.MetricType, value float64, unit string, attributes map[string]string) error {
	if c == nil {
		return nil
	}

	metricValue := value
	if math.IsNaN(metricValue) || math.IsInf(metricValue, 0) {
		metricValue = 0
	}

	event := c.newBaseEnvelope(observabilityv1.EventKind_EVENT_KIND_METRIC, severity, "")
	event.Payload = &observabilityv1.EventEnvelope_Metric{
		Metric: &observabilityv1.MetricEvent{
			Name:       name,
			Type:       metricType,
			Value:      metricValue,
			Unit:       unit,
			Attributes: attributes,
		},
	}

	return c.publishEvent(ctx, event)
}

func (c *Client) EmitHealthStarted(ctx context.Context, check string, message string) error {
	return c.Health(ctx, check, observabilityv1.HealthStatus_HEALTH_STATUS_HEALTHY, observabilityv1.Severity_SEVERITY_INFO, message)
}

func (c *Client) EmitHealthError(ctx context.Context, check string, message string, _ string) error {
	return c.Health(ctx, check, observabilityv1.HealthStatus_HEALTH_STATUS_UNHEALTHY, observabilityv1.Severity_SEVERITY_ERROR, message)
}

func (c *Client) Info(ctx context.Context, message string) error {
	return c.Log(ctx, observabilityv1.Severity_SEVERITY_INFO, c.service, message)
}

func (c *Client) Error(ctx context.Context, message string) error {
	return c.Log(ctx, observabilityv1.Severity_SEVERITY_ERROR, c.service, message)
}

func (c *Client) Printf(format string, args ...any) {
	if c == nil {
		return
	}
	_ = c.Log(context.Background(), observabilityv1.Severity_SEVERITY_INFO, c.service, fmt.Sprintf(format, args...))
}

func (c *Client) Println(args ...any) {
	if c == nil {
		return
	}
	message := strings.TrimSuffix(fmt.Sprintln(args...), "\n")
	_ = c.Log(context.Background(), observabilityv1.Severity_SEVERITY_INFO, c.service, message)
}

func (c *Client) FatalF(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if c != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = c.Log(ctx, observabilityv1.Severity_SEVERITY_CRITICAL, c.service, msg)
	}

	fmt.Fprintf(os.Stderr, "[FATAL] %s\n", msg)
	os.Exit(1)
}

func (c *Client) publishEvent(ctx context.Context, event *observabilityv1.EventEnvelope) error {
	if c == nil || event == nil {
		return nil
	}

	if c.cmdMirror {
		// Mirror every event to stdout regardless of Kafka publishing.
		writeToCMD("mirror-enabled", event)
	}

	if c.producer == nil || !c.producer.Enabled() {
		// Kafka is unavailable or disabled. Write to stdout only if
		if !c.cmdMirror {
			// mirroring hasn't already produced a copy of this event.
			writeToCMD("producer-disabled", event)
		}
		return nil
	}
	err := c.producer.Publish(ctx, event)
	if err != nil {
		// Publishing failed. Log the failure to stdout
		// and suppress the error if FailOpen is true; otherwise return it.
		fmt.Fprintf(os.Stdout, "[OBS-CMD: publish error] %s\n", err)
		if c.FailOpen {
			return nil
		}

		return err
	}

	return nil
}

func (c *Client) newBaseEnvelope(kind observabilityv1.EventKind, severity observabilityv1.Severity, correlationID string) *observabilityv1.EventEnvelope {
	eventID := randomHex(16)
	if strings.TrimSpace(correlationID) == "" {
		correlationID = eventID
	}

	return &observabilityv1.EventEnvelope{
		SchemaVersion: c.schemaVersion,
		EventId:       eventID,
		OccurredAt:    timestamppb.Now(),
		Source: &observabilityv1.Source{
			Service: c.service,
		},
		Trace: &observabilityv1.TraceContext{
			CorrelationId: correlationID,
		},
		Kind:     kind,
		Severity: severity,
	}
}

func writeToCMD(reason string, event *observabilityv1.EventEnvelope) {
	var msg string

	switch p := event.Payload.(type) {
	case *observabilityv1.EventEnvelope_Log:
		msg = p.Log.Message

	case *observabilityv1.EventEnvelope_Health:
		msg = p.Health.Message

	case *observabilityv1.EventEnvelope_Domain:
		msg = p.Domain.Summary

	case *observabilityv1.EventEnvelope_Audit:
		msg = p.Audit.Reason

	case *observabilityv1.EventEnvelope_Metric:
		msg = fmt.Sprintf("%s=%v %s", p.Metric.Name, p.Metric.Value, p.Metric.Unit)

	default:
		msg = "<unknown event>"
	}

	fmt.Fprintf(os.Stdout, "[OBS-CMD: %s] %s\n", reason, msg)
}

func parseEnvBool(key string, defaultValue bool) bool {
	raw, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return defaultValue
	}
}

func parseCSVEnv(key string) []string {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		items = append(items, trimmed)
	}

	return items
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "fallback-event-id"
	}
	return hex.EncodeToString(buf)
}
