package pbMethods

/*
Help functions to get *pb.PathElem for the configuration
Is used to set the values of the gNMI (gRPC Network Managment Interface) configuration updates
*/

import (
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

/*
Get path down to bridge-port: interfaces -> interface at specific port -> bridge-port
*/
func GetPath2bridge(port string) (path []*pb.PathElem) {
	path1lvl := []*pb.PathElem{
		{
			Name: "interfaces",
			Key:  map[string]string{"namespace": "urn:ietf:params:xml:ns:yang:ietf-interfaces"},
		},
	}
	path2lvl := GetPath1lvlDown1Key(path1lvl, "interface", "name", port)
	path = GetPath1lvlDown1Key(path2lvl, "bridge-port", "namespace", "urn:ieee:std:802.1Q:yang:ieee802-dot1q-bridge")

	return path
}

/*
Get path down one level bellow the provided prePath, where no keys are required
*/
func GetPath1lvlDown0Keys(prePath []*pb.PathElem, name1 string) (path []*pb.PathElem) {
	postPath := []*pb.PathElem{
		{
			Name: name1,
			Key:  map[string]string{},
		},
	}
	if prePath == nil {
		path = postPath
	} else {
		path = append(prePath, postPath...)
	}
	return path
}

/*
Get path down one level bellow the provided prePath, where 1 key are required
*/
func GetPath1lvlDown1Key(prePath []*pb.PathElem, name string, leftKey string, rightkey string) (path []*pb.PathElem) {
	postPath := []*pb.PathElem{
		{
			Name: name,
			Key:  map[string]string{leftKey: rightkey},
		},
	}
	if prePath == nil {
		path = postPath
	} else {
		path = append(prePath, postPath...)
	}
	return path
}

/*
Get path down one level bellow the provided prePath, where 2 keys are required
*/
func GetPath1lvlDown2Keys(prePath []*pb.PathElem, name string, leftKey1 string, rightkey1 string, leftKey2 string, rightkey2 string) (path []*pb.PathElem) {
	postPath := []*pb.PathElem{
		{
			Name: name,
			Key:  map[string]string{leftKey1: rightkey1, leftKey2: rightkey2},
		},
	}
	if prePath == nil {
		path = postPath
	} else {
		path = append(prePath, postPath...)
	}
	return path
}

/*
Get path down one level bellow the provided prePath, where 3 keys are required
*/
func GetPath1lvlDown3Keys(prePath []*pb.PathElem, name1 string, leftKey1 string, rightkey1 string, leftKey2 string, rightkey2 string, leftKey3 string, rightKey3 string) (path []*pb.PathElem) {
	postPath := []*pb.PathElem{
		{
			Name: name1,
			Key:  map[string]string{leftKey1: rightkey1, leftKey2: rightkey2, leftKey3: rightKey3},
		},
	}
	if prePath == nil {
		path = postPath
	} else {
		path = append(prePath, postPath...)
	}
	return path
}

/*
Help function to get config update from the path
*/
func GetUpdate(deviceIp string, configPath []*pb.PathElem, configValue pb.TypedValue) (update *pb.Update) {
	update = &pb.Update{
		Path: &pb.Path{
			Elem:   configPath,
			Target: deviceIp,
		},
		Val: &configValue,
	}
	return update
}
