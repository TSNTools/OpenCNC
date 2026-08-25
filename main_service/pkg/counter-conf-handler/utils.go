package counterconfhandler

import (
	"fmt"
	"os"

	storewrapper "main-service/pkg/store-wrapper"
	monitor "main-service/pkg/structures/temp-monitor-conf"

	//"git.cs.kau.se/hamzchah/opencnc_kafka-exporter/logger/pkg/logger"
	"github.com/ghodss/yaml"
)

//var log = logger.GetLogger()

func genSwitchMonitorConfigFromYAML(sw monitor.MonitorConfig, conf *monitor.Config) {
	// Create at least one interval
	device := monitor.MonitorConfig{
		DeviceIP:   sw.DeviceIP,
		DeviceName: sw.DeviceName,
		Protocol:   sw.Protocol,
		Configs: []*monitor.Conf{
			{
				Counters: []*monitor.IntervalCounters{
					{
						Interval: 10000,
						// Monitor at least one resource
						// Primed with counters for testing
						Counters: []*monitor.DeviceCounter{
							{
								Name: "sw0p1-eth-interface-in-total-frames",
								Path: "elem: <name: 'interfaces' key: <key: 'namespace' value: 'urn:ietf:params:xml:ns:yang:ietf-interfaces'>> elem: <name: 'interface' key: <key:'name' value: 'sw0p1'>>                         elem: <name: 'ethernet' <key: 'namespace' value: 'urn:ieee:std:802.3:yang:ieee802-ethernet-interface'>>                         elem: <name: 'statistics'> elem: <name: 'frame'> elem: <name: 'in-total-frames'>",
							},
							{
								Name: "sw0p2-eth-interface-in-total-frames",
								Path: "elem: <name: 'interfaces' key: <key: 'namespace' value: 'urn:ietf:params:xml:ns:yang:ietf-interfaces'>> elem: <name: 'interface' key: <key:'name' value: 'sw0p1'>>                         elem: <name: 'ethernet' <key: 'namespace' value: 'urn:ieee:std:802.3:yang:ieee802-ethernet-interface'>>                         elem: <name: 'statistics'> elem: <name: 'frame'> elem: <name: 'in-total-octets'>",
							},
							{
								Name: "sw0p3-eth-interface-in-total-frames",
								Path: "elem: <name: 'interfaces' key: <key: 'namespace' value: 'urn:ietf:params:xml:ns:yang:ietf-interfaces'>> elem: <name: 'interface' key: <key:'name' value: 'sw0p1'>>                         elem: <name: 'ethernet' <key: 'namespace' value: 'urn:ieee:std:802.3:yang:ieee802-ethernet-interface'>>                         elem: <name: 'statistics'> elem: <name: 'frame'> elem: <name: 'in-frames'>",
							},
							{
								Name: "sw0p4-eth-interface-in-total-frames",
								Path: "elem: <name: 'interfaces' key: <key: 'namespace' value: 'urn:ietf:params:xml:ns:yang:ietf-interfaces'>> elem: <name: 'interface' key: <key:'name' value: 'sw0p1'>>                         elem: <name: 'ethernet' <key: 'namespace' value: 'urn:ieee:std:802.3:yang:ieee802-ethernet-interface'>>                         elem: <name: 'statistics'> elem: <name: 'frame'> elem: <name: 'in-multicast-frames'>",
							},
						},
					},
				},
			},
		},
	}

	//Converts the map returned from readCounterFile() into appropriate format that fits
	//the struct monitor.MonitorConfig
	for _, config := range readCounterFile() {
		var intervalCounterList []*monitor.IntervalCounters
		for _, intervalCounter := range config.Config {
			var counterList []*monitor.DeviceCounter
			for _, counter := range intervalCounter.Counters {
				counterList = append(counterList, &monitor.DeviceCounter{
					Name: counter.Name,
					Path: counter.Path,
				})
			}
			newIntervalCounter := monitor.IntervalCounters{
				Interval: int32(intervalCounter.Interval),
				Counters: counterList,
			}
			intervalCounterList = append(intervalCounterList, &newIntervalCounter)

		}
		newConfig := monitor.Conf{
			Counters: intervalCounterList,
		}
		device.Configs = append(device.Configs, &newConfig)
	}

	conf.Devices = append(conf.Devices, &device)
}

// Reads the counter-configuration file and returns a list of the struct Counter_conf
func readCounterFile() []Counter_conf {
	fileContent, err := os.ReadFile("counter-monitor-conf.yaml")
	if err != nil {
		//log.Errorf("Failed reading file: %v", err)
		fmt.Println("Failed reading file: %v", err)

		return nil
	}

	yamlConf := make(map[string][]Counter_conf)
	err = yaml.Unmarshal(fileContent, &yamlConf)
	if err != nil {
		//log.Fatalf("error: %v", err)
		fmt.Println("error: %v", err)

	}

	return yamlConf["counter_conf"]
}

// Creates the config for kv store and is then sent to kv store
func CreateMonitorConfig() error {
	switches := GetMonitorConfigDevices()
	// Create empty config file/object
	var conf = &monitor.Config{}

	// Loop over object containing all devices
	for _, sw := range switches {

		// Create at least one config from file "counter-monitor-conf.yaml"
		genSwitchMonitorConfigFromYAML(sw, conf)
		// log.Infof("Added switch: %v", sw)
	}

	for _, sw := range switches {
		storewrapper.StoreMonitorConfig(conf, sw.DeviceIP)
		// log.Infof("Store monitor config for: %v", sw.DeviceIP)
	}

	//log.Info("Created monitoring configuration")
	fmt.Println("Created monitoring configuration")

	return nil
}

func GetMonitorConfigDevices() []monitor.MonitorConfig {
	switchesTopo, err := storewrapper.GetTopology()

	if err != nil {
		fmt.Println(err)
		return nil
	}

	switches := []monitor.MonitorConfig{}

	for _, sw := range switchesTopo.Nodes {
		if sw.Properties != nil && sw.Properties.Bridge != nil {
			switches = append(switches, monitor.MonitorConfig{DeviceIP: sw.ManagementInfo.IpAddress, DeviceName: sw.Name, Protocol: "NETCONF"})
			//TODO: Check for other protocols besides NETCONF
		}
	}

	return switches
}
