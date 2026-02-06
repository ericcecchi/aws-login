package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/ini.v1"
)

func TestLoadUserConfigMissing(t *testing.T) {
	home := setTempHome(t)
	path := filepath.Join(home, "missing.toml")
	cfg, err := loadUserConfig(path)
	if err != nil {
		t.Fatalf("loadUserConfig error: %v", err)
	}
	if cfg.Aliases == nil {
		t.Fatalf("expected aliases map to be initialized")
	}
}

func TestLoadAWSConfigCreatesFile(t *testing.T) {
	setTempHome(t)
	cfg, err := loadAWSConfig()
	if err != nil {
		t.Fatalf("loadAWSConfig error: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected config to be returned")
	}
	if _, err := os.Stat(expandPath(awsConfigPath)); err != nil {
		t.Fatalf("expected aws config file to exist: %v", err)
	}
}

func TestLoadAWSCredentialsCreatesFile(t *testing.T) {
	setTempHome(t)
	cfg, path, err := loadAWSCredentials()
	if err != nil {
		t.Fatalf("loadAWSCredentials error: %v", err)
	}
	if cfg == nil {
		t.Fatalf("expected credentials config to be returned")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected credentials file to exist: %v", err)
	}
}

func TestLoadUserConfigFile(t *testing.T) {
	home := setTempHome(t)
	path := filepath.Join(home, "config.toml")
	if err := os.WriteFile(path, []byte("[defaults]\nsso_session=\"my\"\n\n[aliases.dev]\naccount_id=\"123\"\ndefault_role=\"admin\"\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := loadUserConfig(path)
	if err != nil {
		t.Fatalf("loadUserConfig error: %v", err)
	}
	if cfg.Defaults.SSOSession != "my" {
		t.Fatalf("expected defaults sso_session, got %q", cfg.Defaults.SSOSession)
	}
	if _, ok := cfg.Aliases["dev"]; !ok {
		t.Fatalf("expected dev alias")
	}
}

func TestListSSOSessions(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[sso-session prod]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1

[profile default]
region = us-east-1
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	sessions := listSSOSessions(cfg)
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions["prod"].Region != "us-east-1" {
		t.Fatalf("unexpected session: %+v", sessions["prod"])
	}
}

func TestListProfiles(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[profile zeta]
region = us-east-1
[profile alpha]
region = us-east-1
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	profiles := listProfiles(cfg)
	if len(profiles) != 2 || profiles[0] != "alpha" || profiles[1] != "zeta" {
		t.Fatalf("unexpected profiles: %v", profiles)
	}
}

func TestGetProfileInfo(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[profile dev]
sso_session = prod
region = us-west-2
sso_account_id = 123
sso_role_name = admin

[default]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	info, err := getProfileInfo(cfg, "dev")
	if err != nil {
		t.Fatalf("getProfileInfo error: %v", err)
	}
	if info.SSOSession != "prod" || info.Region != "us-west-2" {
		t.Fatalf("unexpected profile info: %+v", info)
	}

	defaultInfo, err := getProfileInfo(cfg, "default")
	if err != nil {
		t.Fatalf("getProfileInfo default error: %v", err)
	}
	if defaultInfo.SSOStart == "" || defaultInfo.SSORegion == "" {
		t.Fatalf("unexpected default profile info: %+v", defaultInfo)
	}
}

func TestResolveAlias(t *testing.T) {
	alias := AliasConfig{
		AccountID:   "123",
		DefaultRole: "admin",
		Roles:       []string{"admin", "read"},
		Region:      "us-east-1",
		ProfileByRole: map[string]string{
			"admin": "profile-admin",
		},
	}
	acct, role, profile, kube, region, err := resolveAlias(alias, "")
	if err != nil {
		t.Fatalf("resolveAlias error: %v", err)
	}
	if acct != "123" || role != "admin" || profile != "profile-admin" || region != "us-east-1" || kube != "" {
		t.Fatalf("unexpected alias resolution: %q %q %q %q %q", acct, role, profile, kube, region)
	}

	_, _, _, _, _, err = resolveAlias(AliasConfig{}, "")
	if err == nil {
		t.Fatalf("expected error for missing account_id")
	}

	_, _, _, _, _, err = resolveAlias(AliasConfig{AccountID: "123"}, "")
	if err == nil {
		t.Fatalf("expected error for missing role")
	}

	_, _, _, _, _, err = resolveAlias(alias, "not-allowed")
	if err == nil {
		t.Fatalf("expected error for disallowed role")
	}
}

func TestMaybeBootstrapConfig(t *testing.T) {
	home := setTempHome(t)
	configPath := filepath.Join(home, "config", "aws-login.toml")
	awsCfg, err := ini.Load([]byte(`
[sso-session prod]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	buf := &bytes.Buffer{}
	maybeBootstrapConfig(configPath, awsCfg, buf)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected config to be created: %v", err)
	}
	if !bytes.Contains(data, []byte("sso_session = \"prod\"")) {
		t.Fatalf("expected default session in config, got:\n%s", string(data))
	}
}
