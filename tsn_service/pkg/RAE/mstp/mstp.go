package mstp

/*
The public functions to update MSTP tables
*/

/*
With the provided invariables, update the configuration of the MSTP port table at the k/v-store adn config-service
*/
func UpdateMstpPortTable(priority int, pathCost int, componentId uint, port uint, mstid uint, deviceIp string) error {

	// (1) Get Configuration from k/v-store
	// TODO: get configuration from k/v-store,

	// (2) Make edits in the relevant table
	// configUpdatesTree, configUpdatesPb, _ := SetMstpPortTable(&root, priority, pathCost, componentId, mstid, port, deviceIp) // (2a) Make edits in the tree structure

	//TODO: (3) Update config service
	// mstp_root := st.GetMSTPNamespaceSubtree(&root) // (3a) Make subtrees where the updates takes place
	//TODO: (3b) Send updates to config service
	//TODO: (3c) Read response from config service

	// (4) Update k/v-store
	// TODO: update the k/v-store, now only prints the result
	return nil
}
