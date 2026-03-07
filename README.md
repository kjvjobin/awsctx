# awsctx

`awsctx` is a Go CLI similar to `kubectx`, built to switch AWS CLI profiles quickly.

It supports:
- Plain profiles from `~/.aws/credentials`
- SSO profiles from `~/.aws/config`
- Interactive selection through `fzf`
- Local `awsctx` settings customization
- Writing/updating static and SSO profiles securely

## Why awsctx?

- `kubectx`-style UX: running `awsctx` opens `fzf` profile selection directly.
- Built for both static credentials and AWS SSO profile flows.
- Includes profile management commands (`configure static` and `configure sso`), not just switching.
- Supports `use`, `toggle`, `current`, and shell export integration.
- Optional SSO auto-login with session-validity check to avoid unnecessary re-auth prompts.
- One-command shell setup via `awsctx init <zsh|bash|fish> --write`.
- Ships as cross-platform release binaries with checksums for direct install.

## Quick Start (GitHub Release)

Download the latest binary for your platform.

### macOS (Apple Silicon)

```bash
curl -L -o /tmp/awsctx https://github.com/kjvjobin/awsctx/releases/latest/download/awsctx_darwin_arm64
chmod +x /tmp/awsctx
sudo mv /tmp/awsctx /usr/local/bin/awsctx
```

### macOS (Intel)

```bash
curl -L -o /tmp/awsctx https://github.com/kjvjobin/awsctx/releases/latest/download/awsctx_darwin_amd64
chmod +x /tmp/awsctx
sudo mv /tmp/awsctx /usr/local/bin/awsctx
```

### Linux (x86_64)

```bash
curl -L -o /tmp/awsctx https://github.com/kjvjobin/awsctx/releases/latest/download/awsctx_linux_amd64
chmod +x /tmp/awsctx
sudo mv /tmp/awsctx /usr/local/bin/awsctx
```

Verify:

```bash
awsctx --help
```

## Shell Integration (Required for env switching)

`awsctx` can update its own state, but shell integration is needed to update `AWS_PROFILE` in your current shell session.

### zsh

```bash
awsctx init zsh --write
source ~/.zshrc
```

### bash

```bash
awsctx init bash --write
source ~/.bashrc
```

### fish

```bash
awsctx init fish --write
source ~/.config/fish/config.fish
```

Preview snippets without writing:

```bash
awsctx init zsh
awsctx init bash
awsctx init fish
```

## Usage

Core commands:

```bash
awsctx                 # open fzf picker
awsctx list            # list profiles
awsctx current         # show active profile
awsctx use <profile>   # switch directly
awsctx toggle          # switch to previous profile
awsctx env             # print export/unset command
```

SSO login on demand:

```bash
awsctx use <profile> --login
```

If `fzf` is not installed:

```bash
awsctx use <profile>
```

## Configure awsctx

Show current settings:

```bash
awsctx settings
```

Update paths or `fzf` command:

```bash
awsctx settings \
  --credentials-file ~/.aws/credentials \
  --aws-config-file ~/.aws/config \
  --state-file ~/.config/awsctx/state.json \
  --fzf-command "fzf --height 40%"
```

Enable automatic SSO login on switch:

```bash
awsctx settings --auto-sso-login true
```

## Configure AWS Profiles

Static credentials profile:

```bash
awsctx configure static \
  --profile dev \
  --access-key-id AKIA... \
  --secret-access-key ... \
  --region us-east-1
```

SSO profile:

```bash
awsctx configure sso \
  --profile engineering \
  --sso-start-url https://example.awsapps.com/start \
  --sso-region us-east-1 \
  --sso-account-id 123456789012 \
  --sso-role-name Admin \
  --region us-east-1
```

## Build from Source

```bash
go build -o bin/awsctx ./cmd/awsctx
```

Install built binary:

```bash
mkdir -p "$HOME/.local/bin"
cp bin/awsctx "$HOME/.local/bin/awsctx"
```

## Security Notes

- Uses restrictive file permissions (`0700` dirs, `0600` files) for awsctx-managed files.
- Uses atomic writes (`temp file + rename`) to reduce corruption risk.
- Validates profile names before activation.
- Prefer prompts or env vars over inline secrets to avoid shell-history leaks.

## Limitations

- Rewrites INI files and does not preserve original comments/order.
