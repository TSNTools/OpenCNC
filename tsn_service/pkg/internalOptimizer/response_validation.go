// The code will retrieve the stored original request and preserved ForwardingPlaneModel, verify that their network context matches, and call the shared v0.2.0 response validator.
package internalOptimizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
	forwardingplane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"

	tsncontracts "github.com/Kashaf14/tsn-optimization-contracts"
)

// validatedOptimizationResult is intentionally private to internalOptimizer.
// The preserved native model is not mutated during retrieval or validation.
// Any subsequent configuration merge must clone Original first.
type validatedOptimizationResult struct {
	Original        *forwardingplane.ForwardingPlaneModel
	OptimizerResult json.RawMessage
	Validation      *tsncontracts.ResponseValidationReport
}

// jsonEquivalent compares JSON values (rather than raw bytes). UseNumber keeps
// integer identities exact when checking the saved snapshot's contract view.
func jsonEquivalent(left, right []byte) bool {
	decode := func(data []byte) (any, error) {
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var value any
		if err := decoder.Decode(&value); err != nil {
			return nil, err
		}
		return value, nil
	}
	a, errA := decode(left)
	b, errB := decode(right)
	return errA == nil && errB == nil && reflect.DeepEqual(a, b)
}

// contractBaselineForJob obtains the original contract-format forwardingPlane
// from the EXACT submitted request, not from a newly built request.
//
// The preserved native protobuf is checked against the request's topology,
// configuration, and stream identities. Differences caused by OaaS-only stream
// options (e.g. decisionVariables or traffic-framing overrides) are not used
// to overwrite the submitted contract context.
//
// This verifies consistency for the fields below. It is NOT a byte-for-byte
// equivalence proof between protobuf and the OaaS contract's entire model.
func contractBaselineForJob(
	requestJSON []byte,
	original *forwardingplane.ForwardingPlaneModel,
) ([]byte, error) {
	if original == nil {
		return nil, fmt.Errorf("preserved native forwarding-plane model is required")
	}
	var request struct {
		SchemaVersion   string          `json:"schemaVersion"`
		ForwardingPlane json.RawMessage `json:"forwardingPlane"`
	}
	if err := json.Unmarshal(requestJSON, &request); err != nil {
		return nil, fmt.Errorf("decode saved original request: %w", err)
	}
	if request.SchemaVersion != "1.4" || len(request.ForwardingPlane) == 0 ||
		bytes.Equal(bytes.TrimSpace(request.ForwardingPlane), []byte("null")) {
		return nil, fmt.Errorf("saved request is missing its v1.4 forwarding-plane context")
	}

	// This is the same deterministic projection used when the original request
	// was constructed. Do not reconstruct the full request or recalculate policy.
	projected, err := requestadapter.ExtractForwardingPlane(
		original, requestadapter.SchemaJSON(),
	)
	if err != nil {
		return nil, fmt.Errorf("project preserved forwarding-plane snapshot: %w", err)
	}
	projectedJSON, err := json.Marshal(projected)
	if err != nil {
		return nil, fmt.Errorf("marshal preserved forwarding-plane projection: %w", err)
	}

	var requestedPlane, savedProjection map[string]json.RawMessage
	if err := json.Unmarshal(request.ForwardingPlane, &requestedPlane); err != nil || requestedPlane == nil {
		return nil, fmt.Errorf("saved request contains invalid forwarding-plane JSON")
	}
	if err := json.Unmarshal(projectedJSON, &savedProjection); err != nil {
		return nil, fmt.Errorf("decode preserved forwarding-plane projection: %w", err)
	}
	for _, field := range []string{"topology", "configuration"} {
		requested, hasRequested := requestedPlane[field]
		preserved, hasPreserved := savedProjection[field]
		if hasRequested != hasPreserved ||
			(hasRequested && !jsonEquivalent(requested, preserved)) {
			return nil, fmt.Errorf("saved request %s differs from preserved native forwarding plane", field)
		}
	}

	// Compare stable identities, not entire stream documents: the original
	// request adapter may add OaaS-only per-stream optimization instructions.
	type streamIdentity struct {
		Definition struct {
			ID       string          `json:"id"`
			StreamID json.RawMessage `json:"streamId"`
		} `json:"definition"`
	}
	getStreamIdentities := func(raw json.RawMessage) (map[string]json.RawMessage, int, error) {
		var streams []streamIdentity
		if err := json.Unmarshal(raw, &streams); err != nil {
			return nil, 0, err
		}
		seen := make(map[string]json.RawMessage, len(streams))
		for _, stream := range streams {
			if stream.Definition.ID == "" || len(stream.Definition.StreamID) == 0 {
				return nil, 0, fmt.Errorf("missing stable stream identity")
			}
			if _, exists := seen[stream.Definition.ID]; exists {
				return nil, 0, fmt.Errorf("duplicate stream identity")
			}
			seen[stream.Definition.ID] = stream.Definition.StreamID
		}
		return seen, len(streams), nil
	}
	originalStreams, countOriginal, err := getStreamIdentities(savedProjection["streams"])
	if err != nil {
		return nil, fmt.Errorf("preserved stream identities: %w", err)
	}
	requestStreams, countRequested, err := getStreamIdentities(requestedPlane["streams"])
	if err != nil {
		return nil, fmt.Errorf("saved request stream identities: %w", err)
	}
	if countOriginal != countRequested {
		return nil, fmt.Errorf("saved request stream count differs from preserved native model")
	}
	for id, savedID := range originalStreams {
		requestedID, found := requestStreams[id]
		if !found || !jsonEquivalent(savedID, requestedID) {
			return nil, fmt.Errorf("stream %q identity differs from preserved native model", id)
		}
	}

	return append([]byte(nil), request.ForwardingPlane...), nil
}

