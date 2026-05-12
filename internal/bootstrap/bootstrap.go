package bootstrap

import (
	"context"
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/code-agent-43824/kriptosfera/internal/config"
	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

const (
    payloadStateFile = ".payload-state.json"
    payloadReadyFile = ".payload-ready"
    payloadLockFile  = ".bootstrap.lock"
    payloadManifest  = "manifest.json"
    lockTTL          = 10 * time.Minute
)

type PayloadManifest struct {
    Version string             `json:"version"`
    Files   []PayloadManifestFile `json:"files"`
}

type PayloadManifestFile struct {
    Path   string `json:"path"`
    SHA256 string `json:"sha256"`
}

type PayloadState struct {
    Version       string `json:"version"`
    PayloadMode   string `json:"payloadMode"`
    PayloadSHA256 string `json:"payloadSha256"`
}

func DefaultConfig() (config.RuntimeConfig, error) {
	return config.DefaultRuntimeConfig()
}

func Run(cfg config.RuntimeConfig) error {
    root, err := appRoot()
    if err != nil {
        return err
    }
    logPath := filepath.Join(root, "logs", "launcher.log")
    logger, err := logging.New(logPath)
    if err != nil {
        return err
    }
	defer logger.Close()
	logger.Info("launcher start version=%s os=%s mode=%s", cfg.Version, runtime.GOOS, cfg.Payload.Mode)
	progress := newProgressReporter(cfg, logger)
	defer progress.Close()

	source, err := newPayloadSource(cfg, logger, progress)
	if err != nil {
		return err
	}
	manager := PayloadManager{}
	prepareResult, err := manager.Prepare(context.Background(), source, root, cfg, logger, progress)
    if err != nil {
        return err
    }
    appDir := prepareResult.AppDir

    appCfg, err := config.Load(filepath.Join(appDir, "config", "app-config.json"))
    if err != nil {
        return err
    }
    logger.Info("payload ready reused=%t start_url=%s", prepareResult.Reused, appCfg.StartURL)

    profileDir := filepath.Join(root, "profiles", appCfg.ProfileName)
    if err := os.MkdirAll(profileDir, 0o755); err != nil {
        return err
    }

    chromeDir := filepath.Join(appDir, "chromium")
    if runtime.GOOS != "windows" {
        logger.Info("non-windows environment detected; stub launch only")
        return writeDryRun(appDir, profileDir, appCfg, logger)
    }

    chromePath, err := resolveChromiumExecutable(chromeDir)
    if err != nil {
        logger.Info("chromium runtime missing; stub launch only path=%s", filepath.Join(chromeDir, "chrome.exe"))
        return writeDryRun(appDir, profileDir, appCfg, logger)
    }

	args := buildChromiumArgs(profileDir, appCfg)
	logger.Info("launch chromium path=%s args=%s", chromePath, strings.Join(args, " "))

	cmd := exec.Command(chromePath, args...)
	stdoutLog, stderrLog, err := openChromiumLogFiles(root)
	if err != nil {
		return err
	}
	defer stdoutLog.Close()
	defer stderrLog.Close()
	cmd.Stdout = stdoutLog
	cmd.Stderr = stderrLog
	return cmd.Start()
}

func newPayloadSource(cfg config.RuntimeConfig, logger *logging.Logger, progress ProgressReporter) (PayloadSource, error) {
	switch cfg.Payload.Mode {
	case "", config.PayloadModeEmbedded:
		return NewEmbeddedPayloadSource(cfg, embeddedPayload), nil
	case config.PayloadModeRemote:
		source, err := NewRemotePayloadSource(cfg, logger)
		if err != nil {
			return nil, err
		}
		source.progressReporter = progress
		source.progress = progress.SetDownloadProgress
		return source, nil
	default:
		return nil, fmt.Errorf("unsupported payload mode: %s", cfg.Payload.Mode)
	}
}

func openChromiumLogFiles(root string) (*os.File, *os.File, error) {
	logsDir := filepath.Join(root, "logs")
	if err := os.MkdirAll(logsDir, 0o755); err != nil {
		return nil, nil, err
	}

	stdoutLog, err := os.OpenFile(filepath.Join(logsDir, "chromium.stdout.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	stderrLog, err := os.OpenFile(filepath.Join(logsDir, "chromium.stderr.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		_ = stdoutLog.Close()
		return nil, nil, err
	}

	return stdoutLog, stderrLog, nil
}

func resolveChromiumExecutable(chromeDir string) (string, error) {
    candidates := []string{"chrome.exe", "chromium.exe"}
    for _, name := range candidates {
        path := filepath.Join(chromeDir, name)
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
    }
    return "", fmt.Errorf("chromium runtime not found in %s", chromeDir)
}

func buildChromiumArgs(profileDir string, appCfg config.AppConfig) []string {
    args := []string{
        "--user-data-dir=" + profileDir,
    }

    if appCfg.WindowMode == "app" {
        args = append(args, "--app="+appCfg.StartURL)
    } else {
        args = append(args, appCfg.StartURL)
    }

    args = append(args, appCfg.ChromiumArgs...)
    return args
}

func verifyExtractedPayload(root string) error {
    manifest, err := loadManifest(filepath.Join(root, payloadManifest))
    if err != nil {
        return err
    }
    if manifest.Version == "" {
        return errors.New("payload manifest version is empty")
    }
    for _, item := range manifest.Files {
        if item.Path == "" {
            return errors.New("payload manifest contains empty path")
        }
        if item.SHA256 == "" {
            return fmt.Errorf("payload manifest missing checksum for %s", item.Path)
        }
        target := filepath.Join(root, filepath.FromSlash(item.Path))
        if checksum, err := checksumFile(target); err != nil {
            return err
        } else if checksum != item.SHA256 {
            return fmt.Errorf("payload file checksum mismatch: %s", item.Path)
        }
    }
    return nil
}

func loadManifest(path string) (PayloadManifest, error) {
    var manifest PayloadManifest
    data, err := os.ReadFile(path)
    if err != nil {
        return manifest, err
    }
    if err := json.Unmarshal(data, &manifest); err != nil {
        return manifest, err
    }
    sort.Slice(manifest.Files, func(i, j int) bool {
        return manifest.Files[i].Path < manifest.Files[j].Path
    })
    return manifest, nil
}

func loadPayloadState(root string) (PayloadState, error) {
    var state PayloadState
    data, err := os.ReadFile(filepath.Join(root, payloadStateFile))
    if err != nil {
        return state, err
    }
    if err := json.Unmarshal(data, &state); err != nil {
        return state, err
    }
    return state, nil
}

func writePayloadState(root string, state PayloadState) error {
    data, err := json.MarshalIndent(state, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(filepath.Join(root, payloadStateFile), append(data, '\n'), 0o644)
}

func writeDryRun(appDir, profileDir string, appCfg config.AppConfig, logger *logging.Logger) error {
    diagnosticsPath := filepath.Join(appDir, "diagnostics", "diagnostics.html")
    dryRunPath := filepath.Join(appDir, "diagnostics", "runtime-dry-run.txt")
    content := strings.Join([]string{
        "Kriptosfera runtime dry-run",
        "startUrl=" + appCfg.StartURL,
        "profileDir=" + profileDir,
        "diagnosticsPath=" + diagnosticsPath,
        "timestamp=" + time.Now().UTC().Format(time.RFC3339),
        "",
    }, "\n")
	if err := os.WriteFile(dryRunPath, []byte(content), 0o644); err != nil {
		return err
	}
	logger.Info("dry-run prepared file=%s diagnostics=%s", dryRunPath, diagnosticsPath)
	return nil
}

func acquireLock(appDir string) (func(), error) {
    lockPath := filepath.Join(filepath.Dir(appDir), filepath.Base(appDir)+payloadLockFile)
    if info, err := os.Stat(lockPath); err == nil {
        if time.Since(info.ModTime()) > lockTTL {
            _ = os.Remove(lockPath)
        }
    }

    file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
    if err != nil {
        if errors.Is(err, os.ErrExist) {
            return nil, fmt.Errorf("bootstrap already in progress: %s", lockPath)
        }
        return nil, err
    }
    _, _ = file.WriteString(time.Now().UTC().Format(time.RFC3339) + "\n")

    return func() {
        _ = file.Close()
        _ = os.Remove(lockPath)
    }, nil
}

func unzip(data []byte, dest string) error {
	return unzipReaderAt(bytes.NewReader(data), int64(len(data)), dest)
}

func unzipReaderAt(readerAt io.ReaderAt, size int64, dest string) error {
	r, err := zip.NewReader(readerAt, size)
	if err != nil {
		return err
	}
    cleanDest := filepath.Clean(dest)
    for _, f := range r.File {
        target := filepath.Join(cleanDest, filepath.Clean(f.Name))
        if !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) && filepath.Clean(target) != cleanDest {
            return errors.New("zip path traversal detected")
        }
        if f.FileInfo().IsDir() {
            if err := os.MkdirAll(target, 0o755); err != nil {
                return err
            }
            continue
        }
        if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
            return err
        }
        rc, err := f.Open()
        if err != nil {
            return err
        }
        out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode())
        if err != nil {
            rc.Close()
            return err
        }
        _, copyErr := io.Copy(out, rc)
        closeErr := out.Close()
        rcErr := rc.Close()
        if copyErr != nil {
            return copyErr
        }
        if closeErr != nil {
            return closeErr
        }
        if rcErr != nil {
            return rcErr
        }
    }
    return nil
}

func checksumBytes(data []byte) string {
    sum := sha256.Sum256(data)
    return hex.EncodeToString(sum[:])
}

func checksumFile(path string) (string, error) {
    file, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer file.Close()

    hash := sha256.New()
    if _, err := io.Copy(hash, file); err != nil {
        return "", err
    }
    return hex.EncodeToString(hash.Sum(nil)), nil
}

func appRoot() (string, error) {
    if runtime.GOOS == "windows" {
        base := os.Getenv("LOCALAPPDATA")
        if base == "" {
            return "", errors.New("LOCALAPPDATA is empty")
        }
        path := filepath.Join(base, "Kriptosfera")
        return path, os.MkdirAll(path, 0o755)
    }
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    path := filepath.Join(home, ".local", "share", "Kriptosfera")
    return path, os.MkdirAll(path, 0o755)
}
