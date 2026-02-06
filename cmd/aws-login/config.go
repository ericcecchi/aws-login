package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/ini.v1"
)

const (
	defaultConfigPath = "~/.aws-login.toml"
	awsConfigPath     = "~/.aws/config"
)

func loadAWSConfig() (*ini.File, error) {
	path := expandPath(awsConfigPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("unable to create %s", path)
		}
		if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
			return nil, fmt.Errorf("unable to create %s", path)
		}
	}
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read %s", path)
	}
	return cfg, nil
}

func loadAWSCredentials() (*ini.File, string, error) {
	path := expandPath(awsCredentialsPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, "", fmt.Errorf("unable to create %s", path)
		}
		if err := os.WriteFile(path, []byte{}, 0o600); err != nil {
			return nil, "", fmt.Errorf("unable to create %s", path)
		}
	}
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, "", fmt.Errorf("unable to read %s", path)
	}
	return cfg, path, nil
}

func loadUserConfig(path string) (Config, error) {
	cfg := Config{Aliases: map[string]AliasConfig{}}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to read %s: %w", path, err)
	}
	if cfg.Aliases == nil {
		cfg.Aliases = map[string]AliasConfig{}
	}
	return cfg, nil
}

func maybeBootstrapConfig(path string, awsCfg *ini.File, w io.Writer) {
	if _, err := os.Stat(path); err == nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		logLine(w, fmt.Sprintf("Warning: failed to create config directory: %v", err))
		return
	}

	sessions := listSSOSessions(awsCfg)
	defaultSession := ""
	if len(sessions) == 1 {
		for name := range sessions {
			defaultSession = name
		}
	}

	lines := []string{
		"# aws-login configuration",
		"",
		"[defaults]",
	}
	if defaultSession != "" {
		lines = append(lines, fmt.Sprintf("sso_session = \"%s\"", defaultSession))
	} else {
		lines = append(lines, "# sso_session = \"my-session\"")
	}
	lines = append(lines,
		"",
		"# Define aliases to map friendly names to account IDs and roles.",
		"# [aliases.dev]",
		"# account_id = \"123456789012\"",
		"# default_role = \"admin\"",
		"# roles = [\"admin\", \"read\"]",
	)

	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644); err != nil {
		logLine(w, fmt.Sprintf("Warning: failed to create config at %s: %v", path, err))
		return
	}
	logLine(w, fmt.Sprintf("Created default config at %s", path))
}

func listSSOSessions(cfg *ini.File) map[string]SessionInfo {
	out := map[string]SessionInfo{}
	for _, section := range cfg.Sections() {
		name := section.Name()
		if !strings.HasPrefix(name, "sso-session ") {
			continue
		}
		sessionName := strings.TrimSpace(strings.TrimPrefix(name, "sso-session "))
		startURL := section.Key("sso_start_url").String()
		region := section.Key("sso_region").String()
		if sessionName != "" && startURL != "" && region != "" {
			out[sessionName] = SessionInfo{Name: sessionName, StartURL: startURL, Region: region}
		}
	}
	return out
}

func listProfiles(cfg *ini.File) []string {
	profiles := []string{}
	for _, section := range cfg.Sections() {
		name := section.Name()
		if strings.HasPrefix(name, "profile ") {
			profiles = append(profiles, strings.TrimSpace(strings.TrimPrefix(name, "profile ")))
		}
	}
	sort.Strings(profiles)
	return profiles
}

func getProfileInfo(cfg *ini.File, profile string) (ProfileInfo, error) {
	sectionName := "profile " + profile
	section, err := cfg.GetSection(sectionName)
	if err != nil {
		if profile == "default" {
			section, err = cfg.GetSection("default")
		}
	}
	if err != nil {
		return ProfileInfo{}, fmt.Errorf("profile '%s' not found", profile)
	}

	return ProfileInfo{
		SSOSession: section.Key("sso_session").String(),
		SSOStart:   section.Key("sso_start_url").String(),
		SSORegion:  section.Key("sso_region").String(),
		Region:     section.Key("region").String(),
		AccountID:  section.Key("sso_account_id").String(),
		RoleName:   section.Key("sso_role_name").String(),
	}, nil
}

func resolveAlias(alias AliasConfig, roleOverride string) (string, string, string, string, string, error) {
	if alias.AccountID == "" {
		return "", "", "", "", "", fmt.Errorf("alias is missing account_id")
	}
	role := roleOverride
	if role == "" {
		role = alias.DefaultRole
	}
	if role == "" {
		return "", "", "", "", "", fmt.Errorf("alias does not define a role")
	}
	if len(alias.Roles) > 0 {
		allowed := false
		for _, r := range alias.Roles {
			if r == role {
				allowed = true
				break
			}
		}
		if !allowed {
			return "", "", "", "", "", fmt.Errorf("role '%s' is not allowed for alias", role)
		}
	}
	profile := ""
	if alias.ProfileByRole != nil {
		profile = alias.ProfileByRole[role]
	}
	return alias.AccountID, role, profile, alias.KubeContext, alias.Region, nil
}
