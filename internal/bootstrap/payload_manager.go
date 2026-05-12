package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/code-agent-43824/kriptosfera/internal/config"
	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

type PrepareResult struct {
	AppDir string
	Reused bool
}

type PayloadManager struct{}

func (m PayloadManager) Prepare(ctx context.Context, source PayloadSource, root string, cfg config.RuntimeConfig, logger *logging.Logger) (PrepareResult, error) {
	appDir := filepath.Join(root, "apps", "demo", cfg.Version)
	parentDir := filepath.Dir(appDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return PrepareResult{}, err
	}

	unlock, err := acquireLock(appDir)
	if err != nil {
		return PrepareResult{}, err
	}
	defer unlock()

	if prepared, err := isPreparedPayload(appDir, source.Version(), source.Mode(), source.ExpectedSHA256()); err != nil {
		return PrepareResult{}, err
	} else if prepared {
		logger.Info("payload already prepared path=%s mode=%s", appDir, source.Mode())
		return PrepareResult{AppDir: appDir, Reused: true}, nil
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

	if err := unzipReaderAt(archive.ReaderAt, archive.Size, tempDir); err != nil {
		return PrepareResult{}, wrapLauncherError(ErrPayloadExtractFailed, "не удалось распаковать payload", err)
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

	if err := verifyExtractedPayload(appDir); err != nil {
		return false, nil
	}
	return true, nil
}
