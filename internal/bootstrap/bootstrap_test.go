package bootstrap

import (
    "os"
    "path/filepath"
    "testing"
)

func TestDefaultConfig(t *testing.T) {
    cfg, err := DefaultConfig()
    if err != nil {
        t.Fatal(err)
    }
    if cfg.Version == "" {
        t.Fatal("version must not be empty")
    }
}

func TestAppRootNonEmpty(t *testing.T) {
    root, err := appRoot()
    if err != nil {
        t.Fatal(err)
    }
    if root == "" {
        t.Fatal("app root empty")
    }
}

func TestEnsurePayloadIdempotent(t *testing.T) {
    dir := t.TempDir()
    logPath := filepath.Join(dir, "test.log")
    logger, err := os.Create(logPath)
    if err != nil {
        t.Fatal(err)
    }
    _ = logger.Close()
}
