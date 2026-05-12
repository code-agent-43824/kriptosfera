//go:build !windows

package bootstrap

import "github.com/code-agent-43824/kriptosfera/internal/logging"

func newPlatformProgressReporter(logger *logging.Logger) ProgressReporter {
	return noopProgressReporter{}
}
