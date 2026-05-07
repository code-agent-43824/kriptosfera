package bootstrap

import (
    "archive/zip"
    "bytes"
    "embed"
    "errors"
    "fmt"
    "io"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"

    "github.com/code-agent-43824/kriptosfera/internal/config"
    "github.com/code-agent-43824/kriptosfera/internal/logging"
)

//go:embed payload.zip
var embeddedPayload []byte

//go:embed app-version.txt
var versionFile embed.FS

type RuntimeConfig struct {
    ProductName string
    Version     string
}

func DefaultConfig() (RuntimeConfig, error) {
    raw, err := versionFile.ReadFile("app-version.txt")
    if err != nil {
        return RuntimeConfig{}, err
    }
    return RuntimeConfig{ProductName: "Kriptosfera Demo", Version: strings.TrimSpace(string(raw))}, nil
}

func Run(cfg RuntimeConfig) error {
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
    logger.Info("launcher start version=%s os=%s", cfg.Version, runtime.GOOS)

    appDir := filepath.Join(root, "apps", "demo", cfg.Version)
    if err := ensurePayload(appDir, logger); err != nil {
        return err
    }
    appCfg, err := config.Load(filepath.Join(appDir, "config", "app-config.json"))
    if err != nil {
        return err
    }
    logger.Info("payload ready start_url=%s", appCfg.StartURL)

    if runtime.GOOS != "windows" {
        logger.Info("non-windows environment detected; dry-run only")
        fmt.Printf("Payload prepared at %s\nStart URL: %s\n", appDir, appCfg.StartURL)
        return nil
    }

    chromePath := filepath.Join(appDir, "chromium", "chrome.exe")
    if _, err := os.Stat(chromePath); err != nil {
        return fmt.Errorf("chromium runtime not found at %s", chromePath)
    }

    profileDir := filepath.Join(root, "profiles", appCfg.ProfileName)
    if err := os.MkdirAll(profileDir, 0o755); err != nil {
        return err
    }

    args := []string{
        "--user-data-dir=" + profileDir,
        "--app=" + appCfg.StartURL,
    }
    args = append(args, appCfg.ChromiumArgs...)
    logger.Info("launch chromium path=%s args=%s", chromePath, strings.Join(args, " "))

    cmd := exec.Command(chromePath, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Start()
}

func ensurePayload(appDir string, logger *logging.Logger) error {
    marker := filepath.Join(appDir, ".payload-ready")
    if _, err := os.Stat(marker); err == nil {
        logger.Info("payload already prepared path=%s", appDir)
        return nil
    }
    logger.Info("extract payload path=%s", appDir)
    if err := unzip(embeddedPayload, appDir); err != nil {
        return err
    }
    return os.WriteFile(marker, []byte("ok\n"), 0o644)
}

func unzip(data []byte, dest string) error {
    r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
    if err != nil {
        return err
    }
    for _, f := range r.File {
        target := filepath.Join(dest, filepath.Clean(f.Name))
        if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) && filepath.Clean(target) != filepath.Clean(dest) {
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
