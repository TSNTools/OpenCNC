package requestadapter

import _ "embed"

//go:embed optimizer-request.schema.json
var optimizerRequestSchema []byte

func SchemaJSON() []byte {
	return append([]byte(nil), optimizerRequestSchema...)
}
