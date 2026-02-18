package app

import (
	"os"
	"path/filepath"
	"testing"

	"awsctx/internal/config"
)

func TestConfigureStatic(t *testing.T) {
	dir := t.TempDir()
	settings := config.Settings{
		CredentialsFile: filepath.Join(dir, "credentials"),
		ConfigFile:      filepath.Join(dir, "config"),
		StateFile:       filepath.Join(dir, "state.json"),
		FZFCommand:      "fzf",
	}
	a := New(settings, filepath.Join(dir, "awsctx.json"))

	err := a.ConfigureStatic(StaticProfileInput{
		Profile:         "dev",
		AccessKeyID:     "AKIA_TEST",
		SecretAccessKey: "SECRET_TEST",
		Region:          "us-east-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	creds, err := os.ReadFile(settings.CredentialsFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(creds) == "" {
		t.Fatal("expected credentials file to be written")
	}
}

func TestConfigureSSO(t *testing.T) {
	dir := t.TempDir()
	settings := config.Settings{
		CredentialsFile: filepath.Join(dir, "credentials"),
		ConfigFile:      filepath.Join(dir, "config"),
		StateFile:       filepath.Join(dir, "state.json"),
		FZFCommand:      "fzf",
	}
	a := New(settings, filepath.Join(dir, "awsctx.json"))

	err := a.ConfigureSSO(SSOProfileInput{
		Profile:   "eng",
		StartURL:  "https://example.awsapps.com/start",
		SSORegion: "us-east-1",
		AccountID: "123456789012",
		RoleName:  "Admin",
	})
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := os.ReadFile(settings.ConfigFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(cfg) == "" {
		t.Fatal("expected config file to be written")
	}
}

func TestUseWithAutoSSOLogin(t *testing.T) {
	dir := t.TempDir()
	settings := config.Settings{
		CredentialsFile: filepath.Join(dir, "credentials"),
		ConfigFile:      filepath.Join(dir, "config"),
		StateFile:       filepath.Join(dir, "state.json"),
		FZFCommand:      "fzf",
		AutoSSOLogin:    true,
	}
	if err := os.WriteFile(settings.CredentialsFile, []byte("[dev]\naws_access_key_id = A\naws_secret_access_key = B\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings.ConfigFile, []byte("[profile sandbox]\nsso_start_url = https://example.awsapps.com/start\nsso_region = us-east-1\nsso_role_name = Admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	a := New(settings, filepath.Join(dir, "awsctx.json"))
	var loggedInProfile string
	a.LoginRunner = func(profile string) error {
		loggedInProfile = profile
		return nil
	}

	if err := a.UseWithLogin("dev", false); err != nil {
		t.Fatal(err)
	}
	if loggedInProfile != "" {
		t.Fatalf("expected no login for static profile, got %q", loggedInProfile)
	}

	if err := a.UseWithLogin("sandbox", false); err != nil {
		t.Fatal(err)
	}
	if loggedInProfile != "sandbox" {
		t.Fatalf("expected login for sandbox profile, got %q", loggedInProfile)
	}
}
