package awsprofiles

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Profile struct {
	Name   string
	Source string
	IsSSO  bool
}

type Section map[string]string

type FileData map[string]Section

func ParseINI(path string) (FileData, error) {
	out := FileData{}
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	cur := ""
	for s.Scan() {
		line := strings.TrimSpace(s.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			cur = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "["), "]"))
			if cur != "" {
				if _, ok := out[cur]; !ok {
					out[cur] = Section{}
				}
			}
			continue
		}
		if cur == "" {
			continue
		}

		k, v, ok := strings.Cut(line, "=")
		if !ok {
			k, v, ok = strings.Cut(line, ":")
			if !ok {
				continue
			}
		}
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" {
			continue
		}
		out[cur][k] = v
	}
	if err := s.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return out, nil
}

func WriteINI(path string, data FileData) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create dir for %s: %w", path, err)
	}

	sections := make([]string, 0, len(data))
	for k := range data {
		sections = append(sections, k)
	}
	sort.Strings(sections)

	var b strings.Builder
	for i, section := range sections {
		b.WriteString("[")
		b.WriteString(section)
		b.WriteString("]\n")

		keys := make([]string, 0, len(data[section]))
		for k := range data[section] {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			b.WriteString(k)
			b.WriteString(" = ")
			b.WriteString(data[section][k])
			b.WriteString("\n")
		}
		if i != len(sections)-1 {
			b.WriteString("\n")
		}
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-awsctx-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod temp file: %w", err)
	}
	if _, err := tmp.WriteString(b.String()); err != nil {
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
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("chmod target file: %w", err)
	}
	return nil
}

func NormalizeConfigProfile(section string) string {
	const prefix = "profile "
	if strings.HasPrefix(section, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(section, prefix))
	}
	return strings.TrimSpace(section)
}

func Discover(credentialsPath, configPath string) ([]Profile, error) {
	creds, err := ParseINI(credentialsPath)
	if err != nil {
		return nil, err
	}
	cfg, err := ParseINI(configPath)
	if err != nil {
		return nil, err
	}

	lookup := map[string]Profile{}

	for section := range creds {
		name := strings.TrimSpace(section)
		if name == "" {
			continue
		}
		lookup[name] = Profile{Name: name, Source: "credentials", IsSSO: false}
	}

	for section, values := range cfg {
		name := NormalizeConfigProfile(section)
		if name == "" {
			continue
		}
		_, hasStart := values["sso_start_url"]
		_, hasRegion := values["sso_region"]
		_, hasRole := values["sso_role_name"]
		isSSO := hasStart && hasRegion && hasRole
		if existing, ok := lookup[name]; ok {
			existing.Source = "credentials+config"
			existing.IsSSO = existing.IsSSO || isSSO
			lookup[name] = existing
		} else {
			lookup[name] = Profile{Name: name, Source: "config", IsSSO: isSSO}
		}
	}

	profiles := make([]Profile, 0, len(lookup))
	for _, p := range lookup {
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	return profiles, nil
}
