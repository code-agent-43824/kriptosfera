package config

import (
	"embed"
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
)

// Payload mode identifiers used in RuntimePayloadConfig.Mode.
const (
	PayloadModeEmbedded = "embedded"
	PayloadModeRemote   = "remote"
)

// RuntimePayloadConfig selects and describes the payload for a build. URL,
// SHA256, and Size are only used in remote mode.
type RuntimePayloadConfig struct {
	Mode    string `json:"mode"`
	Version string `json:"version"`
	URL     string `json:"url,omitempty"`
	SHA256  string `json:"sha256,omitempty"`
	Size    int64  `json:"size,omitempty"`
}

// RuntimeConfig is the build-time configuration embedded in the launcher binary.
type RuntimeConfig struct {
	ProductName string               `json:"productName"`
	Version     string               `json:"version"`
	Payload     RuntimePayloadConfig `json:"payload"`
}

//go:embed app-version.txt runtime-config.json
var configFiles embed.FS

// DefaultRuntimeConfig returns the embedded runtime configuration, falling back
// to app-version.txt (embedded mode) when runtime-config.json is absent and
// filling in sensible defaults for any missing fields.
func DefaultRuntimeConfig() (RuntimeConfig, error) {
	rawConfig, err := configFiles.ReadFile("runtime-config.json")
	if err == nil {
		var cfg RuntimeConfig
		if err := json.Unmarshal(rawConfig, &cfg); err != nil {
			return RuntimeConfig{}, err
		}
		if cfg.ProductName == "" {
			cfg.ProductName = "Kriptosfera Demo"
		}
		if cfg.Version == "" {
			cfg.Version = cfg.Payload.Version
		}
		if cfg.Payload.Version == "" {
			cfg.Payload.Version = cfg.Version
		}
		if cfg.Payload.Mode == "" {
			cfg.Payload.Mode = PayloadModeEmbedded
		}
		return cfg, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return RuntimeConfig{}, err
	}

	raw, err := configFiles.ReadFile("app-version.txt")
	if err != nil {
		return RuntimeConfig{}, err
	}
	version := strings.TrimSpace(string(raw))
	return RuntimeConfig{
		ProductName: "Kriptosfera Demo",
		Version:     version,
		Payload: RuntimePayloadConfig{
			Mode:    PayloadModeEmbedded,
			Version: version,
		},
	}, nil
}
