package bootstrap

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

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
	if cfg.Payload.Mode != config.PayloadModeEmbedded {
		t.Fatalf("expected embedded mode, got %s", cfg.Payload.Mode)
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
	rootDir := t.TempDir()
	logger := testLogger(t)
	cfg := testRuntimeConfig("0.1.0")
	payload := testPayloadZip(t, "0.1.0")
	appDir := testAppDir(rootDir, cfg.Version)

	reused, err := preparePayloadFromBytes(rootDir, cfg, logger, payload)
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

	reused, err = preparePayloadFromBytes(rootDir, cfg, logger, payload)
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
	rootDir := t.TempDir()
	logger := testLogger(t)
	cfg := testRuntimeConfig("0.1.0")
	payload := testPayloadZip(t, "0.1.0")
	appDir := testAppDir(rootDir, cfg.Version)

	if _, err := preparePayloadFromBytes(rootDir, cfg, logger, payload); err != nil {
		t.Fatal(err)
	}
	brokenFile := filepath.Join(appDir, "diagnostics", "diagnostics.html")
	if err := os.Remove(brokenFile); err != nil {
		t.Fatal(err)
	}

	reused, err := preparePayloadFromBytes(rootDir, cfg, logger, payload)
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
	rootDir := t.TempDir()
	logger := testLogger(t)

	if _, err := preparePayloadFromBytes(rootDir, testRuntimeConfig("0.1.0"), logger, testPayloadZip(t, "0.1.0")); err != nil {
		t.Fatal(err)
	}
	if _, err := preparePayloadFromBytes(rootDir, testRuntimeConfig("0.2.0"), logger, testPayloadZip(t, "0.2.0")); err != nil {
		t.Fatal(err)
	}
	appDir := testAppDir(rootDir, "0.2.0")

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

func TestEmbeddedPayloadSourceExposesExpectedMetadata(t *testing.T) {
	cfg := testRuntimeConfig("0.1.0")
	payload := testPayloadZip(t, "0.1.0")
	source := NewEmbeddedPayloadSource(cfg, payload)

	if source.Mode() != config.PayloadModeEmbedded {
		t.Fatalf("expected embedded mode, got %s", source.Mode())
	}
	if source.Version() != "0.1.0" {
		t.Fatalf("expected version 0.1.0, got %s", source.Version())
	}
	archive, err := source.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	data, err := io.ReadAll(io.NewSectionReader(archive.ReaderAt, 0, archive.Size))
	if err != nil {
		t.Fatal(err)
	}
	if checksumBytes(data) != source.ExpectedSHA256() {
		t.Fatal("embedded payload sha256 mismatch")
	}
}

func TestRemotePayloadSourceDownloadsArchive(t *testing.T) {
	payload := testPayloadZip(t, "0.1.0")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cfg := testRemoteRuntimeConfig("0.1.0", server.URL+"/payload.zip", checksumBytes(payload), int64(len(payload)))
	source, err := NewRemotePayloadSource(cfg, testLogger(t))
	if err != nil {
		t.Fatal(err)
	}
	source.client = server.Client()

	archive, err := source.Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	data, err := io.ReadAll(io.NewSectionReader(archive.ReaderAt, 0, archive.Size))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatal("downloaded payload does not match source")
	}
}

func TestDefaultDownloadClientAllowsSlowFirstRunPayload(t *testing.T) {
	if defaultDownloadClient.Timeout < 30*time.Minute {
		t.Fatalf("default remote payload timeout is too short for slow first-run downloads: %s", defaultDownloadClient.Timeout)
	}
}

func TestRemotePayloadSourceRejectsNonHTTPS(t *testing.T) {
	cfg := testRemoteRuntimeConfig("0.1.0", "http://example.test/payload.zip", "abc", 10)
	source, err := NewRemotePayloadSource(cfg, testLogger(t))
	if err != nil {
		t.Fatal(err)
	}
	_, err = source.Open(context.Background())
	if err == nil {
		t.Fatal("expected non-https payload URL error")
	}
	launcherErr := &LauncherError{}
	if !errors.As(err, &launcherErr) || launcherErr.Code != ErrPayloadDownloadFailed {
		t.Fatalf("expected %s, got %v", ErrPayloadDownloadFailed, err)
	}
}

