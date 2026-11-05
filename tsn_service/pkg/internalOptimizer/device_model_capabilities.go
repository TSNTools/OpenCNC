// It identifies candidate GCL-length limitations using the protobuf's YANG target path, parameter, target kind, comparison and threshold. It preserves the original path for later port association.
package internalOptimizer

import (
	"fmt"
	"math"
	"strings"

	"OpenCNC/common/structures/devicemodelregistry"
)

// A recognized TAS GCL limit, retaining its original YANG path.
// Port assignment is intentionally deferred.
type tasGCLLimit struct {
	TargetPath string
	MaxEntries uint32
}

func extractTASGCLLimits(
	model *devicemodelregistry.DeviceModel,
) ([]tasGCLLimit, error) {

	var result []tasGCLLimit

	if model == nil {
		return nil, fmt.Errorf("DeviceModel is nil")
	}

	for _, limit := range model.GetLimitations() {
		if limit == nil {
			continue
		}

		path := limit.GetTargetPath()
		parameter := limit.GetParameter()

		if parameter == nil {
			continue
		}

		// Identify the YANG gate-control-entry list.
		// This is a candidate mapping, pending exact XPath confirmation.
		if !strings.Contains(path, "admin-control-list") ||
			!strings.HasSuffix(path, "gate-control-entry") {
			continue
		}

		name := strings.ToLower(strings.TrimSpace(parameter.GetName()))
		if name != "length" && name != "max gcl entries" {
			continue
		}

		if parameter.GetTargetKind() != devicemodelregistry.TargetKind_YANG_NODE_KIND_LIST {
			continue
		}

		if unit := parameter.GetUnit(); unit != "" && unit != "entries" {
			return nil, fmt.Errorf("unexpected GCL limit unit %q", unit)
		}

		if limit.GetComparison() != devicemodelregistry.LimitationComparison_LIMITATION_LESS_OR_EQUAL {
			continue
		}

		value := limit.GetThreshold()
		if math.IsNaN(value) || math.IsInf(value, 0) ||
			value < 1 || value > float64(^uint32(0)) ||
			math.Trunc(value) != value {
			return nil, fmt.Errorf("invalid GCL threshold %v", value)
		}

		result = append(result, tasGCLLimit{
			TargetPath: path,
			MaxEntries: uint32(value),
		})
	}

	return result, nil
}

// This associates limitations with their devices, not individual ports. We still won't guess port assignments or add those values to the submitted JSON.
func collectTASGCLLimitsByNode(
	models map[string]*devicemodelregistry.DeviceModel,
) (map[string][]tasGCLLimit, error) {

	result := make(map[string][]tasGCLLimit)

	for nodeID, model := range models {
		limits, err := extractTASGCLLimits(model)
		if err != nil {
			return nil, fmt.Errorf(
				"node %q: %w", nodeID, err,
			)
		}

		if len(limits) > 0 {
			result[nodeID] = limits
		}
	}

	return result, nil
}
