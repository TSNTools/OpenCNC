package requestadapter

import "fmt"

// enforces OaaS application-ID referential integrity.
// The JSON Schema checks application shapes but cannot establish whether a
// stream's logical endpoints reference an application defined in this request.
func validateApplicationReferences(forwarding map[string]any, applications []map[string]any) error {
	known := make(map[string]bool, len(applications))
	for i, application := range applications {
		id, ok := application["id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("applications[%d]: nonempty application id is required", i)
		}
		if known[id] {
			return fmt.Errorf("applications[%d]: duplicate application id %q", i, id)
		}
		known[id] = true
	}

	streams, ok := forwarding["streams"].([]any)
	if !ok {
		return fmt.Errorf("forwardingPlane.streams: expected array")
	}
	for i, raw := range streams {
		stream, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("streams[%d]: expected object", i)
		}
		definition, ok := stream["definition"].(map[string]any)
		if !ok {
			return fmt.Errorf("streams[%d]: missing definition", i)
		}
		streamID, _ := definition["id"].(string)

		if rawTalker, present := definition["talkerApplicationId"]; present {
			id, ok := rawTalker.(string)
			if !ok || id == "" || !known[id] {
				return fmt.Errorf("stream %s: unknown talker application %q", streamID, rawTalker)
			}
		}
		if rawListeners, present := definition["listenerApplicationIds"]; present {
			// StreamOptions supplies []string, while a decoded JSON array is []any.
			var listeners []string
			switch values := rawListeners.(type) {
			case []string:
				listeners = values
			case []any:
				for _, value := range values {
					id, ok := value.(string)
					if !ok {
						return fmt.Errorf("stream %s: listener application ids must be strings", streamID)
					}
					listeners = append(listeners, id)
				}
			default:
				return fmt.Errorf("stream %s: listenerApplicationIds must be an array", streamID)
			}
			if len(listeners) == 0 {
				return fmt.Errorf("stream %s: listenerApplicationIds cannot be empty", streamID)
			}
			for _, id := range listeners {
				if !known[id] {
					return fmt.Errorf("stream %s: unknown listener application %q", streamID, id)
				}
			}
		}
	}
	return nil
}
