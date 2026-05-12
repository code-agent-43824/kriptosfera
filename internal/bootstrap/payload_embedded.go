//go:build !remote

package bootstrap

import _ "embed"

//go:embed payload.zip
var embeddedPayload []byte
