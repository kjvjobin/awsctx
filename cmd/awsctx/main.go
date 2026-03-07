package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"awsctx/internal/app"
	"awsctx/internal/config"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	root := flag.NewFlagSet("awsctx", flag.ContinueOnError)
	root.SetOutput(os.Stderr)
	configPath := root.String("config", "", "Path to awsctx config file")
	if err := root.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage()
			return nil
		}
		return err
	}

	settings, resolvedConfigPath, err := config.Load(*configPath)
	if err != nil {
		return err
	}
	a := app.New(settings, resolvedConfigPath)

	rest := root.Args()
	if len(rest) == 0 {
		return a.UseFZF()
	}

	switch rest[0] {
	case "list", "ls":
		return a.List()
	case "current", "cur":
		return a.Current()
	case "env":
		return a.Env()
	case "use":
		return runUse(a, rest[1:])
	case "toggle", "t":
		return a.Toggle()
	case "fzf":
		return a.UseFZF()
	case "settings":
		return runSettings(a, rest[1:])
	case "configure":
		return runConfigure(a, rest[1:])
	case "init":
		return runInit(rest[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", rest[0])
	}
}

func runSettings(a *app.App, args []string) error {
	fs := flag.NewFlagSet("settings", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	credentialsFile := fs.String("credentials-file", "", "AWS credentials file path")
	configFile := fs.String("aws-config-file", "", "AWS config file path")
	stateFile := fs.String("state-file", "", "awsctx state file path")
	fzfCommand := fs.String("fzf-command", "", "fzf command (default: fzf)")
	autoSSOLogin := fs.String("auto-sso-login", "", "Enable/disable auto SSO login on profile switch (true|false)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *credentialsFile == "" && *configFile == "" && *stateFile == "" && *fzfCommand == "" && *autoSSOLogin == "" {
		a.ShowSettings()
		return nil
	}
	parsedAutoSSOLogin, err := parseOptionalBool(*autoSSOLogin)
	if err != nil {
		return err
	}
	return a.UpdateSettings(*credentialsFile, *configFile, *stateFile, *fzfCommand, parsedAutoSSOLogin)
}

func runConfigure(a *app.App, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: awsctx configure <static|sso> [flags]")
	}
	kind := args[0]
	switch kind {
	case "static":
		fs := flag.NewFlagSet("configure static", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		in := app.StaticProfileInput{}
		fs.StringVar(&in.Profile, "profile", "", "profile name")
		fs.StringVar(&in.AccessKeyID, "access-key-id", os.Getenv("AWS_ACCESS_KEY_ID"), "access key id")
		fs.StringVar(&in.SecretAccessKey, "secret-access-key", os.Getenv("AWS_SECRET_ACCESS_KEY"), "secret access key")
		fs.StringVar(&in.SessionToken, "session-token", os.Getenv("AWS_SESSION_TOKEN"), "session token")
		fs.StringVar(&in.Region, "region", "", "default region")
		fs.StringVar(&in.Output, "output", "", "default output format")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		var err error
		in.Profile, err = app.PromptIfMissing(os.Stdin, "profile", "Profile", in.Profile)
		if err != nil {
			return err
		}
		in.AccessKeyID, err = app.PromptIfMissing(os.Stdin, "access-key-id", "Access key ID", in.AccessKeyID)
		if err != nil {
			return err
		}
		in.SecretAccessKey, err = app.PromptIfMissing(os.Stdin, "secret-access-key", "Secret access key", in.SecretAccessKey)
		if err != nil {
			return err
		}
		return a.ConfigureStatic(in)
	case "sso":
		fs := flag.NewFlagSet("configure sso", flag.ContinueOnError)
		fs.SetOutput(os.Stderr)
		in := app.SSOProfileInput{}
		fs.StringVar(&in.Profile, "profile", "", "profile name")
		fs.StringVar(&in.StartURL, "sso-start-url", "", "sso start url")
		fs.StringVar(&in.SSORegion, "sso-region", "", "sso region")
		fs.StringVar(&in.AccountID, "sso-account-id", "", "sso account id")
		fs.StringVar(&in.RoleName, "sso-role-name", "", "sso role name")
		fs.StringVar(&in.Region, "region", "", "default region")
		fs.StringVar(&in.Output, "output", "", "default output format")
		fs.StringVar(&in.SessionName, "sso-session", "", "sso session name")
		if err := fs.Parse(args[1:]); err != nil {
			return err
		}
		var err error
		in.Profile, err = app.PromptIfMissing(os.Stdin, "profile", "Profile", in.Profile)
		if err != nil {
			return err
		}
		in.StartURL, err = app.PromptIfMissing(os.Stdin, "sso-start-url", "SSO start URL", in.StartURL)
		if err != nil {
			return err
		}
		in.SSORegion, err = app.PromptIfMissing(os.Stdin, "sso-region", "SSO region", in.SSORegion)
		if err != nil {
			return err
		}
		in.AccountID, err = app.PromptIfMissing(os.Stdin, "sso-account-id", "SSO account ID", in.AccountID)
		if err != nil {
			return err
		}
		in.RoleName, err = app.PromptIfMissing(os.Stdin, "sso-role-name", "SSO role name", in.RoleName)
		if err != nil {
			return err
		}
		return a.ConfigureSSO(in)
	default:
		return fmt.Errorf("unknown configure kind %q", kind)
	}
}

func runInit(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: awsctx init <zsh|bash|fish> [--write] [--file <path>]")
	}
	shell := args[0]
	fs := flag.NewFlagSet("init "+shell, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	write := fs.Bool("write", false, "Write/update snippet in shell config")
	targetFile := fs.String("file", "", "Target shell config file")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}

	var snippet string
	var install func(path, binPath string) error
	var defaultPath string
	switch shell {
	case "zsh":
		snippet = app.GenerateZshSnippet(exePath)
		install = app.InstallZshSnippet
		defaultPath = home + "/.zshrc"
	case "bash":
		snippet = app.GenerateBashSnippet(exePath)
		install = app.InstallBashSnippet
		defaultPath = home + "/.bashrc"
	case "fish":
		snippet = app.GenerateFishSnippet(exePath)
		install = app.InstallFishSnippet
		defaultPath = home + "/.config/fish/config.fish"
	default:
		return fmt.Errorf("unsupported shell %q (supported: zsh, bash, fish)", shell)
	}

	if !*write {
		fmt.Print(snippet)
		return nil
	}
	path := *targetFile
	if path == "" {
		path = defaultPath
	}
	if err := install(path, exePath); err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "installed awsctx %s integration in %s\n", shell, path)
	return nil
}

