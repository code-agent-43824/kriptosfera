//go:build windows

package bootstrap

import (
	"fmt"
	"os/exec"
)

func registerCryptoProNativeMessagingHost(manifestPath string) error {
	cmd := exec.Command("reg.exe", "add", `HKCU\`+chromeNativeHostKeyPath, "/ve", "/t", "REG_SZ", "/d", manifestPath, "/f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("register native messaging host: %w: %s", err, string(output))
	}
	return nil
}
