package bootstrap

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

const (
	cryptoProPluginComponent = "cryptopro-browser-plugin"
	cryptoProPluginVersion   = "2.0.15700"
	cryptoProPluginLayout    = 1
	cryptoProPluginStateFile = ".cryptopro-plugin-state.json"
	cryptoProPluginReadyFile = ".cryptopro-plugin-ready"
)

var requiredCryptoProPluginFiles = []string{
	"Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.exe",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/nmcades.json",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/npcades.dll",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/cades.dll",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/xades.dll",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/cplib.dll",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi10.dll",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/capi20.dll",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpcspi.dll",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpsuprt.dll",
	"Program Files/Crypto Pro/CAdES Browser Plug-in/Mini CSP/cpui.dll",
}

type CryptoProPluginState struct {
	Component     string `json:"component"`
	Version       string `json:"version"`
	SHA256        string `json:"sha256"`
	LayoutVersion int    `json:"layoutVersion"`
}

type ComponentPrepareResult struct {
	Path    string
	Reused  bool
	Skipped bool
}

type CryptoProPluginManager struct {
	Bundle        []byte
	Version       string
	SHA256        string
	LayoutVersion int
}

func NewEmbeddedCryptoProPluginManager() CryptoProPluginManager {
	info := embeddedCryptoProPluginInfo()
	return CryptoProPluginManager{
		Bundle:        embeddedCryptoProPlugin,
		Version:       cryptoProPluginVersion,
		SHA256:        info.SHA256,
		LayoutVersion: cryptoProPluginLayout,
	}
}

func (m CryptoProPluginManager) Prepare(appDir string, logger *logging.Logger, progress ProgressReporter) (ComponentPrepareResult, error) {
	targetDir := filepath.Join(appDir, "cryptopro", "plugin")
	if len(m.Bundle) == 0 {
		if runtime.GOOS == "windows" {
			return ComponentPrepareResult{}, errors.New("embedded CryptoPro plugin bundle is empty")
		}
		logger.Info("cryptopro plugin extraction skipped: bundle not embedded for os=%s", runtime.GOOS)
		return ComponentPrepareResult{Path: targetDir, Skipped: true}, nil
	}
	if m.Version == "" {
		return ComponentPrepareResult{}, errors.New("embedded CryptoPro plugin version is empty")
	}
	if m.SHA256 == "" {
		return ComponentPrepareResult{}, errors.New("embedded CryptoPro plugin sha256 is empty")
	}
	if m.LayoutVersion == 0 {
		return ComponentPrepareResult{}, errors.New("embedded CryptoPro plugin layout version is empty")
	}

	// Fast path: reuse an already-prepared bundle without taking the lock.
	if prepared, err := m.isPrepared(appDir, targetDir); err != nil {
		return ComponentPrepareResult{}, err
	} else if prepared {
		logger.Info("cryptopro plugin already prepared path=%s version=%s", targetDir, m.Version)
		return ComponentPrepareResult{Path: targetDir, Reused: true}, nil
	}

	unlock, err := acquireLock(appDir)
	if err != nil {
		return ComponentPrepareResult{}, err
	}
	defer unlock()

	// Re-check under the lock in case a concurrent launch just prepared it.
	if prepared, err := m.isPrepared(appDir, targetDir); err != nil {
		return ComponentPrepareResult{}, err
	} else if prepared {
		logger.Info("cryptopro plugin already prepared path=%s version=%s", targetDir, m.Version)
		return ComponentPrepareResult{Path: targetDir, Reused: true}, nil
	}

	if progress != nil {
		progress.SetStatus("Распаковка CryptoPro Browser Plugin...")
	}
	logger.Info("extract cryptopro plugin path=%s version=%s sha256=%s", targetDir, m.Version, m.SHA256)

	parentDir := filepath.Dir(targetDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return ComponentPrepareResult{}, err
	}
	tempDir, err := os.MkdirTemp(parentDir, "plugin-staging-")
	if err != nil {
		return ComponentPrepareResult{}, err
	}
	defer os.RemoveAll(tempDir)

	if err := unzipCryptoProPlugin(bytes.NewReader(m.Bundle), int64(len(m.Bundle)), tempDir); err != nil {
		return ComponentPrepareResult{}, fmt.Errorf("extract cryptopro plugin: %w", err)
	}
	if err := validateCryptoProPluginLayout(tempDir); err != nil {
		return ComponentPrepareResult{}, err
	}
	if err := os.RemoveAll(targetDir); err != nil {
		return ComponentPrepareResult{}, err
	}
	if err := os.Rename(tempDir, targetDir); err != nil {
		return ComponentPrepareResult{}, err
	}
	if err := writeCryptoProPluginState(appDir, CryptoProPluginState{
		Component:     cryptoProPluginComponent,
		Version:       m.Version,
		SHA256:        m.SHA256,
		LayoutVersion: m.LayoutVersion,
	}); err != nil {
		return ComponentPrepareResult{}, err
	}
	if err := os.WriteFile(filepath.Join(appDir, cryptoProPluginReadyFile), []byte("ok\n"), 0o644); err != nil {
		return ComponentPrepareResult{}, err
	}

	return ComponentPrepareResult{Path: targetDir}, nil
}

func (m CryptoProPluginManager) isPrepared(appDir, targetDir string) (bool, error) {
	if _, err := os.Stat(filepath.Join(appDir, cryptoProPluginReadyFile)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	state, err := loadCryptoProPluginState(appDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if state.Component != cryptoProPluginComponent || state.Version != m.Version || state.SHA256 != m.SHA256 || state.LayoutVersion != m.LayoutVersion {
		return false, nil
	}
	if err := validateCryptoProPluginLayout(targetDir); err != nil {
		return false, nil
	}
	return true, nil
}

func validateCryptoProPluginLayout(root string) error {
	// Collect every required suffix and clear it as we find a matching file in a
	// single directory walk, instead of walking the tree once per required file.
	remaining := make(map[string]bool, len(requiredCryptoProPluginFiles))
	for _, required := range requiredCryptoProPluginFiles {
		remaining[filepath.ToSlash(filepath.Clean(required))] = true
	}

	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relSlash := filepath.ToSlash(rel)
		for suffix := range remaining {
			if strings.HasSuffix(relSlash, suffix) {
				delete(remaining, suffix)
			}
		}
		if len(remaining) == 0 {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(remaining) > 0 {
		missing := make([]string, 0, len(remaining))
		for suffix := range remaining {
			missing = append(missing, suffix)
		}
		sort.Strings(missing)
		return fmt.Errorf("cryptopro plugin required files not found: %s", strings.Join(missing, ", "))
	}
	return nil
}

func findFileBySlashSuffix(root, suffix string) (string, error) {
	normalizedSuffix := filepath.ToSlash(filepath.Clean(suffix))
	var found string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if strings.HasSuffix(filepath.ToSlash(rel), normalizedSuffix) {
			found = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if found == "" {
		return "", fmt.Errorf("cryptopro plugin required file not found: %s", suffix)
	}
	return found, nil
}

func loadCryptoProPluginState(root string) (CryptoProPluginState, error) {
	var state CryptoProPluginState
	data, err := os.ReadFile(filepath.Join(root, cryptoProPluginStateFile))
	if err != nil {
		return state, err
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return state, err
	}
	return state, nil
}

func writeCryptoProPluginState(root string, state CryptoProPluginState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, cryptoProPluginStateFile), append(data, '\n'), 0o644)
}
