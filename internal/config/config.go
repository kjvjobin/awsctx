package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const dirPerm = 0o700
const filePerm = 0o600

type Settings struct {
	CredentialsFile string `json:"credentials_file"`
	ConfigFile      string `json:"config_file"`
	StateFile       string `json:"state_file"`
	FZFCommand      string `json:"fzf_command"`
	AutoSSOLogin    bool   `json:"auto_sso_login"`
}

func defaultConfigPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "awsctx", "config.json"), nil
}

func defaultStatePath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "awsctx", "state.json"), nil
}

func defaultCredentialsPath() string {
	if p := os.Getenv("AWS_SHARED_CREDENTIALS_FILE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.aws/credentials"
	}
	return filepath.Join(home, ".aws", "credentials")
}

func defaultAWSConfigPath() string {
	if p := os.Getenv("AWS_CONFIG_FILE"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "~/.aws/config"
	}
	return filepath.Join(home, ".aws", "config")
}

func defaults() (Settings, error) {
	statePath, err := defaultStatePath()
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		CredentialsFile: defaultCredentialsPath(),
		ConfigFile:      defaultAWSConfigPath(),
		StateFile:       statePath,
		FZFCommand:      "fzf",
		AutoSSOLogin:    false,
	}, nil
}

func Load(configPath string) (Settings, string, error) {
	if configPath == "" {
		var err error
		configPath, err = defaultConfigPath()
		if err != nil {
			return Settings{}, "", err
		}
	}

	def, err := defaults()
	if err != nil {
		return Settings{}, "", err
	}

	b, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		return def, configPath, nil
	}
	if err != nil {
		return Settings{}, "", fmt.Errorf("read config: %w", err)
	}

	cfg := def
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Settings{}, "", fmt.Errorf("parse config: %w", err)
	}

	if cfg.CredentialsFile == "" {
		cfg.CredentialsFile = def.CredentialsFile
	}
	if cfg.ConfigFile == "" {
		cfg.ConfigFile = def.ConfigFile
	}
	if cfg.StateFile == "" {
		cfg.StateFile = def.StateFile
	}
	if cfg.FZFCommand == "" {
		cfg.FZFCommand = def.FZFCommand
	}

	return cfg, configPath, nil
}

func Save(configPath string, cfg Settings) error {
	if configPath == "" {
		return errors.New("empty config path")
	}
	if err := os.MkdirAll(filepath.Dir(configPath), dirPerm); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	body, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	body = append(body, '\n')

	return writeAtomic(configPath, body, filePerm)
}

func writeAtomic(path string, content []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-awsctx-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("chmod target file: %w", err)
	}
	return nil
}
