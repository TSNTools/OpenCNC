package devicemodelregistry

import (
	"fmt"
)

// Define the ItemsList struct in Go, which contains a list of Item structs
type DeviceModelRegistry struct {
	DeviceModels []*DeviceModel
}

func (registry *DeviceModelRegistry) Print() {
	fmt.Println("\nDeviceModelRegistry contains:")
	for _, deviceModel := range registry.DeviceModels {
		deviceModel.Print()
	}
}

func (deviceModel *DeviceModel) Print() {

	// Print the name of the device model
	fmt.Printf("\nDeviceModel: %s\n", deviceModel.Name)
	// Print each YangFile associated with this DeviceModel
	for _, yangFile := range deviceModel.YangFiles {
		fmt.Printf("  - YangFile: %s\n", yangFile.Name)
		fmt.Printf("  	- Revision: %s\n", yangFile.Revision)
		//fmt.Printf("  	- Structure: %s\n", yangFile.Description)
	}
}
