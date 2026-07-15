package observability

import (
	"context"
	"testing"

	observabilityv1 "OpenCNC_config_service/common/structures/logging"
)

func TestNewFromEnv_DisabledBuildsClient(t *testing.T) {
	t.Setenv("OBS_ENABLED", "false")
	t.Setenv("OBS_KAFKA_ENABLED", "false")
	t.Setenv("OBS_FAIL_OPEN", "true")
	t.Setenv("OBS_BROKERS", "")

	client, err := NewFromEnv("config-service")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if client == nil {
		t.Fatalf("expected client instance")
	}

	if err := client.Close(); err != nil {
		t.Fatalf("expected close without error, got %v", err)
	}
}

func TestNewFromEnv_EnabledWithoutBrokers_FailOpen(t *testing.T) {
	t.Setenv("OBS_ENABLED", "true")
	t.Setenv("OBS_KAFKA_ENABLED", "true")
	t.Setenv("OBS_FAIL_OPEN", "true")
	t.Setenv("OBS_BROKERS", "")

	client, err := NewFromEnv("config-service")
	if err != nil {
		t.Fatalf("expected fail-open to suppress init error, got %v", err)
	}
	if client == nil {
		t.Fatalf("expected client instance")
	}
}

func TestEmitHealthStarted_NoPanicWithFallback(t *testing.T) {
	c := &Client{
		producer:      nil,
		service:       "config-service",
		schemaVersion: defaultSchemaVersion,
		cmdFallback:   true,
	}

	if err := c.EmitHealthStarted(context.Background(), "startup", "started"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNewHealthEventEnvelope_AutofillsCorrelation(t *testing.T) {
	c := &Client{
		service:       "config-service",
		schemaVersion: defaultSchemaVersion,
	}

	event := c.newHealthEventEnvelope("check", observabilityv1.HealthStatus_HEALTH_STATUS_HEALTHY, observabilityv1.Severity_SEVERITY_INFO, "ok", "")
	if event.GetTrace().GetCorrelationId() == "" {
		t.Fatalf("expected non-empty correlation id")
	}
	if event.GetEventId() == "" {
		t.Fatalf("expected non-empty event id")
	}
}
