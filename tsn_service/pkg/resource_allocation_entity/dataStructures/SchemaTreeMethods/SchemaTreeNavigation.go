package SchemaTreeMethods

/*
Help functions to traverse schema trees
*/

import (
	"strconv"
)

// --- FUNCTIONS FOR FINDING FEATURES IN THE BRIDGES
// Get number of components
// NOTE: Unlike many other examples, this method assume that "root" contain "data" element instead of its children
func getNumberofComponentsInBridge(root *SchemaTree, bridgeName string) int64 {
	//lvl1 := OneLvlDownNamespace(root, "data", "urn:ietf:params:xml:ns:netconf:base:1.0")
	lvl2 := OneLvlDownNamespace(root, "bridges", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")
	lvl3 := OneLvlDown1Key(lvl2, "bridge", "name", bridgeName)
	numberOfComponents_str := OneLvlDown0Keys(lvl3, "components").Value
	numberOfComponents_int, _ := strconv.ParseInt(numberOfComponents_str, 10, 32)
	return numberOfComponents_int
}

func getComponentIdsInBridge(root *SchemaTree, bridgeName string) (componentIds []int64) {
	//lvl1 := OneLvlDownNamespace(root, "data", "urn:ietf:params:xml:ns:netconf:base:1.0")
	lvl2 := OneLvlDownNamespace(root, "bridges", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")
	bridge := OneLvlDown1Key(lvl2, "bridge", "name", bridgeName)

	// Find each component in the bridge
	for _, bridge_param := range bridge.Children {
		if bridge_param.Name == "component" {

			// Find the component ID in the component
			component := bridge_param
			for _, component_param := range component.Children {
				if component_param.Name == "id" {
					componentId, _ := strconv.ParseInt(component_param.Value, 10, 32)
					componentIds = append(componentIds, componentId)
				}
			}
		}

	}
	return componentIds
}

func getAllBridgeNames(root *SchemaTree) (bridgeNames []string) {
	//lvl1 := OneLvlDownNamespace(root, "data", "urn:ietf:params:xml:ns:netconf:base:1.0")
	bridges := OneLvlDownNamespace(root, "bridges", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")

	// Find each bridge
	for _, bridge := range bridges.Children {
		for _, param := range bridge.Children {
			if param.Name == "name" {
				bridgeNames = append(bridgeNames, param.Value)
			}

		}

	}
	return bridgeNames
}

// Get number of ports in a bridge
// NOTE: Unlike many other examples, this method assume that "root" contain "data" element instead of its children
func GetNumberofPortsInBridge(root *SchemaTree, bridgeName string) int64 {
	//lvl1 := OneLvlDownNamespace(root, "data", "urn:ietf:params:xml:ns:netconf:base:1.0")
	lvl2 := OneLvlDownNamespace(root, "bridges", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")
	lvl3 := OneLvlDown1Key(lvl2, "bridge", "name", bridgeName)
	numberOfPorts_str := OneLvlDown0Keys(lvl3, "ports").Value
	numberOfPorts_int, _ := strconv.ParseInt(numberOfPorts_str, 10, 32)
	return numberOfPorts_int
}

func GetMstids(root *SchemaTree, bridgeName string, componentId string) (mstids []int32) {
	//lvl1 := OneLvlDownNamespace(root, "data", "urn:ietf:params:xml:ns:netconf:base:1.0")
	lvl2 := OneLvlDownNamespace(root, "bridges", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")
	lvl3 := OneLvlDown1Key(lvl2, "bridge", "name", bridgeName)
	lvl4 := OneLvlDown1Key(lvl3, "component", "id", componentId)
	bridge_mst := OneLvlDown0Keys(lvl4, "bridge-mst")
	for _, mstid_obj := range bridge_mst.Children {
		mstid, _ := strconv.ParseInt(mstid_obj.Value, 10, 32)
		mstids = append(mstids, int32(mstid))
	}
	return mstids
}

func GetBridgesNamespaceSubtree(root *SchemaTree) SchemaTree {
	bridgeSubtree := OneLvlDownNamespace(root, "bridges", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")
	return *bridgeSubtree
}

func GetMSTPNamespaceSubtree(root *SchemaTree) SchemaTree {
	mstpSubtree := OneLvlDownNamespace(root, "ieee8021-mstp", "urn:ietf:params:xml:ns:yang:smiv2:ieee8021-mstp")
	return *mstpSubtree
}

func GetClosestNamespace(elem *SchemaTree) string {
	currentElem := elem
	for {
		if currentElem.Namespace != "" {
			return currentElem.Namespace
		}

		if currentElem.Parent == nil {
			return ""
		}
		currentElem = currentElem.Parent
	}
}