func TestRemotePayloadSourceRejectsHashMismatch(t *testing.T) {
	payload := testPayloadZip(t, "0.1.0")
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cfg := testRemoteRuntimeConfig("0.1.0", server.URL+"/payload.zip", checksumBytes([]byte("wrong")), int64(len(payload)))
	source, err := NewRemotePayloadSource(cfg, testLogger(t))
	if err != nil {
		t.Fatal(err)
	}
	source.client = server.Client()

	_, err = source.Open(context.Background())
	if err == nil {
		t.Fatal("expected hash mismatch error")
	}
	launcherErr := &LauncherError{}
	if !errors.As(err, &launcherErr) || launcherErr.Code != ErrPayloadHashMismatch {
		t.Fatalf("expected %s, got %v", ErrPayloadHashMismatch, err)
	}
}

func TestDownloadFileRejectsOversizedPayload(t *testing.T) {
	expected := int64(1024)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve well beyond the pinned expected size.
		_, _ = w.Write(make([]byte, expected*4))
	}))
	defer server.Close()

	_, err := DownloadFile(context.Background(), server.Client(), server.URL+"/payload.zip", expected, testLogger(t), nil)
	if err == nil {
		t.Fatal("expected oversized payload error")
	}
	launcherErr := &LauncherError{}
	if !errors.As(err, &launcherErr) || launcherErr.Code != ErrPayloadDownloadFailed {
		t.Fatalf("expected %s, got %v", ErrPayloadDownloadFailed, err)
	}
}

func TestPayloadManagerRemoteReusesCachedPayload(t *testing.T) {
	rootDir := t.TempDir()
	logger := testLogger(t)
	payload := testPayloadZip(t, "0.1.0")
	var hits atomic.Int32
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cfg := testRemoteRuntimeConfig("0.1.0", server.URL+"/payload.zip", checksumBytes(payload), int64(len(payload)))
	source, err := NewRemotePayloadSource(cfg, logger)
	if err != nil {
		t.Fatal(err)
	}
	source.client = server.Client()
	manager := PayloadManager{}

	result, err := manager.Prepare(context.Background(), source, rootDir, cfg, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Reused {
		t.Fatal("first remote prepare must not be reused")
	}

	source2, err := NewRemotePayloadSource(cfg, logger)
	if err != nil {
		t.Fatal(err)
	}
	source2.client = server.Client()
	result, err = manager.Prepare(context.Background(), source2, rootDir, cfg, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Reused {
		t.Fatal("second remote prepare must reuse cached payload")
	}
	if hits.Load() != 1 {
		t.Fatalf("expected one HTTP hit, got %d", hits.Load())
	}
}

func TestCryptoProPluginManagerExtractsAndReusesCurrentState(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)
	bundle := testCryptoProPluginZip(t)
	legacyDir := filepath.Join(appDir, "cryptopro", "plugin")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "old.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	manager := CryptoProPluginManager{
		Bundle:        bundle,
		Version:       "2.0.15000",
		SHA256:        checksumBytes(bundle),
		LayoutVersion: 3,
	}

	result, err := manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Reused {
		t.Fatal("first CryptoPro plugin extraction must not be reused")
	}
	if result.Path != filepath.Join(appDir, "Crypto Pro") {
		t.Fatalf("unexpected CryptoPro plugin root: %s", result.Path)
	}
	if _, err := os.Stat(filepath.Join(appDir, "Crypto Pro", "CAdES Browser Plug-in", "Mini CSP")); err != nil {
		t.Fatalf("expected shortened Mini CSP path: %v", err)
	}
	if _, err := os.Stat(filepath.Join(result.Path, "cryptopro-cades-plugin-2.0.15000")); err == nil {
		t.Fatal("archive top-level directory should not be extracted into AppData")
	}
	if _, err := os.Stat(filepath.Join(result.Path, "Program Files")); err == nil {
		t.Fatal("Program Files wrapper should not be extracted into AppData")
	}
	if _, err := os.Stat(filepath.Join(appDir, "cryptopro")); err == nil {
		t.Fatal("legacy cryptopro/plugin layout should be removed after layout v3 extraction")
	}
	if _, err := findFileBySlashSuffix(result.Path, "CAdES Browser Plug-in/nmcades.exe"); err != nil {
		t.Fatal(err)
	}

	stateBefore, err := os.ReadFile(filepath.Join(appDir, cryptoProPluginStateFile))
	if err != nil {
		t.Fatal(err)
	}
	result, err = manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.Reused {
		t.Fatal("second CryptoPro plugin extraction must reuse prepared state")
	}
	stateAfter, err := os.ReadFile(filepath.Join(appDir, cryptoProPluginStateFile))
	if err != nil {
		t.Fatal(err)
	}
	if string(stateBefore) != string(stateAfter) {
		t.Fatal("CryptoPro plugin state changed on reuse")
	}
}

