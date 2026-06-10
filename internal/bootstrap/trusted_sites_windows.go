//go:build windows

package bootstrap

import (
	"fmt"
	"os/exec"
	"strings"
)

// writeCryptoProTrustedSitesRegistry writes the trusted-sites list to
// HKCU\Software\Crypto Pro\CAdESplugin\TrustedSites as a REG_MULTI_SZ value.
// reg.exe encodes the multi-string entries separated by a literal "\0".
func writeCryptoProTrustedSitesRegistry(keyPath, valueName string, sites []string) error {
	data := strings.Join(sites, `\0`)
	cmd := exec.Command("reg.exe", "add", `HKCU\`+keyPath, "/v", valueName, "/t", "REG_MULTI_SZ", "/d", data, "/f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set cryptopro trusted sites: %w: %s", err, string(output))
	}
	return nil
}
