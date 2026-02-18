package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"awsctx/internal/awsprofiles"
	"awsctx/internal/config"
	"awsctx/internal/state"
)

type App struct {
	Settings    config.Settings
	ConfigPath  string
	Stdout      io.Writer
	Stderr      io.Writer
	Stdin       io.Reader
	LoginRunner func(profile string) error
}

func New(settings config.Settings, configPath string) *App {
	a := &App{
		Settings:   settings,
		ConfigPath: configPath,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
	}
	a.LoginRunner = a.runSSOLogin
	return a
}

func (a *App) List() error {
	profiles, err := awsprofiles.Discover(a.Settings.CredentialsFile, a.Settings.ConfigFile)
	if err != nil {
		return err
	}
	s, err := state.Load(a.Settings.StateFile)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(a.Stdout, 2, 2, 2, ' ', 0)
	fmt.Fprintln(w, "ACTIVE\tPROFILE\tTYPE\tSOURCE")
	for _, p := range profiles {
		kind := "static"
		if p.IsSSO {
			kind = "sso"
		}
		active := ""
		if p.Name == s.Current {
			active = "*"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", active, p.Name, kind, p.Source)
	}
	return w.Flush()
}

func (a *App) Current() error {
	s, err := state.Load(a.Settings.StateFile)
	if err != nil {
		return err
	}
	if s.Current == "" {
		return errors.New("no active profile")
	}
	fmt.Fprintf(a.Stdout, "Active Profile: %s\n", s.Current)
	return nil
}

func (a *App) Env() error {
	s, err := state.Load(a.Settings.StateFile)
	if err != nil {
		return err
	}
	if s.Current == "" {
		fmt.Fprintln(a.Stdout, "unset AWS_PROFILE")
		return nil
	}
	fmt.Fprintf(a.Stdout, "export AWS_PROFILE=%s\n", strconv.Quote(s.Current))
	return nil
}

func (a *App) Use(profile string) error {
	return a.UseWithLogin(profile, false)
}

func (a *App) UseWithLogin(profile string, forceLogin bool) error {
	if profile == "" {
		return errors.New("profile is required")
	}
	p, err := a.lookupProfile(profile)
	if err != nil {
		return err
	}

	s, err := state.Load(a.Settings.StateFile)
	if err != nil {
		return err
	}
	if s.Current != "" && s.Current != profile {
		s.Last = s.Current
	}
	s.Current = profile
	if err := state.Save(a.Settings.StateFile, s); err != nil {
		return err
	}
	if err := a.maybeLoginSSO(p, forceLogin || a.Settings.AutoSSOLogin); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "Active Profile: %s\n", profile)
	return nil
}

func (a *App) Toggle() error {
	s, err := state.Load(a.Settings.StateFile)
	if err != nil {
		return err
	}
	if s.Last == "" {
		return errors.New("no previous profile available")
	}
	p, err := a.lookupProfile(s.Last)
	if err != nil {
		return err
	}
	next := s.Last
	s.Last = s.Current
	s.Current = next
	if err := state.Save(a.Settings.StateFile, s); err != nil {
		return err
	}
	if err := a.maybeLoginSSO(p, a.Settings.AutoSSOLogin); err != nil {
		return err
	}
	fmt.Fprintf(a.Stdout, "Active Profile: %s\n", next)
	return nil
}

func (a *App) UseFZF() error {
	profiles, err := awsprofiles.Discover(a.Settings.CredentialsFile, a.Settings.ConfigFile)
	if err != nil {
		return err
	}
	if len(profiles) == 0 {
		return errors.New("no profiles found")
	}

	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	var b strings.Builder
	for _, p := range profiles {
		b.WriteString(p.Name)
		b.WriteByte('\n')
	}

	parts := strings.Fields(a.Settings.FZFCommand)
	if len(parts) == 0 {
		return errors.New("fzf_command cannot be empty")
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = strings.NewReader(b.String())
	cmd.Stderr = a.Stderr
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("fzf selection failed: %w", err)
	}
	choice := strings.TrimSpace(string(out))
	if choice == "" {
		return errors.New("no profile selected")
	}
	return a.UseWithLogin(choice, false)
}

func (a *App) ShowSettings() {
	fmt.Fprintf(a.Stdout, "config_path=%s\n", a.ConfigPath)
	fmt.Fprintf(a.Stdout, "credentials_file=%s\n", a.Settings.CredentialsFile)
	fmt.Fprintf(a.Stdout, "config_file=%s\n", a.Settings.ConfigFile)
	fmt.Fprintf(a.Stdout, "state_file=%s\n", a.Settings.StateFile)
	fmt.Fprintf(a.Stdout, "fzf_command=%s\n", a.Settings.FZFCommand)
	fmt.Fprintf(a.Stdout, "auto_sso_login=%t\n", a.Settings.AutoSSOLogin)
}

func (a *App) UpdateSettings(credentialsFile, configFile, stateFile, fzfCommand string, autoSSOLogin *bool) error {
	if credentialsFile != "" {
		a.Settings.CredentialsFile = expandPath(credentialsFile)
	}
	if configFile != "" {
		a.Settings.ConfigFile = expandPath(configFile)
	}
	if stateFile != "" {
		a.Settings.StateFile = expandPath(stateFile)
	}
	if fzfCommand != "" {
		a.Settings.FZFCommand = fzfCommand
	}
	if autoSSOLogin != nil {
		a.Settings.AutoSSOLogin = *autoSSOLogin
	}
	if err := config.Save(a.ConfigPath, a.Settings); err != nil {
		return err
	}
	a.ShowSettings()
	return nil
}

