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

	// CBS remains supported. TAS is explicitly opt-in.
	if mechanism != "CBS" && mechanism != "TAS" {
		return requestadapter.RequestOptions{},
			fmt.Errorf("expected OAAS_MECHANISM=CBS or TAS, got %q", mechanism)
	}

	selection := requestadapter.MechanismSelection{
		Name: mechanism,
	}

	// This constraint belongs only to CBS.
	raw := strings.TrimSpace(os.Getenv("OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES"))

	if raw != "" {
		if mechanism != "CBS" {
			return requestadapter.RequestOptions{},
				fmt.Errorf("OAAS_CBS_MAX_BE_FRAME_SIZE_BYTES is CBS-only; unset it for TAS")
		}

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
