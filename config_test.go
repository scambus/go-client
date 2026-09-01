package scambus

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCLIConfig(t *testing.T, contents string) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".scambus")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestResolveAPIURLPriority(t *testing.T) {
	writeCLIConfig(t, `{"api_url":"https://config.test"}`)
	t.Setenv("SCAMBUS_API_URL", "https://env.test")

	cfg := loadCLIConfig()
	if got := resolveAPIURL("https://explicit.test/", cfg); got != "https://explicit.test" {
		t.Fatalf("explicit wins: got %q", got)
	}
	if got := resolveAPIURL("", cfg); got != "https://env.test" {
		t.Fatalf("env wins over config: got %q", got)
	}

	t.Setenv("SCAMBUS_API_URL", "")
	t.Setenv("SCAMBUS_URL", "")
	if got := resolveAPIURL("", cfg); got != "https://config.test" {
		t.Fatalf("config file: got %q", got)
	}
}

func TestResolveAPIURLFallsBackToDefault(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SCAMBUS_API_URL", "")
	t.Setenv("SCAMBUS_URL", "")
	if got := resolveAPIURL("", loadCLIConfig()); got != DefaultAPIURL {
		t.Fatalf("got %q", got)
	}
}

func TestResolveAPIURLUsesLegacyEnvVar(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SCAMBUS_API_URL", "")
	t.Setenv("SCAMBUS_URL", "http://localhost:8080")
	if got := resolveAPIURL("", loadCLIConfig()); got != "http://localhost:8080" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveTokenPrefersNestedAuthToken(t *testing.T) {
	writeCLIConfig(t, `{"auth":{"token":"device-flow"},"jwt_token":"jwt","token":"legacy"}`)
	t.Setenv("SCAMBUS_API_TOKEN", "")

	if got := resolveToken("", loadCLIConfig()); got != "device-flow" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveTokenFallsThroughConfigKeys(t *testing.T) {
	writeCLIConfig(t, `{"jwt_token":"jwt","token":"legacy"}`)
	t.Setenv("SCAMBUS_API_TOKEN", "")
	if got := resolveToken("", loadCLIConfig()); got != "jwt" {
		t.Fatalf("got %q", got)
	}

	writeCLIConfig(t, `{"token":"legacy"}`)
	if got := resolveToken("", loadCLIConfig()); got != "legacy" {
		t.Fatalf("got %q", got)
	}
}

func TestLoadCLIConfigToleratesBadJSON(t *testing.T) {
	writeCLIConfig(t, `{ not json`)
	if cfg := loadCLIConfig(); cfg.APIURL != "" || cfg.Token != "" {
		t.Fatalf("got %+v", cfg)
	}
}

func TestNewReadsCredentialsFromEnvironment(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("SCAMBUS_API_KEY_ID", "env-key")
	t.Setenv("SCAMBUS_API_KEY_SECRET", "env-secret")

	c, err := New(WithAPIURL("https://example.test/api"))
	if err != nil {
		t.Fatal(err)
	}
	if c.authHeader != [2]string{"X-API-Key", "env-key:env-secret"} {
		t.Fatalf("got %v", c.authHeader)
	}
}
