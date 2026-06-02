package bootstrap

import (
	"fmt"
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
	manifest := `{"manifest_version":2,"version":"1.2.13","key":"MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAsePKp3waq5KKtMV6DGvvY706kmxCCvsaVCoHylp2xlNuAlIXZtuRv+0l425qAqXJuMOx0CCniDQFB8LUqPw8W8C3tlZNhLh9RTayAsHMhgjeVJOO1BsX/UYsyt2WM2ZNU93M/VFl8lLpwPUwTx0O+ThLZGWyryUJtOfNJm0aZNCSgviM3Go6kanqBEe5H4SlItMd+96F0oYjh4y71ZfiUruqTPyKv9IfZbg6BWCf6Et5K6gyJtGG2DZ0oyZruub/OfxcJbOIGYBilQmbUIvX9tyzVhlVjgdKRIZxtn+P+xI38MMtKIgvp8giSLyHnUQYTjaw/TcBxVYoJknqUijK1QIDAQAB"}`
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
	if exts[0].ExtensionID != "iifchhfnnmpdbibifmljnfjhpififfog" {
		t.Fatalf("unexpected extension id: %s", exts[0].ExtensionID)
	}
	if exts[0].Version != "1.2.13" {
		t.Fatalf("unexpected extension version: %s", exts[0].Version)
	}
	if exts[0].ManifestError != "" {
		t.Fatalf("unexpected manifest error: %s", exts[0].ManifestError)
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

func TestRequiresExtensionManifestV2Policy(t *testing.T) {
	exts := []ExtensionSpec{
		{Name: "legacy", ManifestVersion: 2},
		{Name: "modern", ManifestVersion: 3},
	}
	if !requiresExtensionManifestV2Policy(exts) {
		t.Fatal("manifest v2 extension must request the compatibility policy")
	}
}

func TestRequiresExtensionManifestV2PolicySkipsModernOrBrokenExtensions(t *testing.T) {
	exts := []ExtensionSpec{
		{Name: "modern", ManifestVersion: 3},
		{Name: "broken", ManifestVersion: 2, ManifestError: "parse error"},
	}
	if requiresExtensionManifestV2Policy(exts) {
		t.Fatal("manifest v2 policy must be skipped for modern or broken extensions")
	}
}

func TestApplyChromeCompatibilityPoliciesReturnsFallbackArgsOnPolicyWriteFailure(t *testing.T) {
	original := writeChromePolicyDWORD
	writeChromePolicyDWORD = func(string, int) error {
		return fmt.Errorf("access denied")
	}
	t.Cleanup(func() { writeChromePolicyDWORD = original })

	got := ApplyChromeCompatibilityPolicies([]ExtensionSpec{{Name: "legacy", ManifestVersion: 2}}, testLogger(t))
	want := chromeLegacyMV2FallbackArgs()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected fallback args: %#v", got)
	}
}

func TestApplyChromeCompatibilityPoliciesReturnsNoFallbackOnPolicyWriteSuccess(t *testing.T) {
	original := writeChromePolicyDWORD
	writeChromePolicyDWORD = func(string, int) error {
		return nil
	}
	t.Cleanup(func() { writeChromePolicyDWORD = original })

	got := ApplyChromeCompatibilityPolicies([]ExtensionSpec{{Name: "legacy", ManifestVersion: 2}}, testLogger(t))
	if len(got) != 0 {
		t.Fatalf("unexpected fallback args after successful policy write: %#v", got)
	}
}

func TestBuildChromiumArgsPlacesExtensionFlagsBeforeURL(t *testing.T) {
	cfg := testAppConfigWithVersion("0.1.0")
	cfg.StartURL = "https://example.test"
	cfg.ChromiumArgs = []string{"--enable-logging"}
	got := buildChromiumArgs(`C:\profile`, cfg, []string{"--load-extension=C:\\ext", "--enable-features=AllowLegacyMV2Extensions"})
	want := []string{
		`--user-data-dir=C:\profile`,
		`--load-extension=C:\ext`,
		`--enable-features=AllowLegacyMV2Extensions`,
		`--app=https://example.test`,
		`--enable-logging`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected chromium args: %#v", got)
	}
}
