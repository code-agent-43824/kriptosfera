package config

import (
	"embed"
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

//go:embed app-version.txt
var versionFile embed.FS

func DefaultRuntimeConfig() (RuntimeConfig, error) {
	raw, err := versionFile.ReadFile("app-version.txt")
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
