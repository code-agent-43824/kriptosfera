package bootstrap

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

func ReportFatal(summary string, err error) {
	message := summary
	if err != nil {
		message = fmt.Sprintf("%s: %v", summary, err)
	}

	dialogText := summary
	var launcherErr *LauncherError
	if errors.As(err, &launcherErr) && launcherErr.Code != "" {
		dialogText = fmt.Sprintf("%s\n\nКод ошибки: %s", summary, launcherErr.Code)
	}
	if root, rootErr := appRoot(); rootErr == nil {
		logPath := filepath.Join(root, "logs", "launcher.log")
		if logger, logErr := logging.New(logPath); logErr == nil {
			logger.Info("fatal error: %s", message)
			_ = logger.Close()
			dialogText = fmt.Sprintf("%s\n\nПодробности: %s", dialogText, logPath)
		}
	}

	showLauncherErrorDialog("Kriptosfera", dialogText)
}
