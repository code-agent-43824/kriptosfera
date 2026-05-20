package bootstrap

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadableExtensionsKeepsOnlyManifestDirs(t *testing.T) {
	appDir := t.TempDir()
	alpha := filepath.Join(appDir, "extensions", "alpha")
	beta := filepath.Join(appDir, "extensions", "beta")
	if err := os.MkdirAll(alpha, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(beta, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(alpha, "manifest.json"), []byte(`{"manifest_version":3,"version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	exts := []ExtensionSpec{
		{Name: "alpha", Path: alpha, ManifestPath: filepath.Join(alpha, "manifest.json"), ManifestVersion: 3, Version: "1.0.0"},
		{Name: "beta", Path: beta, ManifestPath: filepath.Join(beta, "manifest.json")},
	}
	loadable := loadableExtensions(exts)
	if len(loadable) != 1 {
		t.Fatalf("expected 1 loadable extension, got %d", len(loadable))
	}
	if loadable[0].Name != "alpha" {
		t.Fatalf("unexpected loadable extension: %#v", loadable)
	}
}

func TestDetectExtensionsReadsManifestAndExtensionID(t *testing.T) {
	appDir := t.TempDir()
	path := filepath.Join(appDir, "extensions", "cryptopro-cades")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"manifest_version":3,"version":"1.3.17","key":"MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA5mbPcCXE+y3R5iCmWSTHYQzsRm3BYBHYMAK8gmiK4Y3jn8wh3xTjNM6qcWJXr7bCmy+bGfNjJW1Hpr4tVPnVKv45hxZ/7dfzsRUHWxMD/ErWy6UyVYGIR+rUlS8AXSVjx/9rFKqUuaepZ0N1TgvUODaHUWpmNSKkY9o8hI3MV95WQ8FSpmCxVu9iWBjlREtSOWM+8pmvPUccFi38Y9/rvF0OR2h7zbGTMfwZFyTJuVhPL7tKO1rcbO//XM+eIGmYyWlBraEkLpmDHnDcaHjxiB95lBRpW38agTOzL0wTM8UKEA5dZdRDJWqsF8M5cyS3Wmmjmk8TenuAdImhGcU6dwIDAQAB"}`
	if err := os.WriteFile(filepath.Join(path, "manifest.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	exts, err := detectExtensions(appDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(exts) != 1 {
		t.Fatalf("expected 1 extension, got %d", len(exts))
	}
	if exts[0].ExtensionID != "pfhgbfnnjiafkhfdkmpiflachepdcjod" {
		t.Fatalf("unexpected extension id: %s", exts[0].ExtensionID)
	}
	if exts[0].Version != "1.3.17" {
		t.Fatalf("unexpected extension version: %s", exts[0].Version)
	}
	if exts[0].ManifestError != "" {
		t.Fatalf("unexpected manifest error: %s", exts[0].ManifestError)
	}
}

func TestWriteExtensionStatusWritesJSEnvelope(t *testing.T) {
	appDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(appDir, "diagnostics"), 0o755); err != nil {
		t.Fatal(err)
	}
	exts := []ExtensionSpec{{
		Name:            "cryptopro-cades",
		Path:            filepath.Join(appDir, "extensions", "cryptopro-cades"),
		ManifestPath:    filepath.Join(appDir, "extensions", "cryptopro-cades", "manifest.json"),
		ManifestVersion: 3,
		Version:         "1.3.17",
		ExtensionID:     "pfhgbfnnjiafkhfdkmpiflachepdcjod",
	}}
	if err := writeExtensionStatus(appDir, exts, []string{"--load-extension=C:/ext"}); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(appDir, "diagnostics", "extension-status.js"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "window.__KRIPTOSFERA_EXTENSION_STATUS__ = ") {
		t.Fatalf("unexpected js envelope: %s", text)
	}
	if !strings.Contains(text, `"extensionId": "pfhgbfnnjiafkhfdkmpiflachepdcjod"`) {
		t.Fatalf("expected extension id in status: %s", text)
	}
}

func TestBuildExtensionArgs(t *testing.T) {
	exts := []ExtensionSpec{{Path: `C:\ext-a`}, {Path: `C:\ext-b`}}
	got := buildExtensionArgs(exts)
	want := []string{
		`--disable-extensions-except=C:\ext-a,C:\ext-b`,
		`--load-extension=C:\ext-a,C:\ext-b`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected args: %#v", got)
	}
}

func TestBuildChromiumArgsPlacesExtensionFlagsBeforeURL(t *testing.T) {
	cfg := testAppConfigWithVersion("0.1.0")
	cfg.StartURL = "https://example.test"
	cfg.ChromiumArgs = []string{"--enable-logging"}
	got := buildChromiumArgs(`C:\profile`, cfg, []string{"--load-extension=C:\\ext"}, "http://127.0.0.1:12345/diagnostics.html")
	want := []string{
		`--user-data-dir=C:\profile`,
		`--load-extension=C:\ext`,
		`https://example.test`,
		`http://127.0.0.1:12345/diagnostics.html`,
		`--enable-logging`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected chromium args: %#v", got)
	}
}
