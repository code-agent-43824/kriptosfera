package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

type DownloadResult struct {
	TempPath string
	Bytes    int64
	SHA256   string
}

var defaultDownloadClient = &http.Client{Timeout: 5 * time.Minute}

func DownloadFile(ctx context.Context, client *http.Client, url string, expectedSize int64, logger *logging.Logger, progress func(done, total int64)) (DownloadResult, error) {
	if !strings.HasPrefix(strings.ToLower(url), "https://") {
		return DownloadResult{}, wrapLauncherError(ErrPayloadDownloadFailed, "payload URL must use HTTPS", fmt.Errorf("unsupported payload URL: %s", url))
	}
	if client == nil {
		client = defaultDownloadClient
	}

	tempFile, err := os.CreateTemp("", "kriptosfera-payload-*.zip")
	if err != nil {
		return DownloadResult{}, wrapLauncherError(ErrPayloadDownloadFailed, "не удалось подготовить временный файл загрузки", err)
	}
	tempPath := tempFile.Name()
	cleanup := func() { _ = os.Remove(tempPath) }
	defer func() {
		_ = tempFile.Close()
	}()

	if logger != nil {
		logger.Info("download start url=%s expected_size=%d", url, expectedSize)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cleanup()
		return DownloadResult{}, wrapLauncherError(ErrPayloadDownloadFailed, "не удалось подготовить запрос загрузки", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		cleanup()
		return DownloadResult{}, wrapLauncherError(ErrPayloadDownloadFailed, "не удалось загрузить компоненты приложения", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		cleanup()
		return DownloadResult{}, wrapLauncherError(ErrPayloadDownloadFailed, "сервер вернул ошибку при загрузке payload", fmt.Errorf("unexpected status: %s", resp.Status))
	}

	total := resp.ContentLength
	if expectedSize > 0 {
		total = expectedSize
	}

	hash := sha256.New()
	writer := io.MultiWriter(tempFile, hash)
	buf := make([]byte, 64*1024)
	var written int64
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			chunk := buf[:n]
			m, writeErr := writer.Write(chunk)
			written += int64(m)
			if writeErr != nil {
				cleanup()
				return DownloadResult{}, wrapLauncherError(ErrPayloadDownloadFailed, "не удалось сохранить загруженный payload", writeErr)
			}
			if progress != nil {
				progress(written, total)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			cleanup()
			return DownloadResult{}, wrapLauncherError(ErrPayloadDownloadFailed, "загрузка payload оборвалась", readErr)
		}
	}

	if err := tempFile.Close(); err != nil {
		cleanup()
		return DownloadResult{}, wrapLauncherError(ErrPayloadDownloadFailed, "не удалось завершить сохранение payload", err)
	}

	sha := hex.EncodeToString(hash.Sum(nil))
	if logger != nil {
		logger.Info("download complete bytes=%d sha256=%s", written, sha)
	}
	return DownloadResult{TempPath: tempPath, Bytes: written, SHA256: sha}, nil
}
