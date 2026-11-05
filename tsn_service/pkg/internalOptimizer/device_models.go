// to retrieve registered DeviceModels for the nodes in our fpm

package internalOptimizer

import (
	"fmt"

	store "OpenCNC/common/store-wrapper"
	"OpenCNC/common/structures/devicemodelregistry"
	forwarding_plane "OpenCNC/tsn_service/pkg/structures/forwarding_plane"
)

var deviceModelLookup = store.GetDeviceModel

// gets the registered DeviceModel for each node in the fp topology.
func loadNodeDeviceModels(
	fpm *forwarding_plane.ForwardingPlaneModel,
) (map[string]*devicemodelregistry.DeviceModel, error) {

	if fpm == nil || fpm.GetTopology() == nil {
		return nil, fmt.Errorf("forwarding-plane topology is missing")
	}

	models := make(map[string]*devicemodelregistry.DeviceModel)

	for _, node := range fpm.GetTopology().GetNodes() {
		if node == nil {
			return nil, fmt.Errorf("topology contains a nil node")
		}

		modelName := node.GetDeviceInfo().GetDeviceModel()
		if modelName == "" {
			return nil, fmt.Errorf(
				"node %q has no device model",
				node.GetName(),
			)
		}

		model, err := deviceModelLookup(modelName)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to retrieve device model %q for node %q: %w",
				modelName, node.GetName(), err,
			)
		}

		if model == nil {
			return nil, fmt.Errorf(
				"device model %q for node %q was not found",
				modelName, node.GetName(),
			)
		}
		models[node.GetName()] = model
	}

	return models, nil
}