func TestCryptoProPluginManagerRecoversMissingFile(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)
	bundle := testCryptoProPluginZip(t)
	manager := CryptoProPluginManager{
		Bundle:        bundle,
		Version:       "2.0.15000",
		SHA256:        checksumBytes(bundle),
		LayoutVersion: 3,
	}

	result, err := manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	hostPath, err := findFileBySlashSuffix(result.Path, "CAdES Browser Plug-in/nmcades.exe")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(hostPath); err != nil {
		t.Fatal(err)
	}

	result, err = manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Reused {
		t.Fatal("broken CryptoPro plugin extraction must be recovered")
	}
	if _, err := findFileBySlashSuffix(result.Path, "CAdES Browser Plug-in/nmcades.exe"); err != nil {
		t.Fatal(err)
	}
}

func TestCryptoProPluginManagerRequiresCadesAndMiniCSPFiles(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)
	bundle := testCryptoProPluginZipWithPaths(t, []string{
		"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe",
		"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.json",
		"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/npcades.dll",
	})
	manager := CryptoProPluginManager{
		Bundle:        bundle,
		Version:       "2.0.15000",
		SHA256:        checksumBytes(bundle),
		LayoutVersion: 3,
	}

	_, err := manager.Prepare(appDir, logger, noopProgressReporter{})
	if err == nil {
		t.Fatal("expected missing CAdES runtime files to fail layout validation")
	}
	if !strings.Contains(err.Error(), "cades.dll") {
		t.Fatalf("expected missing cades.dll error, got %v", err)
	}
}

func TestCryptoProPluginZipSkipsMSIPseudoPaths(t *testing.T) {
	if !shouldSkipCryptoProPluginZipEntry("cryptopro-cades-plugin-2.0.15000/.:Common/Crypto Pro/Shared/cadescom.dll") {
		t.Fatal("MSI pseudo-path entry must be skipped")
	}
	if shouldSkipCryptoProPluginZipEntry("cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe") {
		t.Fatal("required native host file must not be skipped")
	}
}

func TestCryptoProPluginZipMapsVendorSubtree(t *testing.T) {
	target, skip := mapCryptoProPluginZipEntry("cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi20.dll")
	if skip {
		t.Fatal("required Mini CSP file must not be skipped")
	}
	if target != "CAdES Browser Plug-in/Mini CSP/capi20.dll" {
		t.Fatalf("unexpected mapped path: %s", target)
	}
	if target, skip := mapCryptoProPluginZipEntry("cryptopro-cades-plugin-2.0.15000/Program Files 64/Crypto Pro/CAdES Browser Plug-in/nmcades.exe"); !skip || target != "" {
		t.Fatalf("64-bit Program Files subtree should be skipped, got target=%q skip=%v", target, skip)
	}
}

