package requestadapter

import (
	"errors"

	optimizationcontract "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

// reads the task identifier from OpenCNC.
func ExtractTaskID(
	task *optimizationcontract.OptimizationTask,
) (string, error) {
	if task == nil || task.GetTaskId() == "" {
		return "", errors.New("missing optimization task ID")
	}
	return task.GetTaskId(), nil
}

// ExtractOptimizationTask maps OpenCNC protobuf data to Contract v1.4.
// Only fields supported by the contract are included.
func ExtractOptimizationTask(
	task *optimizationcontract.OptimizationTask,
) (map[string]any, error) {
	taskID, err := ExtractTaskID(task)
	if err != nil {
		return nil, err
	}

	result := map[string]any{
		"taskId":  taskID,
		"intent":  task.GetIntent().String(),
		"trigger": task.GetTrigger().String(),
	}

	// Scope
	if scope := task.GetScope(); scope != nil {
		mapped := map[string]any{}

		if len(scope.GetStreamIds()) > 0 {
			mapped["streamIds"] = scope.GetStreamIds()
		}
		if len(scope.GetLinkIds()) > 0 {
			mapped["linkIds"] = scope.GetLinkIds()
		}
		if len(scope.GetNodeIds()) > 0 {
			mapped["nodeIds"] = scope.GetNodeIds()
		}

		if len(mapped) > 0 {
			result["scope"] = mapped
		}
	}

	// Objectives
	if len(task.GetObjectives()) > 0 {
		objectives := make([]map[string]any, 0, len(task.GetObjectives()))

		for _, obj := range task.GetObjectives() {
			if obj == nil {
				return nil, errors.New("nil optimization objective")
			}

			metric := obj.GetMetric()
			if metric == optimizationcontract.Metric_METRIC_UNSPECIFIED {
				return nil, errors.New("unspecified optimization metric")
			}
			if _, valid := optimizationcontract.Metric_name[int32(metric)]; !valid {
				return nil, errors.New("unknown optimization metric")
			}
			if obj.GetPriority() < 0 || obj.GetWeight() < 0 {
				return nil, errors.New("negative priority or weight")
			}

			objectives = append(objectives, map[string]any{
				"metric":   metric.String(),
				"priority": obj.GetPriority(),
				"weight":   obj.GetWeight(),
			})
		}

		result["objectives"] = objectives
	}

	// Constraints: OpenCNC's nested Type becomes a string in v1.4.
	if len(task.GetConstraints()) > 0 {
		constraints := make([]map[string]any, 0, len(task.GetConstraints()))

		for _, constraint := range task.GetConstraints() {
			if constraint == nil || constraint.GetType() == nil {
				return nil, errors.New("missing constraint type")
			}

			constraintType := constraint.GetType().GetId()
			if constraintType == optimizationcontract.TypeId_TYPE_UNSPECIFIED {
				return nil, errors.New("unspecified constraint type")
			}
			if _, valid := optimizationcontract.TypeId_name[int32(constraintType)]; !valid {
				return nil, errors.New("unknown constraint type")
			}

			mapped := map[string]any{
				"type":  constraintType.String(),
				"value": constraint.GetValue(),
			}

			if len(constraint.GetIds()) > 0 {
				mapped["ids"] = constraint.GetIds()
			}

			constraints = append(constraints, mapped)
		}

		result["constraints"] = constraints
	}

	// Change policy: explicitly preserve false boolean values.
	if policy := task.GetChangePolicy(); policy != nil {
		mapped := map[string]any{
			"allowPathChanges":        policy.GetAllowPathChanges(),
			"allowScheduleChanges":    policy.GetAllowScheduleChanges(),
			"allowQueueChanges":       policy.GetAllowQueueChanges(),
			"allowVlanChanges":        policy.GetAllowVlanChanges(),
			"allowStreamMigration":    policy.GetAllowStreamMigration(),
			"preserveExistingStreams": policy.GetPreserveExistingStreams(),
		}

		// Preserve current behavior: zero limits are omitted.
		if n := policy.GetMaxChangedStreams(); n > 0 {
			mapped["maxChangedStreams"] = n
		}
		if n := policy.GetMaxConfigurationChanges(); n > 0 {
			mapped["maxConfigurationChanges"] = n
		}

		result["changePolicy"] = mapped
	}

	return result, nil
}
