package app

import (
	"strings"
	"testing"
)

func TestUpsertSnippetAppend(t *testing.T) {
	res := upsertSnippet("export PATH=\"$HOME/bin:$PATH\"\n", GenerateZshSnippet("/tmp/awsctx"))
	if res == "" {
		t.Fatal("expected output")
	}
	if !strings.Contains(res, zshStartMarker) || !strings.Contains(res, zshEndMarker) {
		t.Fatal("missing markers")
	}
}

func TestUpsertSnippetReplace(t *testing.T) {
	orig := "line1\n" + GenerateZshSnippet("/old/bin/awsctx") + "\nline2\n"
	res := upsertSnippet(orig, GenerateZshSnippet("/new/bin/awsctx"))
	if strings.Contains(res, "/old/bin/awsctx") {
		t.Fatal("old path should be replaced")
	}
	if !strings.Contains(res, "/new/bin/awsctx") {
		t.Fatal("new path missing")
	}
}
