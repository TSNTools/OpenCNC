package eventhandler

import (
	"context"
	"fmt"
	"time"

	"OpenCNC/common/observability"
	store "OpenCNC/common/store-wrapper"
	storewrapper "OpenCNC/common/store-wrapper"
	observabilityv1 "OpenCNC/common/structures/logging"
	"OpenCNC/common/structures/topology"
	uni "OpenCNC/common/structures/uni"
	configurationHandler "OpenCNC/main_service/pkg/configuration-handler"
	"OpenCNC/tsn_service/pkg/notificationServer"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Take in a configuration request, process it and once a configuration
// has been calculated, return ID of the new configuration.
// Take in a configuration request, process it and once a configuration
// has been calculated, return ID of the new configuration.
func HandleAddStreamEvent(
	ctx context.Context,
	obs *observability.Client,
	configReq *uni.ConfigRequest,
	timeOfReq time.Time,
) (configId string, err error) {
	defer func() {
		if err != nil && obs != nil {
			_ = obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_ERROR,
				"stream",
				"add",
				observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
				"stream",
				"",
				fmt.Sprintf("Failed adding stream: %v", err),
			)
		}
	}()

	// Store requests in k/v store and log the events
	requestIds, err := store.StoreMultipleUniConfRequest(configReq.Requests)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed storing and logging events: %v",
				err,
			))
		}
		return "", err
	}

	if obs != nil {
		_ = obs.Info(ctx, "Configuration requests stored successfully!")
	}

	// Notify TSN service that it should calculate a new configuration
	event := &notificationServer.Event{
		EventId:    "...",
		Type:       notificationServer.EventType_STREAM_ADDED,
		OccurredAt: timestamppb.New(timeOfReq),
		Source:     "some-service",
		Payload: &notificationServer.Event_StreamAdded{
			StreamAdded: &notificationServer.StreamAdded{
				RequestIds: requestIds,
				StreamId:   "...",
			},
		},
	}

	configId, err = notifyTsnService(ctx, obs, event)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed to notify TSN service: %v",
				err,
			))
		}
		return "", err
	}

	tentativeConfiguration, err := storewrapper.GetTopologyConfiguration(configId)
	if err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed getting tentative configuration: %v",
				err,
			))
		}
		return "", err
	}

	if obs != nil {
		_ = obs.Info(ctx, "[Main-service] Tentative configuration retrieved...")
	}

	// Finalize the configuration
	if err = configurationHandler.FinalizeConfiguration(tentativeConfiguration); err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed finalizing configuration: %v",
				err,
			))
		}
		return "", err
	}

	// Send network change to config-service to use new configuration
	if err = notifyConfigService(ctx, obs, configId); err != nil {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed notifying config-service of new configuration: %v",
				err,
			))
		}

		return "", err
	}

	if obs != nil {
		_ = obs.Event(
			ctx,
			observabilityv1.Severity_SEVERITY_INFO,
			"stream",
			"add",
			observabilityv1.DomainResult_DOMAIN_RESULT_SUCCEEDED,
			"stream",
			"",
			"Stream added successfully",
		)
	}

	return configId, nil
}

func RegisterNode(
	ctx context.Context,
	obs *observability.Client,
	node *topology.Node,
) error {
	//HAMZA CHECK!
	if node.GetType() == topology.NodeRole_END_STATION || node.GetType() == topology.NodeRole_BRIDGED_END_STATION {
		if obs != nil {
			_ = obs.Info(ctx, fmt.Sprintf(
				"Registering node %v of type %v",
				node.GetName(),
				node.GetType(),
			))
		}

		if err := store.StoreNode(node); err != nil {
			if obs != nil {
				_ = obs.Event(
					ctx,
					observabilityv1.Severity_SEVERITY_ERROR,
					"node",
					"register",
					observabilityv1.DomainResult_DOMAIN_RESULT_FAILED,
					"node",
					node.GetName(),
					fmt.Sprintf("Failed registering node: %v", err),
				)
			}
			return err
		}

		if obs != nil {
			_ = obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_INFO,
				"node",
				"register",
				observabilityv1.DomainResult_DOMAIN_RESULT_SUCCEEDED,
				"node",
				node.GetName(),
				"Node registered successfully",
			)
		}
	} else {
		if obs != nil {
			_ = obs.Error(ctx, fmt.Sprintf(
				"Failed identifying type of end node: %v",
				node.GetType(),
			))

			_ = obs.Event(
				ctx,
				observabilityv1.Severity_SEVERITY_WARN,
				"node",
				"register",
				observabilityv1.DomainResult_DOMAIN_RESULT_REJECTED,
				"node",
				node.GetName(),
				fmt.Sprintf(
					"Node registration rejected: unsupported node type %v",
					node.GetType(),
				),
			)
		}

		return fmt.Errorf("unsupported node type: %v", node.GetType())
	}

	return nil
}
