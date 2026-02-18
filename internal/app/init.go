package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	zshStartMarker = "# >>> awsctx >>>"
	zshEndMarker   = "# <<< awsctx <<<"
)

func GenerateZshSnippet(binPath string) string {
	quoted := shellQuote(binPath)
	return fmt.Sprintf(`%s
AWSCTX_BIN=%s
awsctx() {
  "$AWSCTX_BIN" "$@" || return $?
  if [ "$#" -eq 0 ] || [ "$1" = "use" ] || [ "$1" = "toggle" ] || [ "$1" = "fzf" ]; then
    eval "$("$AWSCTX_BIN" env)"
  fi
}
%s
`, zshStartMarker, quoted, zshEndMarker)
}

func InstallZshSnippet(zshrcPath, binPath string) error {
	existingBytes, err := os.ReadFile(zshrcPath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", zshrcPath, err)
	}

	snippet := GenerateZshSnippet(binPath)
	existing := string(existingBytes)
	updated := upsertSnippet(existing, snippet)

	if err := os.MkdirAll(filepath.Dir(zshrcPath), 0o700); err != nil {
		return fmt.Errorf("create zsh dir: %w", err)
	}
	return os.WriteFile(zshrcPath, []byte(updated), 0o600)
}

func upsertSnippet(existing, snippet string) string {
	start := strings.Index(existing, zshStartMarker)
	end := strings.Index(existing, zshEndMarker)
	if start >= 0 && end > start {
		end += len(zshEndMarker)
		prefix := strings.TrimRight(existing[:start], "\n")
		suffix := strings.TrimLeft(existing[end:], "\n")
		parts := []string{}
		if prefix != "" {
			parts = append(parts, prefix)
		}
		parts = append(parts, strings.TrimRight(snippet, "\n"))
		if suffix != "" {
			parts = append(parts, suffix)
		}
		return strings.Join(parts, "\n\n") + "\n"
	}

	existing = strings.TrimRight(existing, "\n")
	if existing == "" {
		return snippet
	}
	return existing + "\n\n" + snippet
}

func shellQuote(v string) string {
	return "'" + strings.ReplaceAll(v, "'", "'\\''") + "'"
}
