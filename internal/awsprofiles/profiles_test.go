package awsprofiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverProfiles(t *testing.T) {
	dir := t.TempDir()
	creds := filepath.Join(dir, "credentials")
	cfg := filepath.Join(dir, "config")

	if err := os.WriteFile(creds, []byte("[default]\naws_access_key_id = a\n\n[dev]\naws_access_key_id = b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("[profile dev]\nregion = us-east-1\n\n[profile sso]\nsso_start_url = https://example.awsapps.com/start\nsso_region = us-east-1\nsso_role_name = Admin\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	profiles, err := Discover(creds, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(profiles))
	}

	if profiles[2].Name != "sso" || !profiles[2].IsSSO {
		t.Fatalf("expected sso profile detected: %+v", profiles[2])
	}
}

func TestWriteINI(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "creds")
	data := FileData{
		"dev": Section{"aws_access_key_id": "A", "aws_secret_access_key": "B"},
	}
	if err := WriteINI(p, data); err != nil {
		t.Fatal(err)
	}
	roundtrip, err := ParseINI(p)
	if err != nil {
		t.Fatal(err)
	}
	if roundtrip["dev"]["aws_access_key_id"] != "A" {
		t.Fatalf("unexpected value: %+v", roundtrip)
	}
}
