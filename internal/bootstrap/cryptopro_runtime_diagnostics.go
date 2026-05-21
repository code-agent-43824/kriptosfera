package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const cryptoProRuntimeDiagnosticsFile = "cryptopro-runtime.json"

type CryptoProRuntimeReport struct {
	AppDir          string                          `json:"appDir"`
	PluginRoot      string                          `json:"pluginRoot,omitempty"`
	ExtensionID     string                          `json:"extensionId,omitempty"`
	Bundle          CryptoProRuntimeBundle          `json:"bundle"`
	NativeMessaging CryptoProRuntimeNativeMessaging `json:"nativeMessaging"`
	ExpectedFiles   []CryptoProRuntimeFile          `json:"expectedFiles"`
}

type CryptoProRuntimeBundle struct {
	Component     string `json:"component,omitempty"`
	Version       string `json:"version,omitempty"`
	SHA256        string `json:"sha256,omitempty"`
	LayoutVersion int    `json:"layoutVersion,omitempty"`
}

type CryptoProRuntimeNativeMessaging struct {
	HostName     string `json:"hostName,omitempty"`
	ManifestPath string `json:"manifestPath,omitempty"`
	HostPath     string `json:"hostPath,omitempty"`
	Registered   bool   `json:"registered"`
	Skipped      bool   `json:"skipped"`
	RegistryKey  string `json:"registryKey,omitempty"`
}

type CryptoProRuntimeFile struct {
	Suffix string `json:"suffix"`
	Path   string `json:"path,omitempty"`
	Exists bool   `json:"exists"`
	SHA256 string `json:"sha256,omitempty"`
	Error  string `json:"error,omitempty"`
}

func WriteCryptoProRuntimeDiagnostics(appDir, pluginRoot string, native NativeMessagingResult, extensions []ExtensionSpec) (string, error) {
	report := CryptoProRuntimeReport{
		AppDir:      appDir,
		PluginRoot:  pluginRoot,
		ExtensionID: bestCryptoProExtensionID(extensions),
		NativeMessaging: CryptoProRuntimeNativeMessaging{
			HostName:     native.HostName,
			ManifestPath: native.ManifestPath,
			HostPath:     native.HostPath,
			Registered:   native.Registered,
			Skipped:      native.Skipped,
			RegistryKey:  native.RegistryKey,
		},
		ExpectedFiles: inspectCryptoProRuntimeFiles(pluginRoot),
	}
	if state, err := loadCryptoProPluginState(appDir); err == nil {
		report.Bundle = CryptoProRuntimeBundle{
			Component:     state.Component,
			Version:       state.Version,
			SHA256:        state.SHA256,
			LayoutVersion: state.LayoutVersion,
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(appDir, "diagnostics", cryptoProRuntimeDiagnosticsFile)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	return path, os.WriteFile(path, append(data, '\n'), 0o644)
}

func inspectCryptoProRuntimeFiles(pluginRoot string) []CryptoProRuntimeFile {
	files := make([]CryptoProRuntimeFile, 0, len(requiredCryptoProPluginFiles))
	for _, suffix := range requiredCryptoProPluginFiles {
		item := CryptoProRuntimeFile{Suffix: suffix}
		if pluginRoot == "" {
			item.Error = "plugin root is empty"
			files = append(files, item)
			continue
		}
		path, err := findFileBySlashSuffix(pluginRoot, suffix)
		if err != nil {
			item.Error = err.Error()
			files = append(files, item)
			continue
		}
		item.Path = path
		item.Exists = true
		if checksum, err := checksumFile(path); err == nil {
			item.SHA256 = checksum
		} else {
			item.Error = err.Error()
		}
		files = append(files, item)
	}
	return files
}

func bestCryptoProExtensionID(extensions []ExtensionSpec) string {
	id, err := selectCryptoProExtensionID(extensions)
	if err == nil {
		return id
	}
	return ""
}
