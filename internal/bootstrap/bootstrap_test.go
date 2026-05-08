package bootstrap

import (
    "archive/zip"
    "bytes"
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/code-agent-43824/kriptosfera/internal/config"
    "github.com/code-agent-43824/kriptosfera/internal/logging"
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

func TestEnsurePayloadExtractsAndReusesCurrentState(t *testing.T) {
    appDir := t.TempDir()
    logger := testLogger(t)
    cfg := RuntimeConfig{Version: "0.1.0"}
    payload := testPayloadZip(t, "0.1.0")

    reused, err := ensurePayloadFromBytes(appDir, cfg, logger, payload)
    if err != nil {
        t.Fatal(err)
    }
    if reused {
        t.Fatal("first extraction must not report reused state")
    }

    stateBefore, err := os.ReadFile(filepath.Join(appDir, payloadStateFile))
    if err != nil {
        t.Fatal(err)
    }

    reused, err = ensurePayloadFromBytes(appDir, cfg, logger, payload)
    if err != nil {
        t.Fatal(err)
    }
    if !reused {
        t.Fatal("second extraction must reuse prepared payload")
    }

    stateAfter, err := os.ReadFile(filepath.Join(appDir, payloadStateFile))
    if err != nil {
        t.Fatal(err)
    }
    if string(stateBefore) != string(stateAfter) {
        t.Fatal("state file changed on reused payload")
    }
}

func TestEnsurePayloadRecoversMissingFile(t *testing.T) {
    appDir := t.TempDir()
    logger := testLogger(t)
    cfg := RuntimeConfig{Version: "0.1.0"}
    payload := testPayloadZip(t, "0.1.0")

    if _, err := ensurePayloadFromBytes(appDir, cfg, logger, payload); err != nil {
        t.Fatal(err)
    }
    brokenFile := filepath.Join(appDir, "diagnostics", "diagnostics.html")
    if err := os.Remove(brokenFile); err != nil {
        t.Fatal(err)
    }

    reused, err := ensurePayloadFromBytes(appDir, cfg, logger, payload)
    if err != nil {
        t.Fatal(err)
    }
    if reused {
        t.Fatal("broken payload must be re-extracted")
    }
    if _, err := os.Stat(brokenFile); err != nil {
        t.Fatal(err)
    }
}

func TestEnsurePayloadReextractsOnVersionChange(t *testing.T) {
    appDir := t.TempDir()
    logger := testLogger(t)

    if _, err := ensurePayloadFromBytes(appDir, RuntimeConfig{Version: "0.1.0"}, logger, testPayloadZip(t, "0.1.0")); err != nil {
        t.Fatal(err)
    }
    if _, err := ensurePayloadFromBytes(appDir, RuntimeConfig{Version: "0.2.0"}, logger, testPayloadZip(t, "0.2.0")); err != nil {
        t.Fatal(err)
    }

    state, err := loadPayloadState(appDir)
    if err != nil {
        t.Fatal(err)
    }
    if state.Version != "0.2.0" {
        t.Fatalf("expected version 0.2.0, got %s", state.Version)
    }

    manifest, err := loadManifest(filepath.Join(appDir, payloadManifest))
    if err != nil {
        t.Fatal(err)
    }
    if manifest.Version != "0.2.0" {
        t.Fatalf("expected manifest version 0.2.0, got %s", manifest.Version)
    }
}

func TestWriteDryRunCreatesStubFile(t *testing.T) {
    appDir := t.TempDir()
    logger := testLogger(t)
    if err := os.MkdirAll(filepath.Join(appDir, "diagnostics"), 0o755); err != nil {
        t.Fatal(err)
    }

    err := writeDryRun(appDir, filepath.Join(appDir, "profile"), testAppConfig(), logger)
    if err != nil {
        t.Fatal(err)
    }

    dryRunPath := filepath.Join(appDir, "diagnostics", "runtime-dry-run.txt")
    data, err := os.ReadFile(dryRunPath)
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(string(data), "startUrl=https://example.test") {
        t.Fatal("dry-run file does not contain start URL")
    }
}

func TestResolveChromiumExecutableFindsChrome(t *testing.T) {
    dir := t.TempDir()
    chromePath := filepath.Join(dir, "chrome.exe")
    if err := os.WriteFile(chromePath, []byte("stub"), 0o644); err != nil {
        t.Fatal(err)
    }

    resolved, err := resolveChromiumExecutable(dir)
    if err != nil {
        t.Fatal(err)
    }
    if resolved != chromePath {
        t.Fatalf("expected %s, got %s", chromePath, resolved)
    }
}

func TestBuildChromiumArgsAppMode(t *testing.T) {
    args := buildChromiumArgs(`C:\Profiles\demo`, testAppConfig())
    joined := strings.Join(args, " ")
    if !strings.Contains(joined, "--user-data-dir=C:\\Profiles\\demo") {
        t.Fatal("missing user-data-dir arg")
    }
    if !strings.Contains(joined, "--app=https://example.test") {
        t.Fatal("missing app mode arg")
    }
}

func TestBuildChromiumArgsWindowModeBrowser(t *testing.T) {
    cfg := testAppConfig()
    cfg.WindowMode = "browser"
    args := buildChromiumArgs(`C:\Profiles\demo`, cfg)
    lastArg := args[len(args)-len(cfg.ChromiumArgs)-1]
    if lastArg != "https://example.test" {
        t.Fatalf("expected plain URL arg, got %s", lastArg)
    }
}

func TestUnzipRejectsTraversal(t *testing.T) {
    var buf bytes.Buffer
    zw := zip.NewWriter(&buf)
    w, err := zw.Create("../evil.txt")
    if err != nil {
        t.Fatal(err)
    }
    if _, err := w.Write([]byte("oops")); err != nil {
        t.Fatal(err)
    }
    if err := zw.Close(); err != nil {
        t.Fatal(err)
    }

    if err := unzip(buf.Bytes(), t.TempDir()); err == nil {
        t.Fatal("expected path traversal error")
    }
}

func testLogger(t *testing.T) *logging.Logger {
    t.Helper()
    logger, err := logging.New(filepath.Join(t.TempDir(), "launcher.log"))
    if err != nil {
        t.Fatal(err)
    }
    t.Cleanup(func() { _ = logger.Close() })
    return logger
}

func ensurePayloadFromBytes(appDir string, cfg RuntimeConfig, logger *logging.Logger, payload []byte) (bool, error) {
    original := embeddedPayload
    embeddedPayload = payload
    defer func() { embeddedPayload = original }()
    return ensurePayload(appDir, cfg, logger)
}

func testPayloadZip(t *testing.T, version string) []byte {
    t.Helper()
    files := map[string]string{
        "config/app-config.json": mustJSON(t, testAppConfigWithVersion(version)),
        "diagnostics/diagnostics.html": "<html><body>ok</body></html>\n",
    }

    manifest := PayloadManifest{Version: version}
    for _, path := range []string{"config/app-config.json", "diagnostics/diagnostics.html"} {
        manifest.Files = append(manifest.Files, PayloadManifestFile{
            Path:   path,
            SHA256: checksumBytes([]byte(files[path])),
        })
    }
    files[payloadManifest] = mustJSON(t, manifest)

    var buf bytes.Buffer
    zw := zip.NewWriter(&buf)
    for _, path := range []string{"config/app-config.json", "diagnostics/diagnostics.html", payloadManifest} {
        w, err := zw.Create(path)
        if err != nil {
            t.Fatal(err)
        }
        if _, err := w.Write([]byte(files[path])); err != nil {
            t.Fatal(err)
        }
    }
    if err := zw.Close(); err != nil {
        t.Fatal(err)
    }
    return buf.Bytes()
}

func mustJSON(t *testing.T, value any) string {
    t.Helper()
    data, err := json.Marshal(value)
    if err != nil {
        t.Fatal(err)
    }
    return string(data)
}

func testAppConfig() config.AppConfig {
    return testAppConfigWithVersion("0.1.0")
}

func testAppConfigWithVersion(version string) config.AppConfig {
    return config.AppConfig{
        AppID:              "ru.kriptosfera.demo",
        ProductName:        "Kriptosfera Demo",
        CustomerName:       "Demo Customer",
        Version:            version,
        StartURL:           "https://example.test",
        AllowedOrigins:     []string{"https://example.test"},
        ProfileName:        "demo",
        WindowMode:         "app",
        DiagnosticsEnabled: true,
        ChromiumArgs:       []string{"--no-first-run"},
    }
}
