package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/code-agent-43824/kriptosfera/internal/config"
	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

// PrepareResult reports where the payload was made available and whether an
// already-prepared copy was reused instead of re-extracted.
type PrepareResult struct {
	AppDir string
	Reused bool
}

// PayloadManager prepares the payload under the per-app directory. It is safe
// against concurrent launches via a bootstrap lock and reuses a previously
// prepared payload when its version and SHA-256 still match.
type PayloadManager struct{}

// Prepare ensures the payload identified by source is extracted and verified
// under root/apps/demo/<version>. It acquires a bootstrap lock, reuses a valid
// existing copy when possible, and otherwise downloads/extracts into a staging
// directory, verifies it against the manifest, and atomically renames it into
// place.
func (m PayloadManager) Prepare(ctx context.Context, source PayloadSource, root string, cfg config.RuntimeConfig, logger *logging.Logger, progress ProgressReporter) (PrepareResult, error) {
	appDir := filepath.Join(root, "apps", "demo", cfg.Version)
	parentDir := filepath.Dir(appDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return PrepareResult{}, err
	}

	// Fast path: if the payload is already prepared, reuse it without taking the
	// bootstrap lock, so a second launch of an already-installed app never
	// contends with (or is blocked by) a concurrent first-run extraction.
	if prepared, err := isPreparedPayload(appDir, source.Version(), source.Mode(), source.ExpectedSHA256()); err != nil {
		return PrepareResult{}, err
	} else if prepared {
		logger.Info("payload already prepared path=%s mode=%s", appDir, source.Mode())
		return PrepareResult{AppDir: appDir, Reused: true}, nil
	}

	unlock, err := acquireLock(appDir)
	if err != nil {
		return PrepareResult{}, err
	}
	defer unlock()

	// Re-check under the lock: another instance may have finished preparing the
	// payload while we were waiting to acquire it.
	if prepared, err := isPreparedPayload(appDir, source.Version(), source.Mode(), source.ExpectedSHA256()); err != nil {
		return PrepareResult{}, err
	} else if prepared {
		logger.Info("payload already prepared path=%s mode=%s", appDir, source.Mode())
		return PrepareResult{AppDir: appDir, Reused: true}, nil
	}

	if progress != nil {
		progress.SetStatus("Подготовка рабочей среды...")
	}
	logger.Info("extract payload path=%s mode=%s", appDir, source.Mode())
	archive, err := source.Open(ctx)
	if err != nil {
		return PrepareResult{}, err
	}
	defer archive.Close()

	tempDir, err := os.MkdirTemp(parentDir, filepath.Base(appDir)+"-staging-")
	if err != nil {
		return PrepareResult{}, err
	}
	defer os.RemoveAll(tempDir)

	if progress != nil {
		progress.SetStatus("Распаковка компонентов...")
	}
	if err := unzipReaderAt(archive.ReaderAt, archive.Size, tempDir); err != nil {
		return PrepareResult{}, wrapLauncherError(ErrPayloadExtractFailed, "не удалось распаковать payload", err)
	}
	if progress != nil {
		progress.SetStatus("Проверка компонентов...")
	}
	if err := verifyExtractedPayload(tempDir); err != nil {
		return PrepareResult{}, wrapLauncherError(ErrPayloadManifestInvalid, "не удалось проверить manifest payload", err)
	}
	if err := writePayloadState(tempDir, PayloadState{
		Version:       source.Version(),
		PayloadMode:   source.Mode(),
		PayloadSHA256: source.ExpectedSHA256(),
	}); err != nil {
		return PrepareResult{}, err
	}
	if err := os.WriteFile(filepath.Join(tempDir, payloadReadyFile), []byte("ok\n"), 0o644); err != nil {
		return PrepareResult{}, err
	}

	if err := os.RemoveAll(appDir); err != nil {
		return PrepareResult{}, err
	}
	if err := os.Rename(tempDir, appDir); err != nil {
		return PrepareResult{}, err
	}
	if progress != nil {
		progress.SetStatus("Запуск приложения...")
	}

	return PrepareResult{AppDir: appDir, Reused: false}, nil
}

func isPreparedPayload(appDir, version, payloadMode, payloadSHA string) (bool, error) {
	if _, err := os.Stat(filepath.Join(appDir, payloadReadyFile)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}

	state, err := loadPayloadState(appDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if state.Version != version || state.PayloadSHA256 != payloadSHA || state.PayloadMode != payloadMode {
		return false, nil
	}

	if err := payloadFilesPresent(appDir); err != nil {
		return false, nil
	}
	return true, nil
}
