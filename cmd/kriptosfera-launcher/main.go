// Command kriptosfera-launcher is the entrypoint for the Kriptosfera desktop
// launcher. It loads the build-time runtime configuration and delegates the
// full bootstrap-and-launch flow to the bootstrap package, surfacing any fatal
// error to the user via a native dialog before exiting non-zero.
package main

import (
	"os"

	"github.com/code-agent-43824/kriptosfera/internal/bootstrap"
)

func main() {
	cfg, err := bootstrap.DefaultConfig()
	if err != nil {
		bootstrap.ReportFatal("Не удалось подготовить конфигурацию запуска", err)
		os.Exit(1)
	}
	if err := bootstrap.Run(cfg); err != nil {
		bootstrap.ReportFatal("Не удалось запустить приложение", err)
		os.Exit(1)
	}
}
