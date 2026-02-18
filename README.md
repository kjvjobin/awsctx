# awsctx

`awsctx` is a Go CLI similar to `kubectx`, built to switch AWS CLI profiles quickly.

It supports:
- Plain profiles from `~/.aws/credentials`
- SSO profiles from `~/.aws/config`
- Interactive selection through `fzf`
- Local `awsctx` settings customization
- Writing/updating static and SSO profiles securely

## Build

```bash
go build -o bin/awsctx ./cmd/awsctx
```

## Install

```bash
mkdir -p "$HOME/.local/bin"
cp bin/awsctx "$HOME/.local/bin/awsctx"
```

Add to `PATH` (zsh):

```bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
source ~/.zshrc
```

## Commands

```bash
awsctx list
awsctx current
awsctx use <profile>
awsctx use <profile> --login
awsctx toggle
awsctx fzf
awsctx env
awsctx settings
awsctx init zsh
awsctx configure static ...
awsctx configure sso ...
```

## Shell integration

Install zsh integration once:

```bash
awsctx init zsh --write
source ~/.zshrc
```

Preview the snippet before writing:

```bash
awsctx init zsh
```

If you run from the repo binary directly:

```bash
bin/awsctx init zsh --write
source ~/.zshrc
```

Verify:

```bash
awsctx use sandbox
echo "$AWS_PROFILE"
aws sts get-caller-identity
```

Login on demand for SSO profiles:

```bash
awsctx use sandbox --login
```

## Configure awsctx behavior

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

Enable auto SSO login during `use`, `toggle`, and `fzf` profile switches:

```bash
awsctx settings --auto-sso-login true
```

## Configure profile entries

### Static credentials profile

```bash
awsctx configure static \
  --profile dev \
  --access-key-id AKIA... \
  --secret-access-key ... \
  --region us-east-1
```

### SSO profile

```bash
awsctx configure sso \
  --profile engineering \
  --sso-start-url https://example.awsapps.com/start \
  --sso-region us-east-1 \
  --sso-account-id 123456789012 \
  --sso-role-name Admin \
  --region us-east-1
```

## Security notes

- `awsctx` uses strict file permissions (`0700` dirs, `0600` files) for its own state/config and rewritten AWS files.
- Writes are atomic (`temp file + rename`) to reduce corruption risk.
- `awsctx use` validates profile names from AWS files before activation.
- Avoid passing secrets on CLI in shared environments because shell history may capture them; prefer env vars or prompt input.

## Limitations

- This tool rewrites INI files and does not preserve comments/order.
