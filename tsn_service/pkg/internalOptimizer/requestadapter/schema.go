package requestadapter

import tsncontracts "github.com/Kashaf14/tsn-optimization-contracts"

// SchemaJSON returns the schema from the shared contract package.
func SchemaJSON() []byte {
	return tsncontracts.OptimizerRequestSchemaJSON()
}
