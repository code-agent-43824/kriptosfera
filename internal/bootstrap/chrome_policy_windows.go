//go:build windows

package bootstrap

import (
	"fmt"
	"os/exec"
	"strconv"
)

const chromePolicyKeyPath = `Software\Policies\Google\Chrome`

func setChromePolicyDWORD(name string, value int) error {
	cmd := exec.Command("reg.exe", "add", `HKCU\`+chromePolicyKeyPath, "/v", name, "/t", "REG_DWORD", "/d", strconv.Itoa(value), "/f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("set chrome policy %s: %w: %s", name, err, string(output))
	}
	return nil
}
