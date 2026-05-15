package bootstrap

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type ExtensionSpec struct {
	Name            string
	Path            string
	ManifestPath    string
	ManifestVersion int
	Version         string
	ExtensionID     string
	ManifestError   string
}

type extensionManifest struct {
	ManifestVersion int    `json:"manifest_version"`
	Version         string `json:"version"`
	Key             string `json:"key"`
}

func detectExtensions(appDir string) ([]ExtensionSpec, error) {
	extensionsRoot := filepath.Join(appDir, "extensions")
	entries, err := os.ReadDir(extensionsRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	extensions := make([]ExtensionSpec, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		extPath := filepath.Join(extensionsRoot, name)
		spec := ExtensionSpec{
			Name:         name,
			Path:         extPath,
			ManifestPath: filepath.Join(extPath, "manifest.json"),
		}
		manifest, err := readExtensionManifest(spec.ManifestPath)
		if err == nil {
			spec.ManifestVersion = manifest.ManifestVersion
			spec.Version = manifest.Version
			if manifest.Key != "" {
				extensionID, idErr := extensionIDFromKey(manifest.Key)
				if idErr != nil {
					spec.ManifestError = idErr.Error()
				} else {
					spec.ExtensionID = extensionID
				}
			}
		} else if !os.IsNotExist(err) {
			spec.ManifestError = err.Error()
		}
		extensions = append(extensions, spec)
	}

	sort.Slice(extensions, func(i, j int) bool {
		return extensions[i].Name < extensions[j].Name
	})
	return extensions, nil
}

func loadableExtensions(exts []ExtensionSpec) []ExtensionSpec {
	loadable := make([]ExtensionSpec, 0, len(exts))
	for _, ext := range exts {
		if ext.ManifestError == "" && ext.ManifestVersion > 0 {
			loadable = append(loadable, ext)
		}
	}
	return loadable
}

func buildExtensionArgs(exts []ExtensionSpec) []string {
	if len(exts) == 0 {
		return nil
	}
	paths := make([]string, 0, len(exts))
	for _, ext := range exts {
		paths = append(paths, ext.Path)
	}
	joined := strings.Join(paths, ",")
	return []string{
		"--disable-extensions-except=" + joined,
		"--load-extension=" + joined,
	}
}

func readExtensionManifest(path string) (extensionManifest, error) {
	var manifest extensionManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return manifest, err
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		return manifest, fmt.Errorf("parse manifest %s: %w", path, err)
	}
	if manifest.ManifestVersion == 0 {
		return manifest, fmt.Errorf("manifest %s has no manifest_version", path)
	}
	if manifest.Version == "" {
		return manifest, fmt.Errorf("manifest %s has no version", path)
	}
	return manifest, nil
}

func extensionIDFromKey(key string) (string, error) {
	publicKey, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return "", fmt.Errorf("decode extension key: %w", err)
	}
	hash := sha256.Sum256(publicKey)
	out := make([]byte, 32)
	for i := 0; i < 16; i++ {
		out[i*2] = byte('a' + (hash[i] >> 4))
		out[i*2+1] = byte('a' + (hash[i] & 0x0f))
	}
	return string(out), nil
}
