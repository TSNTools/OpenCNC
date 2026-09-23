package netconf

import (
	"fmt"

	"OpenCNC/common/structures/topology"
	"OpenCNC/config_service/pkg/plugins"
	protocolbackends "OpenCNC/config_service/pkg/protocolbackends"

	"github.com/beevik/etree"
)

type NetconfSnapshot struct {
	XML []byte // parsed model, cached payload, metadata...
}

func (s *NetconfSnapshot) Clone() protocolbackends.Snapshot {
	//todo: check it, this is a placeholder so far
	return &NetconfSnapshot{
		XML: append([]byte(nil), s.XML...),
	}
}

func (s *NetconfSnapshot) Update(feature *plugins.FeatureXML, node *topology.Node, portId string) error {
	if feature == nil {
		return fmt.Errorf("feature XML is nil")
	}
	if len(feature.XML) == 0 {
		return fmt.Errorf("feature XML is empty")
	}
	if len(s.XML) == 0 {
		return fmt.Errorf("snapshot XML is empty")
	}

	// 1. Parse base snapshot document
	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(s.XML); err != nil {
		return fmt.Errorf("failed parsing snapshot XML: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return fmt.Errorf("snapshot XML has no root element")
	}

	// 2. Parse incoming feature XML
	featureDoc := etree.NewDocument()
	if err := featureDoc.ReadFromBytes(feature.XML); err != nil {
		return fmt.Errorf("failed parsing feature XML: %w", err)
	}

	featureRoot := featureDoc.Root()
	if featureRoot == nil {
		return fmt.Errorf("feature XML has no root element")
	}

	// 3. Branch based on target scope
	if portId == "" {
		// ─────────────────────────────────────────────────────────────
		// CASE 1: Device/Bridge level update (e.g. <bridges>)
		// ─────────────────────────────────────────────────────────────
		if existing := root.FindElement(feature.Container); existing != nil {
			root.RemoveChild(existing)
		} else if existing := doc.FindElement("//" + feature.Container); existing != nil {
			if parent := existing.Parent(); parent != nil {
				parent.RemoveChild(existing)
			}
		}

		// Attach directly inside root (<config>)
		root.AddChild(featureRoot.Copy())
	} else {
		// ─────────────────────────────────────────────────────────────
		// CASE 2: Interface level update (e.g. <bridge-port>)
		// ─────────────────────────────────────────────────────────────
		interfaces := doc.FindElement("//interfaces")
		if interfaces == nil {
			return fmt.Errorf("snapshot does not contain <interfaces>")
		}

		// Resolve the device's physical port name using node.Ports if available.
		// NETCONF interface lists are keyed by interface name (e.g. "PORT_0"),
		// whereas portId might be an internal topology ID (e.g. "P0").
		portName := portId
		if node != nil {
			for _, p := range node.Ports {
				if p.Id == portId {
					if p.Name != "" {
						portName = p.Name
					}
					break
				}
			}
		}

		var interfaceElement *etree.Element
		for _, intf := range interfaces.FindElements("interface") {
			name := intf.FindElement("name")
			if name == nil {
				continue
			}
			if name.Text() == portName || name.Text() == portId {
				interfaceElement = intf
				break
			}
		}

		if interfaceElement == nil {
			return fmt.Errorf("interface %q (portId %q) not found in snapshot", portName, portId)
		}

		// Handle feature insertion / update
		if feature.Container == "bridge-port" {
			// Merge incoming bridge-port child elements (VLAN, PCP) into existing <bridge-port>
			bp := interfaceElement.FindElement("bridge-port")
			if bp == nil {
				bp = interfaceElement.FindElement(".//bridge-port")
			}
			if bp == nil {
				interfaceElement.AddChild(featureRoot.Copy())
			} else {
				for _, child := range featureRoot.ChildElements() {
					for _, old := range bp.FindElements(child.Tag) {
						bp.RemoveChild(old)
					}
					bp.AddChild(child.Copy())
				}
			}
		} else {
			// Subtree features like <gate-parameter-table>
			var targetParent *etree.Element = interfaceElement
			if existing := interfaceElement.FindElement(".//" + feature.Container); existing != nil {
				if p := existing.Parent(); p != nil {
					targetParent = p
				}
				targetParent.RemoveChild(existing)
			} else if bp := interfaceElement.FindElement(".//bridge-port"); bp != nil {
				// On Relyum (2021), gate-parameter-table belongs inside <bridge-port>
				targetParent = bp
			}
			targetParent.AddChild(featureRoot.Copy())
		}
	}

	// 4. Re-serialize and update snapshot state
	doc.Indent(2)
	updatedXML, err := doc.WriteToBytes()
	if err != nil {
		return fmt.Errorf("failed serializing updated snapshot: %w", err)
	}

	s.XML = updatedXML
	return nil
}

// PayloadXML extracts the child elements inside <config> (<interfaces>, <bridges>, etc.)
// into a clean XML string suitable for NETCONF edit-config.
func (s *NetconfSnapshot) PayloadXML() (string, error) {
	if len(s.XML) == 0 {
		return "", fmt.Errorf("snapshot XML is empty")
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(s.XML); err != nil {
		return "", fmt.Errorf("failed parsing snapshot XML: %w", err)
	}

	root := doc.Root()
	if root == nil {
		return "", fmt.Errorf("snapshot has no root element")
	}

	// If root is <config>, extract all top-level child elements
	if root.Tag == "config" {
		childDoc := etree.NewDocument()
		for _, child := range root.ChildElements() {
			childDoc.AddChild(child.Copy())
		}
		childDoc.Indent(2)
		return childDoc.WriteToString()
	}

	return string(s.XML), nil
}
