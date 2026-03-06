package main

import (
	"os"
	"path/filepath"
	"testing"

	"awsctx/internal/app"
	"awsctx/internal/config"
)

func TestParseOptionalBool(t *testing.T) {
	tests := []struct {
		in      string
		want    *bool
		wantErr bool
	}{
		{in: "", want: nil, wantErr: false},
		{in: "true", want: boolPtr(true), wantErr: false},
		{in: "false", want: boolPtr(false), wantErr: false},
		{in: "nope", want: nil, wantErr: true},
	}

	for _, tc := range tests {
		got, err := parseOptionalBool(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Fatalf("expected error for input %q", tc.in)
			}
			continue
		}
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", tc.in, err)
		}
		if (got == nil) != (tc.want == nil) {
			t.Fatalf("nil mismatch for input %q", tc.in)
		}
		if got != nil && *got != *tc.want {
			t.Fatalf("unexpected value for input %q: got %v want %v", tc.in, *got, *tc.want)
		}
	}
}

func TestRunHelpFlag(t *testing.T) {
	if err := run([]string{"--help"}); err != nil {
		t.Fatalf("expected nil for --help, got %v", err)
	}
}

func TestRunUnknownCommand(t *testing.T) {
	err := run([]string{"nope"})
	if err == nil {
		t.Fatal("expected unknown command error")
	}
}

func TestRunUseArgumentParsing(t *testing.T) {
	a := makeTestApp(t)

	if err := runUse(a, []string{"sandbox"}); err != nil {
		t.Fatalf("expected success: %v", err)
	}
	if err := runUse(a, []string{"--login", "sandbox"}); err != nil {
		t.Fatalf("expected success with --login: %v", err)
	}
	if err := runUse(a, []string{"--bad", "sandbox"}); err == nil {
		t.Fatal("expected error for bad flag")
	}
}

func makeTestApp(t *testing.T) *app.App {
	t.Helper()
	dir := t.TempDir()
	settings := config.Settings{
		CredentialsFile: filepath.Join(dir, "credentials"),
		ConfigFile:      filepath.Join(dir, "config"),
		StateFile:       filepath.Join(dir, "state.json"),
		FZFCommand:      "fzf",
	}
	if err := osWriteFile(settings.CredentialsFile, []byte("[sandbox]\naws_access_key_id = A\naws_secret_access_key = B\n")); err != nil {
		t.Fatal(err)
	}
	return app.New(settings, filepath.Join(dir, "awsctx.json"))
}

func boolPtr(v bool) *bool { return &v }

func osWriteFile(path string, b []byte) error {
	return os.WriteFile(path, b, 0o600)
}
