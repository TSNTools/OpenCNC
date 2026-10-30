package internalOptimizer

import (
	"fmt"
	"os"

	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
)

func buildRequestOptions() (requestadapter.RequestOptions, error) {
	mechanism := os.Getenv("OAAS_MECHANISM")

	// Initial integration supports CBS only.
	if mechanism != "CBS" {
		return requestadapter.RequestOptions{},
			fmt.Errorf("expected OAAS_MECHANISM=CBS, got %q", mechanism)
	}

	return requestadapter.RequestOptions{
		Mechanisms: []requestadapter.MechanismSelection{
			{Name: mechanism},
		},
	}, nil
}
