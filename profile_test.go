package main

import (
	"path/filepath"
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
	if got != "aws-login-prod-account-admin" {
		t.Fatalf("unexpected profile name: %q", got)
	}
}

func TestWriteProfile(t *testing.T) {
	home := setTempHome(t)
	creds := RoleCredentials{AccessKeyID: "AKIA", SecretAccessKey: "secret", SessionToken: "token"}
	if err := writeProfile("test-profile", "us-east-1", creds); err != nil {
		t.Fatalf("writeProfile error: %v", err)
	}

	configPath := expandPath(awsConfigPath)
	credsPath := expandPath(awsCredentialsPath)
	if filepath.Dir(configPath) == "" || filepath.Dir(credsPath) == "" {
		t.Fatalf("expected valid paths under home %q", home)
	}

	cfg, err := ini.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	section, err := cfg.GetSection("profile test-profile")
	if err != nil {
		t.Fatalf("expected profile section: %v", err)
	}
	if section.Key("region").String() != "us-east-1" {
		t.Fatalf("expected region set")
	}

	credsCfg, err := ini.Load(credsPath)
	if err != nil {
		t.Fatalf("load credentials: %v", err)
	}
	credsSection, err := credsCfg.GetSection("test-profile")
	if err != nil {
		t.Fatalf("expected creds section: %v", err)
	}
	if credsSection.Key("aws_access_key_id").String() != "AKIA" {
		t.Fatalf("expected access key")
	}
}
