package eventhandler

import (
	"fmt"
	store "main-service/pkg/store-wrapper"
	"main-service/pkg/structures/configuration"
	"main-service/pkg/structures/event"
	"main-service/pkg/structures/notification"
	"time"

	"github.com/google/uuid"
)

// Takes in requests, stores them, and logs the events
func storeRequestsInStore(requestList []*configuration.Request, timeOfReq time.Time) (*notification.IdList, error) {
	var storingOk = true
	var err error
	var requestIds []*notification.UUID

	// Store all requests in a k/v store
	for _, request := range requestList {
		// Store request in k/v store and get the ID for the request
		id, err := store.StoreUniConfRequest(request)
		requestIds = append(requestIds, &notification.UUID{
			Value: fmt.Sprintf("%v", id.GetValue()),
		})

		if err != nil {
			storingOk = false
		} else {
			if err = storeEvent(request, timeOfReq); err != nil {
				return nil, err
			}
		}
	}

	// Stop handling event if storing of configurations failed
	if !storingOk {
		//	log.Errorf("Storing configuration requests failed: %v", err)
		fmt.Printf("Storing configuration requests failed: %v", err)

		return nil, err
	}

	var reqIdList = notification.IdList{
		Values: requestIds,
	}

	return &reqIdList, nil
}

// Create and store an event
func storeEvent(req *configuration.Request, timeOfReq time.Time) error {
	// Create an event from the request
	ev, err := createEvent(req, timeOfReq)
	if err != nil {
		//log.Errorf("Failed creating event from request: %v", err)
		fmt.Printf("Failed creating event from request: %v", err)

		return err
	}

	// Log the event
	if err = store.LogEvent(ev); err != nil {
		//	log.Errorf("Failed to log event: %v", err)
		fmt.Printf("Failed to log event: %v", err)

		return err
	}

	return nil
}

// Create an event from the request
func createEvent(req *configuration.Request, timeOfReq time.Time) (*event.Event, error) {
	// TODO: Add correct data to event, still don't know:
	// 		* Event types that should exist
	// 		* What is a Handler
	// 		* Should EventGroupId just be a uuid?
	// 		* OccuranceTime comes from where?
	// 		* Duration is measured where?
	// 		* What is LogInfo?

	var event = event.Event{
		EventId:       fmt.Sprintf("%v", uuid.New()),
		EventType:     event.EventType_ADD_STREAM,
		Status:        event.EventStatus_PASSED,
		Handlers:      []*event.EventHandler{},
		EventGroupId:  fmt.Sprintf("%v", uuid.New()),
		OccurenceTime: timeOfReq.UnixNano(),
		Duration:      123,
		LogInfo:       &event.LogInfo{},
	}

	return &event, nil
}
