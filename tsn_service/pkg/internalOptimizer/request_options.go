package internalOptimizer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"OpenCNC/tsn_service/pkg/internalOptimizer/requestadapter"
)

func buildRequestOptions() (requestadapter.RequestOptions, error) {
	mechanism := os.Getenv("OAAS_MECHANISM")

	// Initial integration supports CBS only.
	if mechanism != "CBS" {
		return requestadapter.RequestOptions{},
			fmt.Errorf("expected OAAS_MECHANISM=CBS, got %q", mechanism)
	}

	selection := requestadapter.MechanismSelection{
		Name: mechanism,
	}

	// Optional request-level CBS constraint.
	// The CBS solver separately requires a valid bound for computation.
	raw := strings.TrimSpace(os.Getenv("OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES"))

	if raw != "" {
		beFrameSize, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || beFrameSize == 0 {
			return requestadapter.RequestOptions{},
				fmt.Errorf(
					"OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES must be a positive integer, got %q",
					raw,
				)
		}

		selection.Constraints = map[string]any{
			"maxBestEffortFrameSizeBytes": beFrameSize,
		}
	}

	return requestadapter.RequestOptions{
		Mechanisms: []requestadapter.MechanismSelection{
			selection,
		},
	}, nil
}
