package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/code-agent-43824/kriptosfera/internal/logging"
)

const (
	// cryptoProTrustedSitesKeyPath is the documented per-user location of the
	// CAdES Browser Plug-in trusted-sites list (under HKCU). Sites on this list
	// do not trigger the plug-in's per-operation confirmation dialog. See
	// https://docs.cryptopro.ru/cades/plugin (plugin-safety).
	cryptoProTrustedSitesKeyPath   = `Software\Crypto Pro\CAdESplugin`
	cryptoProTrustedSitesValueName = "TrustedSites"
	trustedSitesStateFile          = ".cryptopro-trusted-sites-state.json"
)

// writeCryptoProTrustedSites is swapped in tests; the default is the
// platform-specific REG_MULTI_SZ writer (HKCU, per-user, no admin).
var writeCryptoProTrustedSites = writeCryptoProTrustedSitesRegistry

// TrustedSitesResult describes what the launcher did with the CryptoPro CAdES
// plug-in trusted-sites list.
type TrustedSitesResult struct {
	Sites       []string
	RegistryKey string
	Written     bool
	Reused      bool
	Skipped     bool
}

type trustedSitesState struct {
	Sites []string `json:"sites"`
}

// PrepareCryptoProTrustedSites writes the configured trusted sites to the
// per-user CryptoPro CAdES plug-in registry value
// (HKCU\Software\Crypto Pro\CAdESplugin\TrustedSites, REG_MULTI_SZ). The write is
// skipped when the list is empty and is reused when nothing changed since the
// last run (gated by a state file), mirroring native-messaging registration.
func PrepareCryptoProTrustedSites(appDir string, sites []string, logger *logging.Logger) (TrustedSitesResult, error) {
	normalized := normalizeTrustedSites(sites)
	result := TrustedSitesResult{
		Sites:       normalized,
		RegistryKey: "HKCU\\" + cryptoProTrustedSitesKeyPath,
	}
	if len(normalized) == 0 {
		logger.Info("cryptopro trusted sites skipped: none configured")
		result.Skipped = true
		return result, nil
	}
	if state, err := loadTrustedSitesState(appDir); err == nil && equalStringSlices(state.Sites, normalized) {
		logger.Info("cryptopro trusted sites reused count=%d key=%s", len(normalized), result.RegistryKey)
		result.Reused = true
		return result, nil
	}
	if err := writeCryptoProTrustedSites(cryptoProTrustedSitesKeyPath, cryptoProTrustedSitesValueName, normalized); err != nil {
		return TrustedSitesResult{}, err
	}
	if err := writeTrustedSitesState(appDir, trustedSitesState{Sites: normalized}); err != nil {
		return TrustedSitesResult{}, err
	}
	result.Written = true
	logger.Info("cryptopro trusted sites written count=%d key=%s", len(normalized), result.RegistryKey)
	return result, nil
}

// normalizeTrustedSites trims, drops empties, and de-duplicates while keeping the
// configured order stable.
func normalizeTrustedSites(sites []string) []string {
	seen := make(map[string]bool, len(sites))
	out := make([]string, 0, len(sites))
	for _, s := range sites {
		s = strings.TrimSpace(s)
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func loadTrustedSitesState(appDir string) (trustedSitesState, error) {
	var st trustedSitesState
	data, err := os.ReadFile(filepath.Join(appDir, trustedSitesStateFile))
	if err != nil {
		return st, err
	}
	if err := json.Unmarshal(data, &st); err != nil {
		return st, err
	}
	return st, nil
}

func writeTrustedSitesState(appDir string, st trustedSitesState) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(appDir, trustedSitesStateFile), append(data, '\n'), 0o644)
}
