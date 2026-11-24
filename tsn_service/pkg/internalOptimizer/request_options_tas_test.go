package internalOptimizer

import (
	"strings"
	"testing"
)

func TestBuildRequestOptionsTAS(t *testing.T) {
	t.Setenv("OAAS_MECHANISM", "TAS")
	t.Setenv("OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES", "")

	options, err := buildRequestOptions()
	if err != nil {
		t.Fatal(err)
	}

	if len(options.Mechanisms) != 1 {
		t.Fatalf("expected one mechanism, got %d", len(options.Mechanisms))
	}

	m := options.Mechanisms[0]
	if m.Name != "TAS" {
		t.Fatalf("expected TAS, got %q", m.Name)
	}

	if len(m.Constraints) != 0 {
		t.Fatalf("TAS must not receive CBS constraints: %v", m.Constraints)
	}
}

func TestBuildRequestOptionsCBSPreserved(t *testing.T) {
	t.Setenv("OAAS_MECHANISM", "CBS")
	t.Setenv("OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES", "1522")

	options, err := buildRequestOptions()
	if err != nil {
		t.Fatal(err)
	}

	m := options.Mechanisms[0]
	if m.Name != "CBS" {
		t.Fatalf("expected CBS, got %q", m.Name)
	}

	if m.Constraints["maxBestEffortFrameSizeBytes"] != uint64(1522) {
		t.Fatalf("CBS constraint was not preserved: %v", m.Constraints)
	}
}

func TestBuildRequestOptionsTASRejectsCBSConstraint(t *testing.T) {
	t.Setenv("OAAS_MECHANISM", "TAS")
	t.Setenv("OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES", "1522")

	_, err := buildRequestOptions()
	if err == nil || !strings.Contains(err.Error(), "CBS-only") {
		t.Fatalf("expected CBS-only constraint rejection, got %v", err)
	}
}