func TestCryptoProPluginManagerSkipsInvalidMSIPseudoPaths(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)
	bundle := testCryptoProPluginZipWithMSIPseudoPath(t)
	manager := CryptoProPluginManager{
		Bundle:        bundle,
		Version:       "2.0.15000",
		SHA256:        checksumBytes(bundle),
		LayoutVersion: 3,
	}

	result, err := manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := findFileBySlashSuffix(result.Path, "CAdES Browser Plug-in/nmcades.exe"); err != nil {
		t.Fatal(err)
	}
	// The MSI pseudo-path entry (name contains ':') must be skipped, so the file
	// must not exist. We only treat a successful stat (err == nil) as a failure:
	// on Windows os.Stat of a path containing ':' returns a syntax error rather
	// than fs.ErrNotExist, so checking os.IsNotExist would be non-portable.
	if _, err := os.Stat(filepath.Join(result.Path, "cryptopro-cades-plugin-2.0.15000", ".:Common")); err == nil {
		t.Fatal("MSI pseudo-path entry should not be extracted")
	}
}

func TestEmbeddedCryptoProBundleContainsRequiredFiles(t *testing.T) {
	if len(embeddedCryptoProPlugin) == 0 {
		t.Skip("embedded CryptoPro bundle not present (placeholder/dev build)")
	}
	reader, err := zip.NewReader(bytes.NewReader(embeddedCryptoProPlugin), int64(len(embeddedCryptoProPlugin)))
	if err != nil {
		t.Fatalf("open embedded bundle: %v", err)
	}
	has := func(suffix string) bool {
		want := filepath.ToSlash(filepath.Clean(suffix))
		for _, f := range reader.File {
			if f.FileInfo().IsDir() || shouldSkipCryptoProPluginZipEntry(f.Name) {
				continue
			}
			if strings.HasSuffix(filepath.ToSlash(filepath.Clean(f.Name)), want) {
				return true
			}
		}
		return false
	}
	for _, required := range requiredCryptoProPluginFiles {
		if !has(required) {
			t.Errorf("embedded CryptoPro bundle is missing required file: %s", required)
		}
	}
}

func TestPrepareCryptoProNativeMessagingReusesRegistration(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)

	var registrations int
	original := registerNativeMessagingHost
	registerNativeMessagingHost = func(string) error { registrations++; return nil }
	t.Cleanup(func() { registerNativeMessagingHost = original })

	bundle := testCryptoProPluginZip(t)
	manager := CryptoProPluginManager{Bundle: bundle, Version: "2.0.15000", SHA256: checksumBytes(bundle), LayoutVersion: 3}
	pluginResult, err := manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	extensions := []ExtensionSpec{{Name: "cryptopro-cades", ExtensionID: "iifchhfnnmpdbibifmljnfjhpififfog"}}

	first, err := PrepareCryptoProNativeMessaging(appDir, pluginResult.Path, extensions, logger)
	if err != nil {
		t.Fatal(err)
	}
	if first.Reused {
		t.Fatal("first native messaging preparation must not be reused")
	}
	second, err := PrepareCryptoProNativeMessaging(appDir, pluginResult.Path, extensions, logger)
	if err != nil {
		t.Fatal(err)
	}
	if !second.Reused {
		t.Fatal("second native messaging preparation must reuse existing registration")
	}
	if registrations != 1 {
		t.Fatalf("expected exactly one registry registration, got %d", registrations)
	}
}

func TestAcquireLockWaitsThenSucceedsAfterRelease(t *testing.T) {
	appDir := filepath.Join(t.TempDir(), "apps", "demo", "0.1.0")
	if err := os.MkdirAll(filepath.Dir(appDir), 0o755); err != nil {
		t.Fatal(err)
	}

	unlock, err := acquireLock(appDir)
	if err != nil {
		t.Fatal(err)
	}

	released := make(chan struct{})
	go func() {
		time.Sleep(150 * time.Millisecond)
		unlock()
		close(released)
	}()

	// Should block until the first holder releases, then succeed (not error).
	unlock2, err := acquireLock(appDir)
	if err != nil {
		t.Fatalf("expected lock acquisition to succeed after release, got %v", err)
	}
	defer unlock2()
	<-released
}