type StaticProfileInput struct {
	Profile         string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Region          string
	Output          string
}

type SSOProfileInput struct {
	Profile     string
	StartURL    string
	SSORegion   string
	AccountID   string
	RoleName    string
	Region      string
	Output      string
	SessionName string
}

func (a *App) ConfigureStatic(in StaticProfileInput) error {
	if in.Profile == "" {
		return errors.New("--profile is required")
	}
	if in.AccessKeyID == "" || in.SecretAccessKey == "" {
		return errors.New("--access-key-id and --secret-access-key are required")
	}

	credsPath := expandPath(a.Settings.CredentialsFile)
	creds, err := awsprofiles.ParseINI(credsPath)
	if err != nil {
		return err
	}
	if creds[in.Profile] == nil {
		creds[in.Profile] = awsprofiles.Section{}
	}
	creds[in.Profile]["aws_access_key_id"] = in.AccessKeyID
	creds[in.Profile]["aws_secret_access_key"] = in.SecretAccessKey
	if in.SessionToken != "" {
		creds[in.Profile]["aws_session_token"] = in.SessionToken
	}
	if err := awsprofiles.WriteINI(credsPath, creds); err != nil {
		return err
	}

	if in.Region != "" || in.Output != "" {
		cfgPath := expandPath(a.Settings.ConfigFile)
		cfg, err := awsprofiles.ParseINI(cfgPath)
		if err != nil {
			return err
		}
		section := awsConfigSection(in.Profile)
		if cfg[section] == nil {
			cfg[section] = awsprofiles.Section{}
		}
		if in.Region != "" {
			cfg[section]["region"] = in.Region
		}
		if in.Output != "" {
			cfg[section]["output"] = in.Output
		}
		if err := awsprofiles.WriteINI(cfgPath, cfg); err != nil {
			return err
		}
	}
	fmt.Fprintf(a.Stdout, "configured static profile %q\n", in.Profile)
	return nil
}

func (a *App) ConfigureSSO(in SSOProfileInput) error {
	if in.Profile == "" {
		return errors.New("--profile is required")
	}
	if in.StartURL == "" || in.SSORegion == "" || in.AccountID == "" || in.RoleName == "" {
		return errors.New("--sso-start-url, --sso-region, --sso-account-id, and --sso-role-name are required")
	}

	cfgPath := expandPath(a.Settings.ConfigFile)
	cfg, err := awsprofiles.ParseINI(cfgPath)
	if err != nil {
		return err
	}
	section := awsConfigSection(in.Profile)
	if cfg[section] == nil {
		cfg[section] = awsprofiles.Section{}
	}
	cfg[section]["sso_start_url"] = in.StartURL
	cfg[section]["sso_region"] = in.SSORegion
	cfg[section]["sso_account_id"] = in.AccountID
	cfg[section]["sso_role_name"] = in.RoleName
	if in.Region != "" {
		cfg[section]["region"] = in.Region
	}
	if in.Output != "" {
		cfg[section]["output"] = in.Output
	}
	if in.SessionName != "" {
		cfg[section]["sso_session"] = in.SessionName
	}
	if err := awsprofiles.WriteINI(cfgPath, cfg); err != nil {
		return err
	}

	fmt.Fprintf(a.Stdout, "configured sso profile %q\n", in.Profile)
	return nil
}

func (a *App) lookupProfile(profile string) (awsprofiles.Profile, error) {
	profiles, err := awsprofiles.Discover(a.Settings.CredentialsFile, a.Settings.ConfigFile)
	if err != nil {
		return awsprofiles.Profile{}, err
	}
	for _, p := range profiles {
		if p.Name == profile {
			return p, nil
		}
	}
	return awsprofiles.Profile{}, fmt.Errorf("unknown profile %q", profile)
}

func (a *App) maybeLoginSSO(p awsprofiles.Profile, shouldLogin bool) error {
	if !shouldLogin || !p.IsSSO {
		return nil
	}
	if a.hasValidSession(p.Name) {
		return nil
	}
	return a.LoginRunner(p.Name)
}

func (a *App) runSSOLogin(profile string) error {
	cmd := exec.Command("aws", "sso", "login", "--profile", profile)
	cmd.Stdin = a.Stdin
	cmd.Stdout = a.Stdout
	cmd.Stderr = a.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws sso login failed for %q: %w", profile, err)
	}
	return nil
}

func (a *App) hasValidSession(profile string) bool {
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--profile", profile)
	cmd.Stdin = nil
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

func PromptIfMissing(reader io.Reader, field, prompt, currentValue string) (string, error) {
	if currentValue != "" {
		return currentValue, nil
	}
	fmt.Fprintf(os.Stderr, "%s: ", prompt)
	s := bufio.NewReader(reader)
	v, err := s.ReadString('\n')
	if err != nil {
		if errors.Is(err, io.EOF) {
			v = strings.TrimSpace(v)
			if v == "" {
				return "", fmt.Errorf("%s is required", field)
			}
			return v, nil
		}
		return "", err
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	return v, nil
}

func awsConfigSection(profile string) string {
	if profile == "default" {
		return "default"
	}
	return "profile " + profile
}

func expandPath(path string) string {
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	return path
}
