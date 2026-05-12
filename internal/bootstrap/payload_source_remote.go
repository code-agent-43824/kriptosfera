package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/code-agent-43824/kriptosfera/internal/config"
	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

type RemotePayloadSource struct {
	version string
	url     string
	sha256  string
	size    int64
	client  *http.Client
	logger  *logging.Logger
	progressReporter ProgressReporter
	progress func(done, total int64)
}

func NewRemotePayloadSource(cfg config.RuntimeConfig, logger *logging.Logger) (*RemotePayloadSource, error) {
	if cfg.Payload.URL == "" {
		return nil, wrapLauncherError(ErrPayloadNotFound, "не задан URL удалённого payload", fmt.Errorf("payload.url is empty"))
	}
	if cfg.Payload.SHA256 == "" {
		return nil, wrapLauncherError(ErrPayloadNotFound, "не задан checksum удалённого payload", fmt.Errorf("payload.sha256 is empty"))
	}
	return &RemotePayloadSource{
		version: cfg.Payload.Version,
		url: cfg.Payload.URL,
		sha256: cfg.Payload.SHA256,
		size: cfg.Payload.Size,
		logger: logger,
	}, nil
}

func (s *RemotePayloadSource) Mode() string { return config.PayloadModeRemote }

func (s *RemotePayloadSource) Version() string { return s.version }

func (s *RemotePayloadSource) ExpectedSHA256() string { return s.sha256 }

func (s *RemotePayloadSource) Open(ctx context.Context) (PayloadArchive, error) {
	if s.progressReporter != nil {
		s.progressReporter.SetStatus("Подготовка загрузки компонентов...")
	}
	result, err := DownloadFile(ctx, s.client, s.url, s.size, s.logger, s.progress)
	if err != nil {
		return PayloadArchive{}, err
	}
	if result.SHA256 != s.sha256 {
		_ = os.Remove(result.TempPath)
		return PayloadArchive{}, wrapLauncherError(ErrPayloadHashMismatch, "checksum удалённого payload не совпал", fmt.Errorf("expected %s got %s", s.sha256, result.SHA256))
	}
	if s.logger != nil {
		s.logger.Info("payload archive verified sha256=%s", result.SHA256)
	}
	if s.progressReporter != nil {
		s.progressReporter.SetStatus("Проверка и распаковка компонентов...")
	}
	file, err := os.Open(result.TempPath)
	if err != nil {
		_ = os.Remove(result.TempPath)
		return PayloadArchive{}, wrapLauncherError(ErrPayloadDownloadFailed, "не удалось открыть загруженный payload", err)
	}
	closeFn := func() error {
		closeErr := file.Close()
		removeErr := os.Remove(result.TempPath)
		if closeErr != nil {
			return closeErr
		}
		return removeErr
	}
	return PayloadArchive{ReaderAt: file, Size: result.Bytes, Close: closeFn}, nil
}
