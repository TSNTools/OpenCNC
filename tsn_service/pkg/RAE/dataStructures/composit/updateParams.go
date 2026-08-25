package composit

import (
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	pbMethods "tsn-service/pkg/RAE/dataStructures/pbMethods"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Returns both the pointer to the element "elemName" in a SchemaTree and the path to the element "elemName" as []*pb.PathElem
func GetParam0Keys(preNode *st.SchemaTree, prePath []*pb.PathElem, elemName string) (treePath *st.SchemaTree, pbPath []*pb.PathElem) {
	treePath = st.OneLvlDown0Keys(preNode, elemName)
	pbPath = pbMethods.GetPath1lvlDown0Keys(prePath, elemName)

	return treePath, pbPath
}

// Composit function, returns both the SchemaTree pointer and the []*pb.PathElem path to the "bridge-port"
func GetPath2Bridge(preNode *st.SchemaTree, port string) (treePath *st.SchemaTree, path []*pb.PathElem) {
	treePathLvl1, pbPathLvl1 := GetParamNamespace(preNode, nil, "interfaces", "urn:ietf:params:xml:ns:yang:ietf-interfaces")
	treePathLvl2, pbPathLvl2 := GetParam1Key(treePathLvl1, pbPathLvl1, "", "interface", "name", port)
	treePathLvl3, pbPathLvl3 := GetParamNamespace(treePathLvl2, pbPathLvl2, "bridge-port", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")

	return treePathLvl3, pbPathLvl3
}

// Returns both the SchemaTree pointer and the []*pb.PathElem path to the element "elemName" the a given namespace
func GetParamNamespace(preNode *st.SchemaTree, prePath []*pb.PathElem, elemName string, namespace string) (treePath *st.SchemaTree, pbPath []*pb.PathElem) {
	treePath = st.OneLvlDownNamespace(preNode, elemName, namespace)
	pbPath = pbMethods.GetPath1lvlDown1Key(prePath, elemName, "namespace", namespace)

	return treePath, pbPath
}

// Returns an SchemaTree-pointer and []*pb.PathElem to an element from a table or list with one key and one key-value
// When an element is selected from a table: use tableName to name the table and entryName to name the entry
// When an element is selected from a list: use entryName to name the entry in the list
func GetParam1Key(preNode *st.SchemaTree, prePath []*pb.PathElem, tableName string, entryName string, key1 string, val1 string) (treePath *st.SchemaTree, pbPath []*pb.PathElem) {

	var treeEntry *st.SchemaTree
	var elem string

	if tableName != "" {
		treeEntry = st.OneLvlDown0Keys(preNode, tableName)
		elem = tableName
	} else {
		treeEntry = preNode
		elem = entryName
	}

	treePath = st.OneLvlDown1Key(treeEntry, entryName, key1, val1)
	pbPath = pbMethods.GetPath1lvlDown1Key(prePath, elem, key1, val1)

	return treePath, pbPath
}

// Returns an SchemaTree-pointer and []*pb.PathElem to an element from a table or list with two keys with corresponding key-values
// When an element is selected from a table: use tableName to name the table and entryName to name the entry
// When an element is selected from a list: use entryName to name the entry in the list
func GetParam2Keys(preNode *st.SchemaTree, prePath []*pb.PathElem, tableName string, entryName string, key1 string, val1 string, key2 string, val2 string) (treePath *st.SchemaTree, pbPath []*pb.PathElem) {

	var treeEntry *st.SchemaTree
	var elem string

	if tableName != "" {
		treeEntry = st.OneLvlDown0Keys(preNode, tableName)
		elem = tableName
	} else {
		treeEntry = preNode
		elem = entryName
	}

	treePath = st.OneLvlDown2Keys(treeEntry, entryName, key1, val1, key2, val2)
	pbPath = pbMethods.GetPath1lvlDown2Keys(prePath, elem, key1, val1, key2, val2)

	return treePath, pbPath
}

// Returns an SchemaTree-pointer and []*pb.PathElem to an element from a table or list with three keys with corresponding key-values
// When an element is selected from a table: use tableName to name the table and entryName to name the entry
// When an element is selected from a list: use entryName to name the entry in the list
func GetParam3Keys(preNode *st.SchemaTree, prePath []*pb.PathElem, tableName string, entryName string, key1 string, val1 string, key2 string, val2 string, key3 string, val3 string) (treePath *st.SchemaTree, pbPath []*pb.PathElem) {

	var treeEntry *st.SchemaTree
	var elem string

	if tableName != "" {
		treeEntry = st.OneLvlDown0Keys(preNode, tableName)
		elem = tableName
	} else {
		treeEntry = preNode
		elem = entryName
	}

	treePath = st.OneLvlDown3Keys(treeEntry, entryName, key1, val1, key2, val2, key3, val3)
	pbPath = pbMethods.GetPath1lvlDown3Keys(prePath, elem, key1, val1, key2, val2, key3, val3)

	return treePath, pbPath
}
