package bootstrap

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

const (
	cryptoProNativeHostName  = "ru.cryptopro.nmcades"
	chromeNativeHostKeyPath  = `Software\Google\Chrome\NativeMessagingHosts\ru.cryptopro.nmcades`
	nativeMessagingStateFile = ".cryptopro-native-state.json"
)

var registerNativeMessagingHost = registerCryptoProNativeMessagingHost

type NativeMessagingResult struct {
	HostName     string
	ManifestPath string
	HostPath     string
	Registered   bool
	Reused       bool
	Skipped      bool
	RegistryKey  string
}

// nativeMessagingState records what was last written/registered so repeat
// launches can skip re-writing the manifest and re-spawning reg.exe when nothing
// changed. If the user clears the HKCU key out-of-band, deleting this state file
// (or a version/extension change) forces re-registration.
type nativeMessagingState struct {
	HostName     string `json:"hostName"`
	HostPath     string `json:"hostPath"`
	ExtensionID  string `json:"extensionId"`
	ManifestPath string `json:"manifestPath"`
}

type nativeMessagingManifest struct {
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Path           string   `json:"path"`
	Type           string   `json:"type"`
	AllowedOrigins []string `json:"allowed_origins"`
}

func PrepareCryptoProNativeMessaging(appDir, pluginDir string, extensions []ExtensionSpec, logger *logging.Logger) (NativeMessagingResult, error) {
	if pluginDir == "" {
		logger.Info("native messaging skipped: cryptopro plugin path is empty")
		return NativeMessagingResult{HostName: cryptoProNativeHostName, Skipped: true}, nil
	}

	extensionID, preferred, err := selectCryptoProExtensionID(extensions)
	if err != nil {
		return NativeMessagingResult{}, err
	}
	if !preferred {
		logger.Info("native messaging warning: cryptopro-cades extension not found, using fallback extension id=%s", extensionID)
	}
	hostPath, err := findFileBySlashSuffix(pluginDir, "Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe")
	if err != nil {
		return NativeMessagingResult{}, err
	}

	manifestPath := filepath.Join(appDir, "native-host", "cryptopro", cryptoProNativeHostName+".json")
	desiredState := nativeMessagingState{
		HostName:     cryptoProNativeHostName,
		HostPath:     hostPath,
		ExtensionID:  extensionID,
		ManifestPath: manifestPath,
	}
	result := NativeMessagingResult{
		HostName:     cryptoProNativeHostName,
		ManifestPath: manifestPath,
		HostPath:     hostPath,
		Registered:   true,
		RegistryKey:  "HKCU\\" + chromeNativeHostKeyPath,
	}

	// Skip the manifest write and registry registration when nothing changed
	// since the last successful run and the manifest is still present.
	if state, err := loadNativeMessagingState(appDir); err == nil && state == desiredState {
		if _, statErr := os.Stat(manifestPath); statErr == nil {
			logger.Info("native messaging host reused name=%s manifest=%s host=%s", cryptoProNativeHostName, manifestPath, hostPath)
			result.Reused = true
			return result, nil
		}
	}

	manifest := nativeMessagingManifest{
		Name:        cryptoProNativeHostName,
		Description: "CryptoPro CAdES Browser Plugin native host",
		Path:        hostPath,
		Type:        "stdio",
		AllowedOrigins: []string{
			"chrome-extension://" + extensionID + "/",
		},
	}
	if err := writeNativeMessagingManifest(manifestPath, manifest); err != nil {
		return NativeMessagingResult{}, err
	}
	if err := registerNativeMessagingHost(manifestPath); err != nil {
		return NativeMessagingResult{}, err
	}
	if err := writeNativeMessagingState(appDir, desiredState); err != nil {
		return NativeMessagingResult{}, err
	}

	logger.Info("native messaging host ready name=%s manifest=%s host=%s", cryptoProNativeHostName, manifestPath, hostPath)
	return result, nil
}

func loadNativeMessagingState(appDir string) (nativeMessagingState, error) {
	var state nativeMessagingState
	data, err := os.ReadFile(filepath.Join(appDir, nativeMessagingStateFile))
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	return state, nil
}

func writeNativeMessagingState(appDir string, state nativeMessagingState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(appDir, nativeMessagingStateFile), append(data, '\n'), 0o644)
}

// selectCryptoProExtensionID returns the extension id to bind the native host
// to. The preferred return value is true only when the canonical
// "cryptopro-cades" extension was matched; it is false when the function fell
// back to an arbitrary extension, so callers can warn.
func selectCryptoProExtensionID(extensions []ExtensionSpec) (id string, preferred bool, err error) {
	for _, ext := range extensions {
		if ext.Name == "cryptopro-cades" && ext.ExtensionID != "" && ext.ManifestError == "" {
			return ext.ExtensionID, true, nil
		}
	}
	for _, ext := range extensions {
		if ext.ExtensionID != "" && ext.ManifestError == "" {
			return ext.ExtensionID, false, nil
		}
	}
	return "", false, errors.New("cryptopro extension id not found for native messaging manifest")
}

func writeNativeMessagingManifest(path string, manifest nativeMessagingManifest) error {
	if manifest.Name == "" {
		return errors.New("native messaging manifest name is empty")
	}
	if manifest.Path == "" {
		return errors.New("native messaging manifest host path is empty")
	}
	if len(manifest.AllowedOrigins) == 0 {
		return errors.New("native messaging manifest allowed_origins is empty")
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func readNativeMessagingManifest(path string) (nativeMessagingManifest, error) {
	var manifest nativeMessagingManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("parse native messaging manifest %s: %w", path, err)
	}
	return manifest, nil
}
