package app

import (
	"strings"
	"testing"
)

func TestUpsertSnippetAppend(t *testing.T) {
	res := upsertSnippet("export PATH=\"$HOME/bin:$PATH\"\n", GenerateZshSnippet("/tmp/awsctx"), zshStartMarker, zshEndMarker)
	if res == "" {
		t.Fatal("expected output")
	}
	if !strings.Contains(res, zshStartMarker) || !strings.Contains(res, zshEndMarker) {
		t.Fatal("missing markers")
	}
}

func TestUpsertSnippetReplace(t *testing.T) {
	orig := "line1\n" + GenerateZshSnippet("/old/bin/awsctx") + "\nline2\n"
	res := upsertSnippet(orig, GenerateZshSnippet("/new/bin/awsctx"), zshStartMarker, zshEndMarker)
	if strings.Contains(res, "/old/bin/awsctx") {
		t.Fatal("old path should be replaced")
	}
	if !strings.Contains(res, "/new/bin/awsctx") {
		t.Fatal("new path missing")
	}
}

func TestGenerateBashSnippet(t *testing.T) {
	s := GenerateBashSnippet("/tmp/awsctx")
	if !strings.Contains(s, "awsctx() {") {
		t.Fatal("expected bash function")
	}
	if !strings.Contains(s, bashStartMarker) || !strings.Contains(s, bashEndMarker) {
		t.Fatal("expected markers in bash snippet")
	}
}

func TestGenerateFishSnippet(t *testing.T) {
	s := GenerateFishSnippet("/tmp/awsctx")
	if !strings.Contains(s, "function awsctx") {
		t.Fatal("expected fish function")
	}
	if !strings.Contains(s, fishStartMarker) || !strings.Contains(s, fishEndMarker) {
		t.Fatal("expected markers in fish snippet")
	}
}
