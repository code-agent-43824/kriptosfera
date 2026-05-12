//go:build !windows

package bootstrap

import "fmt"

func showLauncherErrorDialog(title, text string) {
	fmt.Printf("%s: %s\n", title, text)
}
