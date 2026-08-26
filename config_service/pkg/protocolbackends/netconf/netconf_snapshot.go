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

func (s *NetconfSnapshot) Update(feature *plugins.FeatureXML, node *topology.Node) error {

	if feature == nil {
		return fmt.Errorf("feature XML is nil")
	}

	if len(feature.XML) == 0 {
		return fmt.Errorf("feature XML is empty")
	}

	if len(s.XML) == 0 {
		return fmt.Errorf("snapshot XML is empty")
	}

	doc := etree.NewDocument()

	if err := doc.ReadFromBytes(s.XML); err != nil {
		return fmt.Errorf("failed parsing snapshot XML: %w", err)
	}

	interfaces := doc.FindElement("//interfaces")
	if interfaces == nil {
		return fmt.Errorf("snapshot does not contain <interfaces>")
	}

	var interfaceElement *etree.Element

	for _, intf := range interfaces.FindElements("interface") {

		name := intf.FindElement("name")

		if name == nil {
			continue
		}

		if name.Text() == node.Name {
			interfaceElement = intf
			break
		}
	}

	if interfaceElement == nil {
		return fmt.Errorf(
			"interface %q not found in snapshot",
			node.Name,
		)
	}

	// Remove existing feature subtree.
	if existing := interfaceElement.FindElement(feature.Container); existing != nil {
		interfaceElement.RemoveChild(existing)
	}

	// Parse new feature subtree.
	featureDoc := etree.NewDocument()

	if err := featureDoc.ReadFromBytes(feature.XML); err != nil {
		return fmt.Errorf(
			"failed parsing feature XML: %w",
			err,
		)
	}

	if featureDoc.Root() == nil {
		return fmt.Errorf(
			"feature XML has no root element",
		)
	}

	// Add the updated feature subtree.
	interfaceElement.AddChild(
		featureDoc.Root().Copy(),
	)

	// Store updated snapshot.
	doc.Indent(2)

	updatedXML, err := doc.WriteToBytes()
	if err != nil {
		return fmt.Errorf(
			"failed serializing updated snapshot: %w",
			err,
		)
	}

	s.XML = updatedXML

	return nil
}
