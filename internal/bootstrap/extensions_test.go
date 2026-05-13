package bootstrap

import (
	"os"
	"path/filepath"
	"reflect"
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
	if err := os.WriteFile(filepath.Join(alpha, "manifest.json"), []byte(`{"manifest_version":3}`), 0o644); err != nil {
		t.Fatal(err)
	}

	exts := []ExtensionSpec{
		{Name: "alpha", Path: alpha, ManifestPath: filepath.Join(alpha, "manifest.json")},
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
	got := buildChromiumArgs(`C:\profile`, cfg, []string{"--load-extension=C:\\ext"})
	want := []string{
		`--user-data-dir=C:\profile`,
		`--load-extension=C:\\ext`,
		`--app=https://example.test`,
		`--enable-logging`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected chromium args: %#v", got)
	}
}
