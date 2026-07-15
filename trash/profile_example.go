package main

/*
profile := &tsn.Profile{
	Id:   "industrial_profile_1",
	Name: "Industrial TSN Profile",

	Type: tsn.ProfileType_INDUSTRIAL,

	Capabilities: &tsn.ProfileCapabilities{
		SupportTas:  true,
		SupportCbs:  true,
		SupportPsfp: true,
	},

	TrafficTypes: TrafficTypes,

 Bind traffic types to queues on sw0p3
	PortMappings: PortMappings,

 VLAN + PCP classification
	VlanMappings: VlanMappings,
}
*/
/*
TrafficTypes := []*tsn.TrafficType{
		{
			Id:           "tc_critical_motion",
			Name:         "Critical Motion",
			Description:  "Time-triggered motion control traffic",
			DeliveryMode: tsn.DeliveryMode_DEADLINE,
			Shaper:       tsn.Shaper_QBV,
			Mtu:          256,
			Properties: &tsn.TrafficTypeProperties{
				IsTimeTriggered:    true,
				RequiresScheduling: true,
				CycleAligned:       true,
				IsPreemptable:      false,
			},
		},
		{
			Id:           "tc_audio_video",
			Name:         "Audio Video",
			Description:  "Streamed AV traffic",
			DeliveryMode: tsn.DeliveryMode_LATENCY,
			Shaper:       tsn.Shaper_QAV,
			Mtu:          1500,
			Properties: &tsn.TrafficTypeProperties{
				IsPeriodic:          true,
				RequiresReservation: true,
				JitterTolerant:      false,
				IsPreemptable:       true,
			},
		},
		{
			Id:           "tc_best_effort",
			Name:         "Best Effort",
			Description:  "Background traffic",
			DeliveryMode: tsn.DeliveryMode_NONE,
			Shaper:       tsn.Shaper_SP,
			Mtu:          1500,
			Properties: &tsn.TrafficTypeProperties{
				JitterTolerant:     true,
				PacketLossTolerant: true,
				DropEligible:       true,
				IsPreemptable:      true,
			},
		},
	}
*/
