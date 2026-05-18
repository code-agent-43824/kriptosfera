//go:build windows

package bootstrap

import _ "embed"

//go:embed cryptopro-plugin.zip
var embeddedCryptoProPlugin []byte
