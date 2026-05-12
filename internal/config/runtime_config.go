package config

import (
	"encoding/json"
	"embed"
	"errors"
	"io/fs"
	"strings"
)

const (
	PayloadModeEmbedded = "embedded"
	PayloadModeRemote   = "remote"
)

type RuntimePayloadConfig struct {
	Mode    string `json:"mode"`
	Version string `json:"version"`
	URL     string `json:"url,omitempty"`
	SHA256  string `json:"sha256,omitempty"`
	Size    int64  `json:"size,omitempty"`
}

type RuntimeConfig struct {
	ProductName string               `json:"productName"`
	Version     string               `json:"version"`
	Payload     RuntimePayloadConfig `json:"payload"`
}

//go:embed app-version.txt runtime-config.json
var configFiles embed.FS

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
