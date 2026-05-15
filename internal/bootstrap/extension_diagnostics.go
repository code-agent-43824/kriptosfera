package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type extensionDiagnostics struct {
	GeneratedAt       string                    `json:"generatedAt"`
	ExtensionsRoot    string                    `json:"extensionsRoot"`
	DetectedCount     int                       `json:"detectedCount"`
	LoadableCount     int                       `json:"loadableCount"`
	ChromiumArgsAdded bool                      `json:"chromiumArgsAdded"`
	ChromiumArgs      []string                  `json:"chromiumArgs"`
	Extensions        []extensionDiagnosticsRow `json:"extensions"`
}

type extensionDiagnosticsRow struct {
	Name            string `json:"name"`
	Path            string `json:"path"`
	ManifestPath    string `json:"manifestPath"`
	ManifestVersion int    `json:"manifestVersion,omitempty"`
	Version         string `json:"version,omitempty"`
	ExtensionID     string `json:"extensionId,omitempty"`
	ManifestError   string `json:"manifestError,omitempty"`
	Loadable        bool   `json:"loadable"`
}

func writeExtensionStatus(appDir string, exts []ExtensionSpec, extensionArgs []string) error {
	rows := make([]extensionDiagnosticsRow, 0, len(exts))
	loadableCount := 0
	for _, ext := range exts {
		loadable := ext.ManifestError == "" && ext.ManifestVersion > 0
		if loadable {
			loadableCount++
		}
		rows = append(rows, extensionDiagnosticsRow{
			Name:            ext.Name,
			Path:            ext.Path,
			ManifestPath:    ext.ManifestPath,
			ManifestVersion: ext.ManifestVersion,
			Version:         ext.Version,
			ExtensionID:     ext.ExtensionID,
			ManifestError:   ext.ManifestError,
			Loadable:        loadable,
		})
	}

	status := extensionDiagnostics{
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		ExtensionsRoot:    filepath.Join(appDir, "extensions"),
		DetectedCount:     len(exts),
		LoadableCount:     loadableCount,
		ChromiumArgsAdded: len(extensionArgs) > 0,
		ChromiumArgs:      extensionArgs,
		Extensions:        rows,
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(appDir, "diagnostics", "extension-status.js")
	payload := append([]byte("window.__KRIPTOSFERA_EXTENSION_STATUS__ = "), data...)
	payload = append(payload, []byte(";\n")...)
	return os.WriteFile(path, payload, 0o644)
}
