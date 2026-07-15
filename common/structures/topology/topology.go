package topology

import "fmt"

func (topo *Topology) Print() {
	if topo == nil {
		fmt.Println("Topology is empty.")
		return
	}

	fmt.Println("Nodes:")
	for _, node := range topo.Nodes {
		fmt.Printf("	Type: %s\n", node.Type.String()) // Enum string
		fmt.Printf("	Name: %s\n", node.Name)

		// Print bridge properties if present
		if node.Properties != nil && node.Properties.Bridge != nil {
			fmt.Printf("	Processing delay (ns): %d\n", node.Properties.Bridge.ProcessingDelayNs)
		}
		// Print end station properties if present
		if node.Properties != nil && node.Properties.EndStation != nil {
			fmt.Printf("	Application type: %s\n", node.Properties.EndStation.ApplicationType)
			fmt.Printf("	Function: %s\n", node.Properties.EndStation.Function)
		}
		// Print bridged end station properties if present
		if node.Properties != nil && node.Properties.BridgedEndStation != nil {
			fmt.Printf("	Processing delay (ns): %d\n", node.Properties.BridgedEndStation.ProcessingDelayNs)
		}

		fmt.Println("	Ports:")
		for _, port := range node.Ports {
			fmt.Printf("		ID: %s\n", port.Id)
			fmt.Printf("		Name: %s\n", port.Name)
			fmt.Printf("		Speed: %d\n", port.Capabilities.PortSpeed)
			fmt.Printf("		Number of queues: %d\n", port.NumberOfQueues)
			fmt.Println()
		}
		fmt.Println()
	}

	fmt.Println("Links:")
	for _, link := range topo.Links {
		fmt.Printf("	ID: %s\n", link.Id)
		fmt.Printf("	Source node: %s\n", link.SourceNode)
		fmt.Printf("	Target node: %s\n", link.TargetNode)
		fmt.Printf("	Source port: %s\n", link.SourcePort)
		fmt.Printf("	Target port: %s\n", link.TargetPort)
		fmt.Printf("	Propagation delay (ns): %d\n", link.PropagationDelayNs)
		fmt.Printf("	Bandwidth (bps): %d\n", link.Bandwidth)
		fmt.Println()
	}
}
