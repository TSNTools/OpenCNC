package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	observabilityv1 "OpenCNC_config_service/common/structures/logging"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const defaultSchemaVersion = "1.0.0"

type Client struct {
	producer      *Producer
	service       string
	schemaVersion string
	cmdFallback   bool
}

// NewFromEnv builds a simplified observability client from OBS_* environment variables.
// It supports automatic fallback to stdout when publishing is unavailable or fails.
func NewFromEnv(service string) (*Client, error) {
	obsEnabled := parseEnvBool("OBS_ENABLED", false)
	kafkaEnabled := parseEnvBool("OBS_KAFKA_ENABLED", false)
	failOpen := parseEnvBool("OBS_FAIL_OPEN", true)
	brokers := parseCSVEnv("OBS_BROKERS")

	p, err := NewProducer(Config{
		Enabled:               obsEnabled,
		KafkaEnabled:          kafkaEnabled,
		FailOpen:              failOpen,
		Brokers:               brokers,
		RequiredSchemaVersion: defaultSchemaVersion,
	})
	if err != nil {
		if failOpen {
			return &Client{
				producer:      nil,
				service:       service,
				schemaVersion: defaultSchemaVersion,
				cmdFallback:   true,
			}, nil
		}
		return nil, err
	}

	return &Client{
		producer:      p,
		service:       service,
		schemaVersion: defaultSchemaVersion,
		cmdFallback:   true,
	}, nil
}

func (c *Client) Close() error {
	if c == nil || c.producer == nil {
		return nil
	}
	return c.producer.Close()
}

func (c *Client) EmitHealthStarted(ctx context.Context, check string, message string) error {
	return c.emitHealth(ctx, check, observabilityv1.HealthStatus_HEALTH_STATUS_HEALTHY, observabilityv1.Severity_SEVERITY_INFO, message, "")
}

func (c *Client) EmitHealthError(ctx context.Context, check string, message string, correlationID string) error {
	return c.emitHealth(ctx, check, observabilityv1.HealthStatus_HEALTH_STATUS_UNHEALTHY, observabilityv1.Severity_SEVERITY_ERROR, message, correlationID)
}

func (c *Client) emitHealth(ctx context.Context, check string, status observabilityv1.HealthStatus, severity observabilityv1.Severity, message string, correlationID string) error {
	if c == nil {
		return nil
	}

	event := c.newHealthEventEnvelope(check, status, severity, message, correlationID)
	if c.producer == nil || !c.producer.Enabled() {
		if c.cmdFallback {
			fallbackToCMD("producer-disabled", event)
		}
		return nil
	}

	if err := c.producer.Publish(ctx, event); err != nil {
		if c.cmdFallback {
			fallbackToCMD(err.Error(), event)
			return nil
		}
		return err
	}

	return nil
}

func (c *Client) newHealthEventEnvelope(check string, status observabilityv1.HealthStatus, severity observabilityv1.Severity, message string, correlationID string) *observabilityv1.EventEnvelope {
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
		Kind:     observabilityv1.EventKind_EVENT_KIND_HEALTH,
		Severity: severity,
		Payload: &observabilityv1.EventEnvelope_Health{
			Health: &observabilityv1.HealthEvent{
				Check:   check,
				Status:  status,
				Message: message,
			},
		},
	}
}

func fallbackToCMD(reason string, event *observabilityv1.EventEnvelope) {
	marshaler := protojson.MarshalOptions{UseProtoNames: true}
	payload, err := marshaler.Marshal(event)
	if err != nil {
		fmt.Fprintf(os.Stdout, "[OBS-FALLBACK] reason=%s marshal_error=%v\n", reason, err)
		return
	}

	fmt.Fprintf(os.Stdout, "[OBS-FALLBACK] reason=%s event=%s\n", reason, string(payload))
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
