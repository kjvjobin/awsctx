package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	zshStartMarker  = "# >>> awsctx >>>"
	zshEndMarker    = "# <<< awsctx <<<"
	bashStartMarker = "# >>> awsctx >>>"
	bashEndMarker   = "# <<< awsctx <<<"
	fishStartMarker = "# >>> awsctx >>>"
	fishEndMarker   = "# <<< awsctx <<<"
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
	return installSnippet(zshrcPath, GenerateZshSnippet(binPath), zshStartMarker, zshEndMarker)
}

func GenerateBashSnippet(binPath string) string {
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
`, bashStartMarker, quoted, bashEndMarker)
}

func InstallBashSnippet(bashrcPath, binPath string) error {
	return installSnippet(bashrcPath, GenerateBashSnippet(binPath), bashStartMarker, bashEndMarker)
}

func GenerateFishSnippet(binPath string) string {
	quoted := shellQuote(binPath)
	return fmt.Sprintf(`%s
set -gx AWSCTX_BIN %s
function awsctx
  command "$AWSCTX_BIN" $argv
  or return $status
  if test (count $argv) -eq 0; or contains -- $argv[1] use toggle fzf
    set -l env_out (command "$AWSCTX_BIN" env 2>/dev/null)
    if string match -q "export AWS_PROFILE=*" -- $env_out
      set -l profile (string replace -r '^export AWS_PROFILE=' '' -- $env_out)
      set profile (string trim -c '"' -- $profile)
      set -gx AWS_PROFILE $profile
    else
      set -e AWS_PROFILE
    end
  end
end
%s
`, fishStartMarker, quoted, fishEndMarker)
}

func InstallFishSnippet(fishConfigPath, binPath string) error {
	return installSnippet(fishConfigPath, GenerateFishSnippet(binPath), fishStartMarker, fishEndMarker)
}

func installSnippet(path, snippet, startMarker, endMarker string) error {
	existingBytes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}

	existing := string(existingBytes)
	updated := upsertSnippet(existing, snippet, startMarker, endMarker)

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create shell config dir: %w", err)
	}
	return os.WriteFile(path, []byte(updated), 0o600)
}

func upsertSnippet(existing, snippet, startMarker, endMarker string) string {
	start := strings.Index(existing, startMarker)
	end := strings.Index(existing, endMarker)
	if start >= 0 && end > start {
		end += len(endMarker)
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
