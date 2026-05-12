package bootstrap

import (
	"fmt"
	"math"

	"github.com/code-agent-43824/kriptosfera/internal/config"
	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

type ProgressReporter interface {
	SetStatus(text string)
	SetDownloadProgress(done, total int64)
	Close() error
}

func newProgressReporter(cfg config.RuntimeConfig, logger *logging.Logger) ProgressReporter {
	if cfg.Payload.Mode != config.PayloadModeRemote {
		return noopProgressReporter{}
	}
	return newPlatformProgressReporter(logger)
}

type noopProgressReporter struct{}

func (noopProgressReporter) SetStatus(string)                 {}
func (noopProgressReporter) SetDownloadProgress(int64, int64) {}
func (noopProgressReporter) Close() error                     { return nil }

func formatDownloadProgress(done, total int64) string {
	if total > 0 {
		percent := int(math.Round((float64(done) / float64(total)) * 100))
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		return fmt.Sprintf("Загрузка компонентов: %d%% (%s из %s)", percent, humanBytes(done), humanBytes(total))
	}
	if done > 0 {
		return fmt.Sprintf("Загрузка компонентов: %s", humanBytes(done))
	}
	return "Загрузка компонентов..."
}

func humanBytes(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(n)
	idx := 0
	for value >= 1024 && idx < len(units)-1 {
		value /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%d %s", n, units[idx])
	}
	return fmt.Sprintf("%.1f %s", value, units[idx])
}