// validateRetrievedOptimizerPayload requires an optimizerResult obtained from
// waitForCompletedManagerResult, which checks the trusted job ID in BOTH
// the status and result envelopes before returning the optimizer payload.
// Passing the same jobID twice to the shared validator is safe only because
// that independent Manager correlation has already succeeded at this boundary.
func validateRetrievedOptimizerPayload(jobID string, optimizerJSON []byte) (*validatedOptimizationResult, error) {
	originalRequestJSON, err := loadJobRequest(jobID)
	if err != nil {
		return nil, fmt.Errorf("response validation original request: %w", err)
	}
	original, err := loadSnapshotForJob(jobID)
	if err != nil {
		return nil, fmt.Errorf("response validation preserved snapshot: %w", err)
	}
	baselineJSON, err := contractBaselineForJob(originalRequestJSON, original)
	if err != nil {
		return nil, fmt.Errorf("response validation baseline: %w", err)
	}

	report, err := tsncontracts.ValidateOptimizerResponseWithContextJSON(
		optimizerJSON, originalRequestJSON, baselineJSON, jobID, jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("shared optimizer response validation: %w", err)
	}
	if report == nil {
		return nil, fmt.Errorf("shared optimizer response validation returned no report")
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(optimizerJSON, &result); err != nil {
		return nil, fmt.Errorf("decode validated optimizer result status: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("optimizer result is not successful (status: %s)", result.Status)
	}
	return &validatedOptimizationResult{
		Original:        original,
		OptimizerResult: append(json.RawMessage(nil), optimizerJSON...),
		Validation:      report,
	}, nil
}

// waitAndValidateOptimizationResult is the only intended entry to Step 4.
// It validates but DOES NOT apply optimizer changes or return an updated FPM.
func waitAndValidateOptimizationResult(ctx context.Context, jobID string) (*validatedOptimizationResult, error) {
	resultJSON, err := waitForCompletedManagerResult(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("retrieve completed Manager result: %w", err)
	}
	return validateRetrievedOptimizerPayload(jobID, resultJSON)
}
