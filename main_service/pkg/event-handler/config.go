package eventhandler

import (
	store "OpenCNC/common/store-wrapper"
	uni "OpenCNC/common/structures/uni"

	"fmt"
)

// Takes in requests, stores them, and logs the events
func storeRequestsInStore(requestList []*uni.Request) ([]string, error) {

	var requestIds []string

	// Store all requests in a k/v store
	for _, request := range requestList {
		// Store request in k/v store and get the ID for the request
		id, err := store.StoreUniConfRequest(request)
		requestIds = append(requestIds, id)
		if err != nil {
			fmt.Printf("Storing configuration requests failed: %v", err)
		}
	}
	return requestIds, nil
}
