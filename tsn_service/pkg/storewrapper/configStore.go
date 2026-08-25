package storewrapper

import (
	adapterResp "tsn-service/pkg/structures/adapterResponse"

	"google.golang.org/protobuf/proto"
)

// Converts *SchemaTree to adapterResponse and stores in k/v store
func StoreDeviceConfig(ipAddr string, tree *SchemaTree) error {
	var adapterResponse = &adapterResp.AdapterResponse{}

	treeConverter(tree, adapterResponse)

	rawAdapterResponse, err := proto.Marshal(adapterResponse)
	if err != nil {
		//log.Errorf("Failed to marshal adapter response: %v", err)
		return err
	}

	sendToStore(rawAdapterResponse, "configurations."+ipAddr+".config")

	return nil
}

// Converts *SchemaTree to adapterResponse
func treeConverter(tree *SchemaTree, adapterResponse *adapterResp.AdapterResponse) *adapterResp.AdapterResponse {
	start := &adapterResp.SchemaEntry{
		Name:      tree.Name,
		Tag:       "start",
		Namespace: tree.Namespace,
		Value:     tree.Value,
	}

	adapterResponse.Entries = append(adapterResponse.Entries, start)

	// //log.Infof("START %s --- %s --- %v", start.Name, start.Namespace, start.Value)

	for _, child := range tree.Children {
		treeConverter(child, adapterResponse)
	}

	end := &adapterResp.SchemaEntry{
		Name: tree.Name,
		Tag:  "end",
	}

	adapterResponse.Entries = append(adapterResponse.Entries, end)

	// //log.Infof("END %s --- %s --- %v", end.Name, end.Namespace, end.Value)

	return adapterResponse
}

// Takes in an IP address of a switch, gets the data from k/v store, and converts it to *SchemaTree
func GetDeviceConfig(ipAddr string) (*SchemaTree, error) {
	// Create a URN where the config is stored
	urn := "configurations." + ipAddr + ".config"

	rawData, err := getFromStore(urn)
	if err != nil {
		//log.Errorf("Failed getting configuration: %v", err)
		return &SchemaTree{}, err
	}

	var adapterResponse = &adapterResp.AdapterResponse{}
	var schemaTree = &SchemaTree{}

	// Deserialize response that was built in adapter
	if err := proto.Unmarshal(rawData, adapterResponse); err != nil {
		//log.Errorf("Failed to unmarshal ProtoBytes: %v", err)
		return &SchemaTree{}, err
	}

	// Get tree structure of response
	schemaTree = getTreeStructure(adapterResponse.Entries)

	return schemaTree, nil
}

// THE BELOW STRUCTURE SHOULD BE IMPLEMENTED THROUGH PROTOBUF
// OR SHOULD BE REPLACED WITH A NATIVE GNMI IMPLEMENTATION
// The following type are used for deconstructing data from the adapter
type SchemaTree struct {
	Name      string
	Namespace string
	Children  []*SchemaTree
	Parent    *SchemaTree
	Value     string
}

// Build a schemaTree from the entries provided
func getTreeStructure(schemaEntries []*adapterResp.SchemaEntry) *SchemaTree {
	var newTree *SchemaTree
	tree := &SchemaTree{}
	lastNode := ""
	for _, entry := range schemaEntries {
		if entry.Value == "" {
			// In a directory
			if entry.Tag == "end" {
				if entry.Name != "data" {
					if lastNode != "leaf" {
						tree = tree.Parent
					}
					lastNode = ""
				}
			} else {
				newTree = &SchemaTree{Parent: tree}

				newTree.Name = entry.Name
				newTree.Namespace = entry.Namespace
				newTree.Parent.Children = append(newTree.Parent.Children, newTree)

				tree = newTree
			}
		} else {
			// In a leaf
			newTree = &SchemaTree{Parent: tree}

			newTree.Name = entry.Name
			newTree.Value = entry.Value
			newTree.Parent.Children = append(newTree.Parent.Children, newTree)

			lastNode = "leaf"
		}
	}
	return tree
}
