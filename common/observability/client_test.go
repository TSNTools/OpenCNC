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
		cmdMirror:     true,
	}

	if err := c.EmitHealthStarted(context.Background(), "startup", "started"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHealth_NoPanicWithFallback(t *testing.T) {
	c := &Client{
		producer:      nil,
		service:       "config-service",
		schemaVersion: defaultSchemaVersion,
		cmdMirror:     true,
	}

	err := c.Health(
		context.Background(),
		"config-service-serve",
		observabilityv1.HealthStatus_HEALTH_STATUS_UNHEALTHY,
		observabilityv1.Severity_SEVERITY_ERROR,
		"failed",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLog_NoPanicWithFallback(t *testing.T) {
	c := &Client{
		producer:      nil,
		service:       "config-service",
		schemaVersion: defaultSchemaVersion,
		cmdMirror:     true,
	}

	err := c.Log(context.Background(), observabilityv1.Severity_SEVERITY_INFO, "config-service", "hello")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestMetric_NoPanicWithFallback(t *testing.T) {
	c := &Client{
		producer:      nil,
		service:       "config-service",
		schemaVersion: defaultSchemaVersion,
		cmdMirror:     true,
	}

	err := c.Metric(
		context.Background(),
		observabilityv1.Severity_SEVERITY_INFO,
		"grpc.requests",
		observabilityv1.MetricType_METRIC_TYPE_COUNTER,
		1,
		"count",
		map[string]string{"method": "Serve"},
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestEvent_NoPanicWithFallback(t *testing.T) {
	c := &Client{
		producer:      nil,
		service:       "config-service",
		schemaVersion: defaultSchemaVersion,
		cmdMirror:     true,
	}

	err := c.Event(
		context.Background(),
		observabilityv1.Severity_SEVERITY_INFO,
		"config.apply",
		"requested",
		observabilityv1.DomainResult_DOMAIN_RESULT_ACCEPTED,
		"configuration",
		"cfg-1",
		"configuration accepted",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestAudit_NoPanicWithFallback(t *testing.T) {
	c := &Client{
		producer:      nil,
		service:       "config-service",
		schemaVersion: defaultSchemaVersion,
		cmdMirror:     true,
	}

	err := c.Audit(
		context.Background(),
		observabilityv1.Severity_SEVERITY_INFO,
		"system",
		"apply",
		"configuration",
		"cfg-1",
		observabilityv1.AuditResult_AUDIT_RESULT_SUCCEEDED,
		"",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestNewBaseEnvelope_AutofillsCorrelation(t *testing.T) {
	c := &Client{
		service:       "config-service",
		schemaVersion: defaultSchemaVersion,
	}

	event := c.newBaseEnvelope(observabilityv1.EventKind_EVENT_KIND_HEALTH, observabilityv1.Severity_SEVERITY_INFO, "")
	if event.GetTrace().GetCorrelationId() == "" {
		t.Fatalf("expected non-empty correlation id")
	}
	if event.GetEventId() == "" {
		t.Fatalf("expected non-empty event id")
	}
}
