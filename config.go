package scambus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const DefaultAPIURL = "https://scambus.net/api"

type cliConfig struct {
	APIURL   string `json:"api_url"`
	JWTToken string `json:"jwt_token"`
	Token    string `json:"token"`
	Auth     struct {
		Token string `json:"token"`
	} `json:"auth"`
}

func loadCLIConfig() cliConfig {
	var cfg cliConfig
	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}
	raw, err := os.ReadFile(filepath.Join(home, ".scambus", "config.json"))
	if err != nil {
		return cfg
	}
	_ = json.Unmarshal(raw, &cfg)
	return cfg
}

func resolveAPIURL(explicit string, cfg cliConfig) string {
	for _, candidate := range []string{
		explicit,
		os.Getenv("SCAMBUS_API_URL"),
		os.Getenv("SCAMBUS_URL"),
		cfg.APIURL,
		DefaultAPIURL,
	} {
		if candidate != "" {
			return strings.TrimRight(candidate, "/")
		}
	}
	return DefaultAPIURL
}

func resolveToken(explicit string, cfg cliConfig) string {
	for _, candidate := range []string{
		explicit,
		os.Getenv("SCAMBUS_API_TOKEN"),
		cfg.Auth.Token,
		cfg.JWTToken,
		cfg.Token,
	} {
		if candidate != "" {
			return candidate
		}
	}
	return ""
}
