package config

import (
	"encoding/json"
	"os"
)

// AppConfig is the product-facing configuration shipped inside the payload as
// config/app-config.json. It is validated by the launcher before use.
type AppConfig struct {
	AppID              string   `json:"appId"`
	ProductName        string   `json:"productName"`
	CustomerName       string   `json:"customerName"`
	Version            string   `json:"version"`
	StartURL           string   `json:"startUrl"`
	AllowedOrigins     []string `json:"allowedOrigins"`
	ProfileName        string   `json:"profileName"`
	WindowMode         string   `json:"windowMode"`
	DiagnosticsEnabled bool     `json:"diagnosticsEnabled"`
	DiagnosticsURL     string   `json:"diagnosticsUrl,omitempty"`
	ChromiumArgs       []string `json:"chromiumArgs"`
	// TrustedSites are added to the CryptoPro CAdES plug-in trusted-sites list
	// (HKCU\Software\Crypto Pro\CAdESplugin\TrustedSites, REG_MULTI_SZ) so the
	// plug-in does not show its per-operation confirmation dialog for these
	// origins. Each entry is a "scheme://host" string; "*" wildcards are allowed
	// in the host (e.g. "https://*.cryptopro.ru"). Empty disables the feature.
	TrustedSites []string `json:"trustedSites,omitempty"`
}

// Load reads and decodes an AppConfig from the JSON file at path.
func Load(path string) (AppConfig, error) {
	var cfg AppConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	err = json.Unmarshal(data, &cfg)
	return cfg, err
}
