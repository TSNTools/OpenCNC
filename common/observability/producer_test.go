package observability

import (
	"errors"
	"testing"

	observabilityv1 "OpenCNC_config_service/common/structures/logging"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestValidateEnvelope_Success(t *testing.T) {
	event := validLogEnvelope()

	if err := validateEnvelope(event, ""); err != nil {
		t.Fatalf("expected no validation error, got %v", err)
	}
}

func TestValidateEnvelope_MissingRequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		event   *observabilityv1.EventEnvelope
		errWant error
	}{
		{
			name:    "nil envelope",
			event:   nil,
			errWant: ErrNilEnvelope,
		},
		{
			name: "missing schema version",
			event: func() *observabilityv1.EventEnvelope {
				e := validLogEnvelope()
				e.SchemaVersion = ""
				return e
			}(),
			errWant: ErrMissingSchemaVersion,
		},
		{
			name: "missing event id",
			event: func() *observabilityv1.EventEnvelope {
				e := validLogEnvelope()
				e.EventId = ""
				return e
			}(),
			errWant: ErrMissingEventID,
		},
		{
			name: "missing source service",
			event: func() *observabilityv1.EventEnvelope {
				e := validLogEnvelope()
				e.Source.Service = ""
				return e
			}(),
			errWant: ErrMissingSourceService,
		},
		{
			name: "missing correlation id",
			event: func() *observabilityv1.EventEnvelope {
				e := validLogEnvelope()
				e.Trace.CorrelationId = ""
				return e
			}(),
			errWant: ErrMissingCorrelationID,
		},
		{
			name: "missing severity",
			event: func() *observabilityv1.EventEnvelope {
				e := validLogEnvelope()
				e.Severity = observabilityv1.Severity_SEVERITY_UNSPECIFIED
				return e
			}(),
			errWant: ErrMissingSeverity,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateEnvelope(tc.event, "")
			if !errors.Is(err, tc.errWant) {
				t.Fatalf("expected error %v, got %v", tc.errWant, err)
			}
		})
	}
}

func TestValidateEnvelope_PayloadKindMismatch(t *testing.T) {
	event := validLogEnvelope()
	event.Kind = observabilityv1.EventKind_EVENT_KIND_METRIC

	err := validateEnvelope(event, "")
	if !errors.Is(err, ErrPayloadKindMismatch) {
		t.Fatalf("expected payload kind mismatch error, got %v", err)
	}
}

func TestValidateEnvelope_RequiredSchemaVersion(t *testing.T) {
	event := validLogEnvelope()

	err := validateEnvelope(event, "2.0.0")
	if !errors.Is(err, ErrSchemaVersionMismatch) {
		t.Fatalf("expected schema mismatch error, got %v", err)
	}
}

func TestTopicForKind_DefaultMapping(t *testing.T) {
	topic, err := topicForKind(observabilityv1.EventKind_EVENT_KIND_AUDIT, DefaultTopicMap())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if topic != defaultTopicAudit {
		t.Fatalf("expected topic %q, got %q", defaultTopicAudit, topic)
	}
}

func TestPartitionKey_FallbackToService(t *testing.T) {
	event := validLogEnvelope()
	event.Trace.CorrelationId = ""

	key := partitionKey(event)
	if key != event.Source.Service {
		t.Fatalf("expected fallback key %q, got %q", event.Source.Service, key)
	}
}

func TestProducer_DisabledModeNoop(t *testing.T) {
	p, err := NewProducer(Config{})
	if err != nil {
		t.Fatalf("expected no constructor error in disabled mode, got %v", err)
	}

	if err := p.Publish(t.Context(), nil); err != nil {
		t.Fatalf("expected no-op publish error in disabled mode, got %v", err)
	}
}

func TestProducer_EnabledModeRequiresBrokers(t *testing.T) {
	_, err := NewProducer(Config{Enabled: true, KafkaEnabled: true})
	if !errors.Is(err, ErrMissingKafkaBrokers) {
		t.Fatalf("expected missing brokers error, got %v", err)
	}
}

func validLogEnvelope() *observabilityv1.EventEnvelope {
	return &observabilityv1.EventEnvelope{
		SchemaVersion: "1.0.0",
		EventId:       "evt-1",
		OccurredAt:    timestamppb.Now(),
		Source: &observabilityv1.Source{
			Service: "config-service",
		},
		Trace: &observabilityv1.TraceContext{
			CorrelationId: "corr-1",
		},
		Kind:     observabilityv1.EventKind_EVENT_KIND_LOG,
		Severity: observabilityv1.Severity_SEVERITY_INFO,
		Payload: &observabilityv1.EventEnvelope_Log{
			Log: &observabilityv1.LogEvent{
				Logger:  "test",
				Message: "hello",
			},
		},
	}
}
