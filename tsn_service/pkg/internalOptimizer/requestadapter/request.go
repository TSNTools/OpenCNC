package requestadapter

import (
	"errors"

	optimizationcontract "OpenCNC/tsn_service/pkg/structures/optimization_contract"
)

// ExtractTaskID reads the task identifier from OpenCNC.
func ExtractTaskID(
	task *optimizationcontract.OptimizationTask,
) (string, error) {

	if task == nil || task.GetTaskId() == "" {
		return "", errors.New("missing optimization task ID")
	}

	return task.GetTaskId(), nil
}

type OptimizationTaskJSON struct {
	TaskID       string                 `json:"taskId"`
	Intent       string                 `json:"intent"`
	Trigger      string                 `json:"trigger"`
	Scope        *OptimizationScopeJSON `json:"scope,omitempty"`
	Objectives   []ObjectiveJSON        `json:"objectives,omitempty"`
	Constraints  []ConstraintJSON       `json:"constraints,omitempty"`
	ChangePolicy *ChangePolicyJSON      `json:"changePolicy,omitempty"`
}

func ExtractOptimizationTask(
	task *optimizationcontract.OptimizationTask,
) (OptimizationTaskJSON, error) {

	taskID, err := ExtractTaskID(task)
	if err != nil {
		return OptimizationTaskJSON{}, err
	}

	objectives, err := extractObjectives(task.GetObjectives())
	if err != nil {
		return OptimizationTaskJSON{}, err
	}

	constraints, err := extractConstraints(task.GetConstraints())
	if err != nil {
		return OptimizationTaskJSON{}, err
	}

	return OptimizationTaskJSON{
		TaskID:       taskID,
		Intent:       task.GetIntent().String(),
		Trigger:      task.GetTrigger().String(),
		Scope:        extractScope(task.GetScope()),
		Objectives:   objectives,
		Constraints:  constraints,
		ChangePolicy: extractChangePolicy(task.GetChangePolicy()),
	}, nil
}

type OptimizationScopeJSON struct {
	StreamIDs []string `json:"streamIds,omitempty"`
	LinkIDs   []string `json:"linkIds,omitempty"`
	NodeIDs   []string `json:"nodeIds,omitempty"`
}

func extractScope(
	scope *optimizationcontract.Scope,
) *OptimizationScopeJSON {
	if scope == nil {
		return nil
	}

	if len(scope.GetStreamIds()) == 0 &&
		len(scope.GetLinkIds()) == 0 &&
		len(scope.GetNodeIds()) == 0 {
		return nil
	}

	return &OptimizationScopeJSON{
		StreamIDs: scope.GetStreamIds(),
		LinkIDs:   scope.GetLinkIds(),
		NodeIDs:   scope.GetNodeIds(),
	}
}

type ObjectiveJSON struct {
	Metric   string  `json:"metric"`
	Priority int32   `json:"priority"`
	Weight   float64 `json:"weight"`
}

func extractObjectives(
	objectives []*optimizationcontract.Objective,
) ([]ObjectiveJSON, error) {
	result := make([]ObjectiveJSON, 0, len(objectives))

	for _, obj := range objectives {
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

		result = append(result, ObjectiveJSON{
			Metric:   metric.String(),
			Priority: obj.GetPriority(),
			Weight:   obj.GetWeight(),
		})
	}

	return result, nil
}

type ConstraintJSON struct {
	Type  string   `json:"type"`
	Value float64  `json:"value"`
	IDs   []string `json:"ids,omitempty"`
}

func extractConstraints(
	constraints []*optimizationcontract.Constraint,
) ([]ConstraintJSON, error) {
	result := make([]ConstraintJSON, 0, len(constraints))

	for _, constraint := range constraints {
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

		result = append(result, ConstraintJSON{
			Type:  constraintType.String(),
			Value: constraint.GetValue(),
			IDs:   constraint.GetIds(),
		})
	}

	return result, nil
}

type ChangePolicyJSON struct {
	AllowPathChanges        bool `json:"allowPathChanges"`
	AllowScheduleChanges    bool `json:"allowScheduleChanges"`
	AllowQueueChanges       bool `json:"allowQueueChanges"`
	AllowVlanChanges        bool `json:"allowVlanChanges"`
	AllowStreamMigration    bool `json:"allowStreamMigration"`
	PreserveExistingStreams bool `json:"preserveExistingStreams"`

	MaxChangedStreams       *uint32 `json:"maxChangedStreams,omitempty"`
	MaxConfigurationChanges *uint32 `json:"maxConfigurationChanges,omitempty"`
}

func extractChangePolicy(
	policy *optimizationcontract.ChangePolicy,
) *ChangePolicyJSON {
	if policy == nil {
		return nil
	}

	result := &ChangePolicyJSON{
		AllowPathChanges:        policy.GetAllowPathChanges(),
		AllowScheduleChanges:    policy.GetAllowScheduleChanges(),
		AllowQueueChanges:       policy.GetAllowQueueChanges(),
		AllowVlanChanges:        policy.GetAllowVlanChanges(),
		AllowStreamMigration:    policy.GetAllowStreamMigration(),
		PreserveExistingStreams: policy.GetPreserveExistingStreams(),
	}

	if n := policy.GetMaxChangedStreams(); n > 0 {
		result.MaxChangedStreams = &n
	}

	if n := policy.GetMaxConfigurationChanges(); n > 0 {
		result.MaxConfigurationChanges = &n
	}

	return result
}