func TestPrepareCryptoProNativeMessagingWritesManifest(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)
	stubNativeMessagingRegistrar(t)
	bundle := testCryptoProPluginZip(t)
	manager := CryptoProPluginManager{
		Bundle:        bundle,
		Version:       "2.0.15000",
		SHA256:        checksumBytes(bundle),
		LayoutVersion: 3,
	}
	pluginResult, err := manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}

	result, err := PrepareCryptoProNativeMessaging(appDir, pluginResult.Path, []ExtensionSpec{{
		Name:        "cryptopro-cades",
		ExtensionID: "iifchhfnnmpdbibifmljnfjhpififfog",
	}}, logger)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Registered {
		t.Fatal("native messaging host must be marked as registered")
	}
	manifest, err := readNativeMessagingManifest(result.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Name != "ru.cryptopro.nmcades" {
		t.Fatalf("unexpected native host name: %s", manifest.Name)
	}
	if manifest.Type != "stdio" {
		t.Fatalf("unexpected native host type: %s", manifest.Type)
	}
	if manifest.AllowedOrigins[0] != "chrome-extension://iifchhfnnmpdbibifmljnfjhpififfog/" {
		t.Fatalf("unexpected allowed origin: %v", manifest.AllowedOrigins)
	}
	if _, err := os.Stat(manifest.Path); err != nil {
		t.Fatal(err)
	}
}

func TestWriteCryptoProRuntimeDiagnostics(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)
	stubNativeMessagingRegistrar(t)
	bundle := testCryptoProPluginZip(t)
	manager := CryptoProPluginManager{
		Bundle:        bundle,
		Version:       "2.0.15000",
		SHA256:        checksumBytes(bundle),
		LayoutVersion: 3,
	}
	pluginResult, err := manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}
	extensions := []ExtensionSpec{{
		Name:        "cryptopro-cades",
		ExtensionID: "iifchhfnnmpdbibifmljnfjhpififfog",
	}}
	nativeResult, err := PrepareCryptoProNativeMessaging(appDir, pluginResult.Path, extensions, logger)
	if err != nil {
		t.Fatal(err)
	}

	reportPath, err := WriteCryptoProRuntimeDiagnostics(appDir, pluginResult.Path, nativeResult, extensions)
	if err != nil {
		t.Fatal(err)
	}
	var report CryptoProRuntimeReport
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.AppDir != appDir {
		t.Fatalf("unexpected appDir: %s", report.AppDir)
	}
	if report.PluginRoot != pluginResult.Path {
		t.Fatalf("unexpected pluginRoot: %s", report.PluginRoot)
	}
	if report.ExtensionID != "iifchhfnnmpdbibifmljnfjhpififfog" {
		t.Fatalf("unexpected extension id: %s", report.ExtensionID)
	}
	if report.Bundle.Version != "2.0.15000" || report.Bundle.SHA256 == "" {
		t.Fatalf("unexpected bundle info: %#v", report.Bundle)
	}
	if report.NativeMessaging.ManifestPath == "" || report.NativeMessaging.HostPath == "" || !report.NativeMessaging.Registered {
		t.Fatalf("unexpected native messaging info: %#v", report.NativeMessaging)
	}
	if len(report.ExpectedFiles) != len(requiredCryptoProPluginFiles) {
		t.Fatalf("expected %d files, got %d", len(requiredCryptoProPluginFiles), len(report.ExpectedFiles))
	}
	for _, file := range report.ExpectedFiles {
		if !file.Exists || file.Path == "" || file.SHA256 == "" || file.Error != "" {
			t.Fatalf("unexpected file report for %s: %#v", file.Suffix, file)
		}
	}
}

func TestPrepareCryptoProNativeMessagingRequiresExtensionID(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)
	stubNativeMessagingRegistrar(t)
	bundle := testCryptoProPluginZip(t)
	manager := CryptoProPluginManager{
		Bundle:        bundle,
		Version:       "2.0.15000",
		SHA256:        checksumBytes(bundle),
		LayoutVersion: 3,
	}
	pluginResult, err := manager.Prepare(appDir, logger, noopProgressReporter{})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := PrepareCryptoProNativeMessaging(appDir, pluginResult.Path, nil, logger); err == nil {
		t.Fatal("expected missing extension id error")
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
	cfg := testAppConfig()
	cfg.DiagnosticsEnabled = false
	args := buildChromiumArgs(`C:\Profiles\demo`, cfg, nil)
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
	cfg.DiagnosticsEnabled = false
	args := buildChromiumArgs(`C:\Profiles\demo`, cfg, nil)
	lastArg := args[len(args)-len(cfg.ChromiumArgs)-1]
	if lastArg != "https://example.test" {
		t.Fatalf("expected plain URL arg, got %s", lastArg)
	}
}