func runUse(a *app.App, args []string) error {
	var profile string
	login := false
	for _, arg := range args {
		switch arg {
		case "--login":
			login = true
		default:
			if strings.HasPrefix(arg, "-") {
				return fmt.Errorf("unknown flag %q", arg)
			}
			if profile != "" {
				return errors.New("usage: awsctx use <profile> [--login]")
			}
			profile = arg
		}
	}
	if profile == "" {
		return errors.New("usage: awsctx use <profile> [--login]")
	}
	return a.UseWithLogin(profile, login)
}

func parseOptionalBool(v string) (*bool, error) {
	if v == "" {
		return nil, nil
	}
	switch v {
	case "true":
		b := true
		return &b, nil
	case "false":
		b := false
		return &b, nil
	default:
		return nil, fmt.Errorf("invalid value for --auto-sso-login: %q (expected true|false)", v)
	}
}

func usage() {
	fmt.Print(`awsctx - switch AWS CLI profiles quickly

Commands:
  list|ls                 List discovered profiles
  current|cur             Print active awsctx profile
  env                     Print shell export/unset for AWS_PROFILE
  use <profile>           Activate profile
  toggle|t                Switch back to previous profile
  fzf                     Pick profile using fzf
  settings                Show current awsctx settings
  settings [flags]        Update awsctx settings
  init <shell>            Print/install shell wrapper (zsh, bash, fish)
  configure static        Add/update static credentials profile
  configure sso           Add/update SSO profile in AWS config

Global flags:
  -config <path>          Path to awsctx config file

Examples:
  awsctx list
  awsctx use prod
  awsctx use prod --login
  eval "$(awsctx env)"
  awsctx fzf
  awsctx settings --fzf-command "fzf --height 40%" --auto-sso-login true
  awsctx init zsh --write
  awsctx init bash --write
  awsctx init fish --write
  awsctx configure static --profile dev --access-key-id AKIA... --secret-access-key ...
  awsctx configure sso --profile eng --sso-start-url https://example.awsapps.com/start --sso-region us-east-1 --sso-account-id 123456789012 --sso-role-name Admin
`)
}
