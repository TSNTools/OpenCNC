package pcp

/*
Set the tables that are necessary for Priority Code Point (PCP) mapping to traffic classes
	* PCP encoding table
	* PCP decoding table
	* traffic class table
	* priority regeneration table
input (for all methods):
 	port: port the table is at
 	deviceIp: device the port is on
*/

import (
	st "tsn-service/pkg/RAE/dataStructures/SchemaTreeMethods"
	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"

	pb "github.com/openconfig/gnmi/proto/gnmi"
)

// var log = logger.GetLogger()
var PCPTYPES = []string{"8P0D", "7P1D", "6P2D", "5P3D"}

/*
Set default PCP encoding table as described in the Q documentation

Ref: IEEE 802.1Q-2018 6.9.3

key parameters:

	port, deviceIp
*/
func SetDefaultPcpEncodingTable(root *st.SchemaTree, port string, deviceIp string) (updates []*pb.Update, err error) {
	// For every PCP type adn priority level
	for pcpType := range PCPTYPES {
		for i := 0; i <= 7; i += 1 {
			// for both drop eligable = true (j=1) and false (j=0)
			for j := 0; j <= 1; j += 1 {
				priorityPointValue, err := getPriorityPointValue(PCPTYPES[pcpType], i, j == 1)
				if err != nil {
					//log.Errorf("Failed extracting the pcp encoding table: %v", err)
					return nil, err
				}
				pcpEncodingUpdate := setPcpEncodingTableValue(root, port, deviceIp, PCPTYPES[pcpType], i, j == 1, priorityPointValue)
				updates = append(updates, pcpEncodingUpdate)
			}
		}
	}
	return updates, nil
}

/*
Set default PCP decoding table as described in the Q documentation

Ref: IEEE 802.1Q-2018 6.9.3

key parameters:

	port, deviceIp
*/
func SetDefaultPcpDecodingTable(root *st.SchemaTree, port string, deviceIp string) (updates []*pb.Update, err error) {
	// For every PCP type adn priority level
	for pcpType := range PCPTYPES {
		for i := 0; i <= 7; i += 1 {
			priorityValue, dropEligible, err := getPriorityAndDropEligibleValue(PCPTYPES[pcpType], i)
			if err != nil {
				//log.Errorf("Failed extracting the pcp decoding table: %v", err)
				return nil, err
			}
			pcpDecodingUpdate := setPcpDecodingTableValue(root, port, deviceIp, PCPTYPES[pcpType], i, priorityValue, dropEligible)
			updates = append(updates, pcpDecodingUpdate...)
		}
	}
	return updates, nil
}

/*
Set default traffic class table as described in the Q documentation

Ref: IEEE 802.1Q-2018 8.6.6

key parameters:

	port, deviceIp

parameters to set:

	nrTrafficClasses
*/
func SetDefaultTrafficClassTable(root *st.SchemaTree, port string, deviceIp string, nrTrafficClasses int) (
	updates []*pb.Update, err error) {

	var defaultTrafficClasses []int
	defaultTrafficClasses, err = getDefaultTrafficClasses(nrTrafficClasses)

	if err != nil {
		//log.Errorf("Failed getting the number of traffic classes: %v", err)
		return nil, err
	}

	for i := 0; i < nrTrafficClasses; i++ {
		priorityUpdate := setTrafficClassPriorityAtIndex(root, i, defaultTrafficClasses[i], port, deviceIp)
		updates = append(updates, priorityUpdate)
	}

	return updates, nil
}

/*
Set default priority regeneration table as described in the Q documentation

Ref: IEEE 802.1Q-2018 6.9.4

key parameters:

	port, deviceIp

parameters to set:

	nrTrafficClasses
*/
func SetDefaultPriorityRegenerationTable(root *st.SchemaTree, port string, deviceIp string, nrTrafficClasses int) (
	updates []*pb.Update, err error) {

	for i := 0; i < nrTrafficClasses; i++ {
		priorityUpdate := setTrafficClassPriorityAtIndex(root, i, i, port, deviceIp)
		updates = append(updates, priorityUpdate)
	}

	return updates, nil
}
