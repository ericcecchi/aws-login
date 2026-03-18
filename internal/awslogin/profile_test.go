package awslogin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/ini.v1"
)

func TestSanitizeProfilePart(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{{
		input: "Admin Role",
		want:  "admin-role",
	}, {
		input: "  ",
		want:  "",
	}, {
		input: "Prod--Role",
		want:  "prod-role",
	}, {
		input: "name_with_underscore",
		want:  "name-with-underscore",
	}}

	for _, tt := range tests {
		if got := sanitizeProfilePart(tt.input); got != tt.want {
			t.Fatalf("sanitizeProfilePart(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestBuildProfileName(t *testing.T) {
	acct := AccountInfo{AccountName: "Prod Account", AccountID: "123"}
	role := RoleInfo{RoleName: "Admin"}
	got := buildProfileName(acct, role)
	if got != "aws-login-123-admin" {
		t.Fatalf("unexpected profile name: %q", got)
	}
}

func TestConfigureProfileUsesAWSCLI(t *testing.T) {
	writeStubScripts(t, map[string]string{"aws": awsStubScript})
	setCallsFile := filepath.Join(t.TempDir(), "configure_set_calls.txt")
	t.Setenv("AWS_LOGIN_TEST_CONFIGURE_SET_FILE", setCallsFile)

	session := SessionInfo{Name: "prod", Region: "us-east-1"}
	if err := configureProfile("test-profile", "us-east-1", session, "123", "admin"); err != nil {
		t.Fatalf("configureProfile error: %v", err)
	}

	data, err := os.ReadFile(setCallsFile)
	if err != nil {
		t.Fatalf("read configure set calls: %v", err)
	}
	got := string(data)
	checks := []string{
		"profile.test-profile.sso_session=prod",
		"profile.test-profile.sso_account_id=123",
		"profile.test-profile.sso_role_name=admin",
		"profile.test-profile.region=us-east-1",
		"profile.test-profile.output=json",
	}
	for _, check := range checks {
		if !strings.Contains(got, check) {
			t.Fatalf("expected %q in configure calls, got:\n%s", check, got)
		}
	}
}

func TestEnsureReusableSSOSessionUsesExisting(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[sso-session prod]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}

	session := SessionInfo{StartURL: "https://example.awsapps.com/start", Region: "us-east-1", LoginArgs: []string{"--profile", "dev"}}
	resolved, err := ensureReusableSSOSession(cfg, session)
	if err != nil {
		t.Fatalf("ensureReusableSSOSession error: %v", err)
	}
	if resolved.Name != "prod" {
		t.Fatalf("expected existing session name, got %q", resolved.Name)
	}
	if len(resolved.LoginArgs) < 2 || resolved.LoginArgs[0] != "--sso-session" || resolved.LoginArgs[1] != "prod" {
		t.Fatalf("expected --sso-session login args, got %v", resolved.LoginArgs)
	}
}

func TestEnsureReusableSSOSessionCreatesNew(t *testing.T) {
	writeStubScripts(t, map[string]string{"aws": awsStubScript})
	setCallsFile := filepath.Join(t.TempDir(), "configure_set_calls.txt")
	t.Setenv("AWS_LOGIN_TEST_CONFIGURE_SET_FILE", setCallsFile)

	cfg, err := ini.Load([]byte(""))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}

	session := SessionInfo{StartURL: "https://example.awsapps.com/start", Region: "us-east-1"}
	resolved, err := ensureReusableSSOSession(cfg, session)
	if err != nil {
		t.Fatalf("ensureReusableSSOSession error: %v", err)
	}
	if resolved.Name != "aws-login" {
		t.Fatalf("expected generated session name aws-login, got %q", resolved.Name)
	}
	if len(resolved.LoginArgs) < 2 || resolved.LoginArgs[0] != "--sso-session" || resolved.LoginArgs[1] != "aws-login" {
		t.Fatalf("expected generated --sso-session login args, got %v", resolved.LoginArgs)
	}

	data, err := os.ReadFile(setCallsFile)
	if err != nil {
		t.Fatalf("read configure set calls: %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "sso-session.aws-login.sso_start_url=https://example.awsapps.com/start") {
		t.Fatalf("missing sso_start_url configure call, got:\n%s", got)
	}
	if !strings.Contains(got, "sso-session.aws-login.sso_region=us-east-1") {
		t.Fatalf("missing sso_region configure call, got:\n%s", got)
	}
}
