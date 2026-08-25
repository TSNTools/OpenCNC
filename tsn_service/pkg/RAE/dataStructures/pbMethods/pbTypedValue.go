package pbMethods

/*
Help functions to get pb.TypedValue for the different supported datatype
Is used to set the values of the gNMI (gRPC Network Management Interface) configuration updates
*/

import (
	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// Return pb.TypedValue for a String
func GetPbStringTypeVal(input string) (value pb.TypedValue) {
	value = pb.TypedValue{
		Value: &pb.TypedValue_StringVal{
			StringVal: input,
		}}
	return value
}

// Return pb.TypedValue for a integer
func GetPbIntTypeVal(input int) (value pb.TypedValue) {
	value = pb.TypedValue{
		Value: &pb.TypedValue_IntVal{
			IntVal: int64(input),
		}}
	return value
}

// Return pb.TypedValue for a unsigned integer
func GetPbUintTypeVal(input uint) (value pb.TypedValue) {
	value = pb.TypedValue{
		Value: &pb.TypedValue_UintVal{
			UintVal: uint64(input),
		}}
	return value
}

// Return pb.TypedValue for a Boolean
func GetPbBoolTypeVal(input bool) (value pb.TypedValue) {
	value = pb.TypedValue{
		Value: &pb.TypedValue_BoolVal{
			BoolVal: input,
		}}
	return value
}

// Return pb.TypedValue for a array of bytes
func GetPbBytesTypeVal(input []byte) (value pb.TypedValue) {
	value = pb.TypedValue{
		Value: &pb.TypedValue_BytesVal{
			BytesVal: input,
		}}
	return value
}
