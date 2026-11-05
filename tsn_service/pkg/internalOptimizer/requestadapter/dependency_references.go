// Checks that stream dependencies reference valid streams.
package requestadapter

import "fmt"

// validateDependencyReferences applies the same P4 reference rule as the Manager:
// every dependency target must name exactly one stream in the converted
// forwardingPlane. It does not perform full JSON Schema validation.
func validateDependencyReferences(forwarding map[string]any, groups []DependencyGroup) error {
	if len(groups) == 0 {
		return nil
	}

	streams, ok := forwarding["streams"].([]any)
	if !ok {
		return fmt.Errorf("forwardingPlane.streams: expected array")
	}

	counts := make(map[string]int, len(streams))
	for i, raw := range streams {
		stream, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("forwardingPlane.streams[%d]: expected object", i)
		}
		definition, ok := stream["definition"].(map[string]any)
		if !ok {
			return fmt.Errorf("forwardingPlane.streams[%d]: missing definition", i)
		}
		id, ok := definition["id"].(string)
		if !ok || id == "" {
			return fmt.Errorf("forwardingPlane.streams[%d]: missing stream ID", i)
		}
		counts[id]++
	}

	for groupIndex, group := range groups {
		for targetIndex, target := range group.Target {
			for streamIndex, id := range target {
				path := fmt.Sprintf("dependencyGroups[%d].target[%d][%d]", groupIndex, targetIndex, streamIndex)
				switch counts[id] {
				case 0:
					return fmt.Errorf("%s: unknown stream reference %q (expected a derived StreamKey)", path, id)
				case 1:
					// Resolved unambiguously.
				default:
					return fmt.Errorf("%s: ambiguous duplicate stream ID %q", path, id)
				}
			}
		}
	}
	return nil
}
