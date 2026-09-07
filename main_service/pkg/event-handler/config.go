package eventhandler

import (
	store "OpenCNC/common/store-wrapper"
	uni "OpenCNC/common/structures/uni"

	"context"
	"fmt"

	"OpenCNC/common/observability"
)

// Takes in requests, stores them, and logs the events
func storeRequestsInStore(
	ctx context.Context,
	obs *observability.Client,
	requestList []*uni.Request,
) ([]string, error) {

	var requestIds []string

	// Store all requests in a k/v store
	for _, request := range requestList {
		// Store request in k/v store and get the ID for the request
		id, err := store.StoreUniConfRequest(request)
		if err != nil {
			if obs != nil {
				_ = obs.Error(ctx, fmt.Sprintf(
					"Storing configuration requests failed: %v",
					err,
				))
			}
			continue
		}

		requestIds = append(requestIds, id)
	}

	return requestIds, nil
}
