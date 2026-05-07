package config

import (
    "encoding/json"
    "os"
)

type AppConfig struct {
    AppID              string   `json:"appId"`
    ProductName        string   `json:"productName"`
    Version            string   `json:"version"`
    StartURL           string   `json:"startUrl"`
    AllowedOrigins     []string `json:"allowedOrigins"`
    ProfileName        string   `json:"profileName"`
    DiagnosticsEnabled bool     `json:"diagnosticsEnabled"`
    ChromiumArgs       []string `json:"chromiumArgs"`
}

func Load(path string) (AppConfig, error) {
    var cfg AppConfig
    data, err := os.ReadFile(path)
    if err != nil {
        return cfg, err
    }
    err = json.Unmarshal(data, &cfg)
    return cfg, err
}