func TestValidateAppConfigAcceptsStartURLAllowedOrigin(t *testing.T) {
	cfg := testAppConfig()
	cfg.StartURL = "https://www.cryptopro.ru/sites/default/files/products/cades/demopage/cades_bes_sample.html"
	cfg.AllowedOrigins = []string{"https://www.cryptopro.ru"}

	if err := validateAppConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAppConfigRejectsStartURLOutsideAllowedOrigins(t *testing.T) {
	cfg := testAppConfig()
	cfg.StartURL = "https://example.test"
	cfg.AllowedOrigins = []string{"https://www.cryptopro.ru"}

	if err := validateAppConfig(cfg); err == nil {
		t.Fatal("expected startUrl origin validation error")
	}
}

func TestValidateAppConfigRejectsInvalidAllowedOrigin(t *testing.T) {
	cfg := testAppConfig()
	cfg.AllowedOrigins = []string{"not an origin"}

	if err := validateAppConfig(cfg); err == nil {
		t.Fatal("expected invalid allowed origin error")
	}
}

func TestValidateAppConfigRejectsNonHTTPSDiagnosticsURL(t *testing.T) {
	cfg := testAppConfig()
	cfg.DiagnosticsURL = "http://127.0.0.1:12345/diagnostics.html"

	if err := validateAppConfig(cfg); err == nil {
		t.Fatal("expected diagnosticsUrl HTTPS validation error")
	}
}

func TestValidateAppConfigRejectsAllowedOriginWithPath(t *testing.T) {
	cfg := testAppConfig()
	cfg.AllowedOrigins = []string{"https://example.test/path"}

	if err := validateAppConfig(cfg); err == nil {
		t.Fatal("expected allowed origin with path error")
	}
}

func TestValidateAppConfigRejectsUnsafeProfileName(t *testing.T) {
	unsafe := []string{
		"",
		".",
		"..",
		"../escape",
		"..\\escape",
		"sub/dir",
		"sub\\dir",
		"C:profile",
		" demo",
		"demo ",
	}
	for _, name := range unsafe {
		cfg := testAppConfig()
		cfg.ProfileName = name
		if err := validateAppConfig(cfg); err == nil {
			t.Fatalf("expected profileName validation error for %q", name)
		}
	}
}

func TestValidateAppConfigAcceptsPlainProfileName(t *testing.T) {
	for _, name := range []string{"demo", "demo-1", "demo_profile", "Профиль1"} {
		cfg := testAppConfig()
		cfg.ProfileName = name
		if err := validateAppConfig(cfg); err != nil {
			t.Fatalf("expected profileName %q to be accepted: %v", name, err)
		}
	}
}

func TestWriteDryRunSkipsFileWhenDiagnosticsDisabled(t *testing.T) {
	appDir := t.TempDir()
	logger := testLogger(t)
	if err := os.MkdirAll(filepath.Join(appDir, "diagnostics"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := testAppConfig()
	cfg.DiagnosticsEnabled = false

	if err := writeDryRun(appDir, filepath.Join(appDir, "profile"), cfg, logger); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(appDir, "diagnostics", "runtime-dry-run.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected no dry-run diagnostics file, got %v", err)
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

func stubNativeMessagingRegistrar(t *testing.T) {
	t.Helper()
	original := registerNativeMessagingHost
	registerNativeMessagingHost = func(string) error { return nil }
	t.Cleanup(func() { registerNativeMessagingHost = original })
}

func preparePayloadFromBytes(rootDir string, cfg config.RuntimeConfig, logger *logging.Logger, payload []byte) (bool, error) {
	source := NewEmbeddedPayloadSource(cfg, payload)
	manager := PayloadManager{}
	result, err := manager.Prepare(context.Background(), source, rootDir, cfg, logger, noopProgressReporter{})
	if err != nil {
		return false, err
	}
	return result.Reused, nil
}

func TestFormatDownloadProgress(t *testing.T) {
	text := formatDownloadProgress(50, 100)
	if text != "Загрузка компонентов: 50% (50 B из 100 B)" {
		t.Fatalf("unexpected progress text: %s", text)
	}

	text = formatDownloadProgress(1536, 0)
	if text != "Загрузка компонентов: 1.5 KB" {
		t.Fatalf("unexpected progress text without total: %s", text)
	}
}

func TestDetectExtensionsReturnsSortedDirectories(t *testing.T) {
	appDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(appDir, "extensions", "zeta"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(appDir, "extensions", "alpha"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "extensions", "README.txt"), []byte("skip"), 0o644); err != nil {
		t.Fatal(err)
	}

	exts, err := detectExtensions(appDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(exts) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(exts))
	}
	if exts[0].Name != "alpha" || exts[1].Name != "zeta" {
		t.Fatalf("unexpected extension order: %#v", exts)
	}
	if got := exts[0].ManifestPath; got != filepath.Join(appDir, "extensions", "alpha", "manifest.json") {
		t.Fatalf("unexpected manifest path: %s", got)
	}
}

func TestDetectExtensionsMissingRootReturnsEmpty(t *testing.T) {
	appDir := t.TempDir()
	exts, err := detectExtensions(appDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(exts) != 0 {
		t.Fatalf("expected no extensions, got %d", len(exts))
	}
}

func testAppDir(rootDir, version string) string {
	return filepath.Join(rootDir, "apps", "demo", version)
}

func testRuntimeConfig(version string) config.RuntimeConfig {
	return config.RuntimeConfig{
		ProductName: "Kriptosfera Demo",
		Version:     version,
		Payload: config.RuntimePayloadConfig{
			Mode:    config.PayloadModeEmbedded,
			Version: version,
		},
	}
}

func testRemoteRuntimeConfig(version, url, sha256 string, size int64) config.RuntimeConfig {
	return config.RuntimeConfig{
		ProductName: "Kriptosfera Demo",
		Version:     version,
		Payload: config.RuntimePayloadConfig{
			Mode:    config.PayloadModeRemote,
			Version: version,
			URL:     url,
			SHA256:  sha256,
			Size:    size,
		},
	}
}

func testPayloadZip(t *testing.T, version string) []byte {
	t.Helper()
	files := map[string]string{
		"config/app-config.json":       mustJSON(t, testAppConfigWithVersion(version)),
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

func testCryptoProPluginZip(t *testing.T) []byte {
	t.Helper()
	return testCryptoProPluginZipWithExtraPaths(t, nil)
}

func testCryptoProPluginZipWithMSIPseudoPath(t *testing.T) []byte {
	t.Helper()
	return testCryptoProPluginZipWithExtraPaths(t, []string{
		"cryptopro-cades-plugin-2.0.15000/.:Common/Crypto Pro/Shared/cadescom.dll",
	})
}

func testCryptoProPluginZipWithExtraPaths(t *testing.T, extraPaths []string) []byte {
	t.Helper()
	paths := append([]string{}, requiredTestCryptoProPluginPaths...)
	paths = append(paths, extraPaths...)
	return testCryptoProPluginZipWithPaths(t, paths)
}

func testCryptoProPluginZipWithPaths(t *testing.T, paths []string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, path := range paths {
		w, err := zw.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(path)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var requiredTestCryptoProPluginPaths = []string{
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.json",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/npcades.dll",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/cades.dll",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/xades.dll",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/cplib.dll",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi10.dll",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi20.dll",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpcspi.dll",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpsuprt.dll",
	"cryptopro-cades-plugin-2.0.15000/Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpui.dll",
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
		DiagnosticsURL:     "https://mescheryakov.pro/kriptosfera/diagnostics/diagnostics.html",
		ChromiumArgs:       []string{"--no-first-run"},
	}
}
