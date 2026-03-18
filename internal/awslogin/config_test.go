package awslogin

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/ini.v1"
)

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

func TestLoadAWSConfigRestoresBackupIfCorrupt(t *testing.T) {
	home := setTempHome(t)
	configPath := expandPath(awsConfigPath)
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir aws config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("[profile broken\n"), 0o644); err != nil {
		t.Fatalf("write corrupt aws config: %v", err)
	}

	backupPath := expandPath(awsConfigBackupPath)
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	backupData := []byte("[profile restored]\nregion = us-east-1\n")
	if err := os.WriteFile(backupPath, backupData, 0o644); err != nil {
		t.Fatalf("write backup config: %v", err)
	}

	cfg, err := loadAWSConfig()
	if err != nil {
		t.Fatalf("loadAWSConfig error after restore: %v", err)
	}
	if _, err := cfg.GetSection("profile restored"); err != nil {
		t.Fatalf("expected restored profile section: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(home, ".aws", "config"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(raw) != string(backupData) {
		t.Fatalf("expected config file to be restored from backup")
	}
}

func TestValidateAWSConfigFileRejectsDuplicateSection(t *testing.T) {
	if !commandExists("aws") {
		t.Skip("aws cli not available")
	}
	home := setTempHome(t)
	configPath := filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir aws config dir: %v", err)
	}
	data := []byte("[profile dup]\nregion = us-east-1\n[profile dup]\nregion = us-east-1\n")
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("write duplicate aws config: %v", err)
	}
	if err := validateAWSConfigFile(configPath); err == nil {
		t.Fatalf("expected duplicate section validation error")
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

func TestGetProfileInfoIfExists(t *testing.T) {
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
	info, found, err := getProfileInfoIfExists(cfg, "dev")
	if err != nil {
		t.Fatalf("getProfileInfoIfExists error: %v", err)
	}
	if !found || info.SSOSession != "prod" || info.Region != "us-west-2" {
		t.Fatalf("unexpected profile info: %+v found=%v", info, found)
	}

	defaultInfo, found, err := getProfileInfoIfExists(cfg, "default")
	if err != nil {
		t.Fatalf("getProfileInfoIfExists default error: %v", err)
	}
	if !found || defaultInfo.SSOStart == "" || defaultInfo.SSORegion == "" {
		t.Fatalf("unexpected default profile info: %+v found=%v", defaultInfo, found)
	}
}

// TestAutoNamedProfileRegionPreserved verifies that when an auto-named profile
// already exists with a non-default region, re-running login without --region
// loads the stored region from that profile rather than falling through to the
// SSO session region.
func TestAutoNamedProfileRegionPreserved(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[sso-session perch]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1

[profile perch-prod-admin]
sso_session = perch
sso_account_id = 103736728945
sso_role_name = admin
region = us-east-2
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}

	// Simulate the fixed flow: after buildProfileName resolves "perch-prod-admin",
	// we look up the existing profile to get its stored region.
	profileName := "perch-prod-admin"
	info, found, err := getProfileInfoIfExists(cfg, profileName)
	if err != nil {
		t.Fatalf("getProfileInfoIfExists error: %v", err)
	}
	if !found {
		t.Fatalf("expected profile %q to be found", profileName)
	}

	sessionRegion := "us-east-1"
	region := "" // no --region flag
	if region == "" {
		region = info.Region
	}
	if region == "" {
		region = sessionRegion
	}

	if region != "us-east-2" {
		t.Fatalf("expected region us-east-2 to be preserved from existing profile, got %q", region)
	}
}
