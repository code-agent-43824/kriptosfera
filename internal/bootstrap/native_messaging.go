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
	cryptoProNativeHostName = "ru.cryptopro.nmcades"
	chromeNativeHostKeyPath = `Software\Google\Chrome\NativeMessagingHosts\ru.cryptopro.nmcades`
)

var registerNativeMessagingHost = registerCryptoProNativeMessagingHost

type NativeMessagingResult struct {
	HostName     string
	ManifestPath string
	HostPath     string
	Registered   bool
	Skipped      bool
	RegistryKey  string
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

	extensionID, err := selectCryptoProExtensionID(extensions)
	if err != nil {
		return NativeMessagingResult{}, err
	}
	hostPath, err := findFileBySlashSuffix(pluginDir, "Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe")
	if err != nil {
		return NativeMessagingResult{}, err
	}

	manifestPath := filepath.Join(appDir, "native-host", "cryptopro", cryptoProNativeHostName+".json")
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

	logger.Info("native messaging host ready name=%s manifest=%s host=%s", cryptoProNativeHostName, manifestPath, hostPath)
	return NativeMessagingResult{
		HostName:     cryptoProNativeHostName,
		ManifestPath: manifestPath,
		HostPath:     hostPath,
		Registered:   true,
		RegistryKey:  "HKCU\\" + chromeNativeHostKeyPath,
	}, nil
}

func selectCryptoProExtensionID(extensions []ExtensionSpec) (string, error) {
	for _, ext := range extensions {
		if ext.Name == "cryptopro-cades" && ext.ExtensionID != "" && ext.ManifestError == "" {
			return ext.ExtensionID, nil
		}
	}
	for _, ext := range extensions {
		if ext.ExtensionID != "" && ext.ManifestError == "" {
			return ext.ExtensionID, nil
		}
	}
	return "", errors.New("cryptopro extension id not found for native messaging manifest")
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
