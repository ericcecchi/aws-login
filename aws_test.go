package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/ini.v1"
)

func TestParseExpiry(t *testing.T) {
	t1 := parseExpiry("2024-01-02T03:04:05Z")
	if t1.IsZero() {
		t.Fatalf("expected parsed time")
	}
	t2 := parseExpiry("2024-01-02T03:04:05UTC")
	if t2.IsZero() {
		t.Fatalf("expected parsed time with UTC suffix")
	}
	if !parseExpiry("").IsZero() {
		t.Fatalf("expected zero time for empty input")
	}
}

func TestFindCachedToken(t *testing.T) {
	home := setTempHome(t)
	cacheDir := filepath.Join(home, ".aws", "sso", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	future := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	future2 := time.Now().UTC().Add(4 * time.Hour).Format(time.RFC3339)
	if err := os.WriteFile(filepath.Join(cacheDir, "a.json"), []byte(`{"startUrl":"https://start","accessToken":"token1","expiresAt":"`+future+`"}`), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "b.json"), []byte(`{"startUrl":"https://start","accessToken":"token2","expiresAt":"`+future2+`"}`), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	if token := findCachedToken("https://start"); token != "token2" {
		t.Fatalf("expected latest token, got %q", token)
	}
}

func TestResolveSessionWithExplicitSSO(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[sso-session prod]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	session, err := resolveSession(cfg, Config{}, &bytes.Buffer{}, "prod", "", true)
	if err != nil {
		t.Fatalf("resolveSession error: %v", err)
	}
	if session.Region != "us-east-1" || len(session.LoginArgs) == 0 {
		t.Fatalf("unexpected session: %+v", session)
	}
}

func TestResolveSessionWithProfileSession(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[sso-session prod]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1

[profile dev]
sso_session = prod
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	session, err := resolveSession(cfg, Config{}, &bytes.Buffer{}, "", "dev", true)
	if err != nil {
		t.Fatalf("resolveSession error: %v", err)
	}
	if session.StartURL == "" || session.Region != "us-east-1" {
		t.Fatalf("unexpected session: %+v", session)
	}
}

func TestResolveSessionWithProfileStartURL(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[sso-session prod]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1

[profile dev]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	session, err := resolveSession(cfg, Config{}, &bytes.Buffer{}, "", "dev", true)
	if err != nil {
		t.Fatalf("resolveSession error: %v", err)
	}
	if session.StartURL == "" || session.Region != "us-east-1" {
		t.Fatalf("unexpected session: %+v", session)
	}
	if len(session.LoginArgs) == 0 || session.LoginArgs[0] != "--profile" {
		t.Fatalf("expected --profile login args, got: %v", session.LoginArgs)
	}
}

func TestResolveSessionMultipleNonInteractive(t *testing.T) {
	cfg, err := ini.Load([]byte(`
[sso-session prod]
sso_start_url = https://example.awsapps.com/start
sso_region = us-east-1

[sso-session stage]
sso_start_url = https://example.awsapps.com/start2
sso_region = us-west-2
`))
	if err != nil {
		t.Fatalf("ini load: %v", err)
	}
	_, err = resolveSession(cfg, Config{}, &bytes.Buffer{}, "", "", true)
	if err == nil {
		t.Fatalf("expected error for multiple sessions in non-interactive mode")
	}
}

func TestAWSCLIJSONAndListFunctions(t *testing.T) {
	writeStubScripts(t, map[string]string{"aws": awsStubScript})
	t.Setenv("AWS_LOGIN_TEST_ACCOUNTS_JSON", `{"accountList":[{"accountId":"123","accountName":"Prod"}]}`)
	t.Setenv("AWS_LOGIN_TEST_ROLES_JSON", `{"roleList":[{"roleName":"Admin"}]}`)
	t.Setenv("AWS_LOGIN_TEST_CREDS_JSON", `{"roleCredentials":{"accessKeyId":"AKIA","secretAccessKey":"secret","sessionToken":"token","expiration":1700000000000}}`)
	accounts, err := listAccounts("token", "us-east-1")
	if err != nil {
		t.Fatalf("listAccounts error: %v", err)
	}
	if len(accounts) != 1 || accounts[0].AccountID != "123" {
		t.Fatalf("unexpected accounts: %+v", accounts)
	}
	roles, err := listRoles("token", "us-east-1", "123")
	if err != nil {
		t.Fatalf("listRoles error: %v", err)
	}
	if len(roles) != 1 || roles[0].RoleName != "Admin" {
		t.Fatalf("unexpected roles: %+v", roles)
	}
	creds, err := getRoleCredentials("token", "us-east-1", "123", "Admin")
	if err != nil {
		t.Fatalf("getRoleCredentials error: %v", err)
	}
	if creds.AccessKeyID != "AKIA" {
		t.Fatalf("unexpected creds: %+v", creds)
	}
}

func TestAWSCLIJSONError(t *testing.T) {
	script := "#!/usr/bin/env bash\n" +
		"echo 'failure' >&2\n" +
		"exit 1\n"
	writeStubScripts(t, map[string]string{"aws": script})
	_, err := awsCLIJSON([]string{"sso", "list-accounts"})
	if err == nil {
		t.Fatalf("expected error from awsCLIJSON")
	}
}

func TestEnsureLoggedInWithLoginStub(t *testing.T) {
	writeStubScripts(t, map[string]string{"aws": awsStubScript})
	home := setTempHome(t)
	cacheDir := filepath.Join(home, ".aws", "sso", "cache")
	future := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	t.Setenv("AWS_LOGIN_TEST_SSO_CACHE_DIR", cacheDir)
	t.Setenv("AWS_LOGIN_TEST_SSO_START_URL", "https://start")
	t.Setenv("AWS_LOGIN_TEST_SSO_TOKEN", "token-login")
	t.Setenv("AWS_LOGIN_TEST_SSO_EXPIRY", future)
	buf := &bytes.Buffer{}
	token, err := ensureLoggedIn(SessionInfo{StartURL: "https://start", Region: "us-east-1", LoginArgs: []string{"--sso-session", "prod"}}, buf)
	if err != nil {
		t.Fatalf("ensureLoggedIn error: %v", err)
	}
	if token != "token-login" {
		t.Fatalf("unexpected token: %q", token)
	}
}

func TestEnsureLoggedInUsesCachedToken(t *testing.T) {
	home := setTempHome(t)
	cacheDir := filepath.Join(home, ".aws", "sso", "cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatalf("mkdir cache: %v", err)
	}
	future := time.Now().UTC().Add(2 * time.Hour).Format(time.RFC3339)
	if err := os.WriteFile(filepath.Join(cacheDir, "token.json"), []byte(`{"startUrl":"https://start","accessToken":"cached","expiresAt":"`+future+`"}`), 0o644); err != nil {
		t.Fatalf("write cache: %v", err)
	}
	script := "#!/usr/bin/env bash\n" +
		"echo 'should not be called' >&2\n" +
		"exit 1\n"
	writeStubScripts(t, map[string]string{"aws": script})

	token, err := ensureLoggedIn(SessionInfo{StartURL: "https://start", Region: "us-east-1"}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("ensureLoggedIn error: %v", err)
	}
	if token != "cached" {
		t.Fatalf("unexpected token: %q", token)
	}
}
