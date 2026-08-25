package SchemaTreeMethods

/*
Help functions to generate schema trees, used to update the configuration at the k/v-store
*/

import "errors"

// The data structure of the schema trees
type SchemaTree struct {
	Name      string
	Namespace string
	Children  []*SchemaTree
	Parent    *SchemaTree
	Value     string
}

// Go down 1 level down the schema tree only after the name
func OneLvlDown0Keys(root *SchemaTree, name string) *SchemaTree {
	for _, param := range root.Children {
		if param.Name == name {
			return param
		}
	}
	return &SchemaTree{"", "", nil, nil, ""}
}

// Return all children with specific name from the schema tree
func OneLvlDownAllInstances(root *SchemaTree, name string) (instances []*SchemaTree) {
	for _, param := range root.Children {
		if param.Name == name {
			instances = append(instances, param)
		}
	}
	return instances
}

// Go down 1 level down the schema tree only the name and one set of keys
func OneLvlDown1Key(root *SchemaTree, name string, key string, val string) *SchemaTree {
	for _, entry := range root.Children {
		if entry.Name == name {
			for _, param := range entry.Children {
				if param.Name == key && param.Value == val {
					return entry
				} else if param.Name == key && param.Value != val {
					break
				}
			}
		}
	}
	return &SchemaTree{"", "", nil, nil, ""}
}

// Go down 1 level down the schema tree after the name and namespace
func OneLvlDownNamespace(root *SchemaTree, name string, namespace string) *SchemaTree {
	for _, entry := range root.Children {
		if entry.Name == name && entry.Namespace == namespace {
			return entry
		}
	}
	return &SchemaTree{"", "", nil, nil, ""}
}

// Go down 1 level down the schema tree after the name and 2 set of keys
func OneLvlDown2Keys(root *SchemaTree, name string, key1 string, val1 string, key2 string, val2 string) *SchemaTree {

	for _, entry := range root.Children {
		if entry.Name == name {
			keysFound := 0
			for _, param := range entry.Children {
				if param.Name == key1 && param.Value == val1 {
					keysFound += 1
				} else if param.Name == key1 && param.Value != val1 {
					break
				}

				if param.Name == key2 && param.Value == val2 {
					keysFound += 1
				} else if param.Name == key2 && param.Value != val2 {
					break
				}
			}
			if keysFound == 2 {
				return entry
			}
		}
	}
	return &SchemaTree{"", "", nil, nil, ""}
}

// Go down 1 level down the schema tree after the name and 3 set of keys
func OneLvlDown3Keys(root *SchemaTree, name string, key1 string, val1 string, key2 string, val2 string, key3 string, val3 string) *SchemaTree {

	for _, entry := range root.Children {
		if entry.Name == name {
			keysFound := 0
			for _, param := range entry.Children {

				if param.Name == key1 && param.Value == val1 {
					keysFound += 1
				} else if param.Name == key1 && param.Value != val1 {
					break
				}

				if param.Name == key2 && param.Value == val2 {
					keysFound += 1
				} else if param.Name == key2 && param.Value != val2 {
					break
				}

				if param.Name == key3 && param.Value == val3 {
					keysFound += 1
				} else if param.Name == key3 && param.Value != val3 {
					break
				}
			}
			if keysFound == 3 {
				return entry
			}
		}
	}
	return &SchemaTree{"", "", nil, nil, ""}
}

// Traverse down the schema tree to the bridge-port
func LvlsDownToBridgePort(root *SchemaTree, port string) *SchemaTree {
	lvl1 := OneLvlDown0Keys(root, "interfaces") // HERE IT SHOULD BE A NAMESPACE
	lvl2 := OneLvlDown1Key(lvl1, "interface", "name", port)
	return OneLvlDown0Keys(lvl2, "bridge-port")
}

// Traverse down the schema tree to the bridge-ports
func LvlsDownToBridgePorts(root *SchemaTree) (bridgePorts []*SchemaTree) {
	lvl1 := OneLvlDown0Keys(root, "interfaces") // HERE IT SHOULD BE A NAMESPACE
	ports := GetAllKeyValues(lvl1, "interface", "name")
	for _, port := range ports {
		lvl2 := OneLvlDown1Key(lvl1, "interface", "name", port)
		bridgePorts = append(bridgePorts, OneLvlDown0Keys(lvl2, "bridge-port"))
	}
	return bridgePorts
}

func GetNamespaceRoot(elem *SchemaTree) *SchemaTree {
	currentElement := elem
	for {
		if currentElement.Namespace != "" {
			return currentElement
		}

		if currentElement.Parent == nil {
			return &SchemaTree{"", "", nil, nil, ""}
		}
	}
}

func GetNamespaceRootWithName(elem *SchemaTree, nameSpace string) *SchemaTree {
	currentElement := elem
	for {
		if currentElement.Namespace == nameSpace {
			return currentElement
		}

		if currentElement.Parent == nil {
			return &SchemaTree{"", "", nil, nil, ""}
		}
	}
}

func GetAllKeyValues(root *SchemaTree, name string, key string) (values []string) {
	for _, entry := range root.Children {
		if entry.Name == name {
			for _, param := range entry.Children {
				if param.Name == key {
					values = append(values, param.Value)
					break
				}
			}
		}
	}
	return values
}

func GetKeyValueInParent(obj *SchemaTree, key string) (string, error) {
	var currentObj = obj
	for {
		if currentObj.Name == key {
			return currentObj.Value, nil
		}

		if currentObj.Parent == nil {
			break
		}
	}
	return "", errors.New("did not find the key: " + key)

}

func GetKeyValuesInParent(obj *SchemaTree, keys ...string) (output []string, err error) {
	var currentObj = obj
	var keyPos = 0
	for {
		if currentObj.Name == keys[keyPos] {
			output = append(output, currentObj.Value)
			keyPos += 1
		}

		if currentObj.Parent == nil {
			break
		}
	}

	if keyPos+1 < len(keys) {
		return nil, errors.New("did not find all the keys")

	}
	return output, nil

}

func HasParameter(root *SchemaTree, parameter string) bool {
	for _, child := range root.Children {
		if child.Name == parameter {
			return true
		}
	}
	return false

}
