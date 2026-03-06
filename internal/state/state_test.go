package state

import (
	"path/filepath"
	"testing"
)

func TestLoadMissingFileReturnsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	d, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if d.Current != "" || d.Last != "" {
		t.Fatalf("expected empty state, got %#v", d)
	}
}

func TestSaveAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	in := Data{Current: "sandbox", Last: "utility"}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if out != in {
		t.Fatalf("roundtrip mismatch: got %#v want %#v", out, in)
	}
}
