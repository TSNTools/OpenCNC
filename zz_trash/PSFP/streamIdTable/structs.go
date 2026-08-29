package streamIdTable

/*
Data structures to set the steam identification table
*/
type StreamHandles struct {
	streamHandle            string
	inFacingOutputPortList  []string
	outFacingOutputPortList []string
	inFacingInputPortList   []string
	outFacingInputPortList  []string
	tsnStreamIdType         int
	nullStreamId            NullStreamIdEntry
	sourceMacId             SourceMacIdEntry
	activeDestMacId         ActiveDestMacIdEntry
}

/*
Tagged
in which extent VLAN tag should be used
enum: {tagged, priority, all}
default: all
*/
type NullStreamIdEntry struct {
	destMac string
	tagged  string
	vlan    uint
}

type SourceMacIdEntry struct {
	sourceMac string
	tagged    string
	vlan      uint
}

type ActiveDestMacIdEntry struct {
	upperDestMac  string
	upperTagged   string
	upperVlan     uint
	upperPriority uint
	lowerDestMac  string
	lowerTagged   string
	lowerVlan     uint
}
