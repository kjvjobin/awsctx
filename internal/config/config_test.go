package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsFromEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	cfg, path, err := Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if path == "" {
		t.Fatal("expected config path")
	}
	if !filepath.IsAbs(path) {
		t.Fatalf("expected absolute config path, got %q", path)
	}
	if path != configPath {
		t.Fatalf("unexpected config path: got %q want %q", path, configPath)
	}
	if cfg.CredentialsFile == "" || cfg.ConfigFile == "" || cfg.StateFile == "" {
		t.Fatalf("expected non-empty default paths: %#v", cfg)
	}
	if cfg.FZFCommand != "fzf" {
		t.Fatalf("unexpected fzf command: %q", cfg.FZFCommand)
	}
}

func TestDefaultPathFunctionsUseEnvOverrides(t *testing.T) {
	t.Setenv("AWS_SHARED_CREDENTIALS_FILE", "/tmp/test-credentials")
	t.Setenv("AWS_CONFIG_FILE", "/tmp/test-config")
	if got := defaultCredentialsPath(); got != "/tmp/test-credentials" {
		t.Fatalf("unexpected credentials default: %q", got)
	}
	if got := defaultAWSConfigPath(); got != "/tmp/test-config" {
		t.Fatalf("unexpected config default: %q", got)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "awsctx.json")
	in := Settings{
		CredentialsFile: "/tmp/creds",
		ConfigFile:      "/tmp/config",
		StateFile:       "/tmp/state",
		FZFCommand:      "fzf --height 40%",
		AutoSSOLogin:    true,
	}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, loadedPath, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if loadedPath != path {
		t.Fatalf("unexpected loaded path: %q", loadedPath)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch:\n got: %#v\nwant: %#v", out, in)
	}
}

func TestLoadFillsMissingFieldsFromDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	path := filepath.Join(dir, "cfg.json")
	if err := os.WriteFile(path, []byte(`{"credentials_file":"/tmp/custom-creds"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, _, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CredentialsFile != "/tmp/custom-creds" {
		t.Fatalf("unexpected credentials file: %q", cfg.CredentialsFile)
	}
	if cfg.ConfigFile == "" || cfg.StateFile == "" || cfg.FZFCommand == "" {
		t.Fatalf("expected defaults to be filled: %#v", cfg)
	}
}
