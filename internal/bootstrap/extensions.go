package bootstrap

import (
	"os"
	"path/filepath"
	"sort"
)

type ExtensionSpec struct {
	Name        string
	Path        string
	ManifestPath string
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
		extensions = append(extensions, ExtensionSpec{
			Name:         name,
			Path:         extPath,
			ManifestPath: filepath.Join(extPath, "manifest.json"),
		})
	}

	sort.Slice(extensions, func(i, j int) bool {
		return extensions[i].Name < extensions[j].Name
	})
	return extensions, nil
}
