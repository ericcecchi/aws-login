package awslogin

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestStripNonDigits(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{{
		input: "123-45-6789",
		want:  "123456789",
	}, {
		input: "abc",
		want:  "",
	}, {
		input: "12 34",
		want:  "1234",
	}, {
		input: "",
		want:  "",
	}}

	for _, tt := range tests {
		if got := stripNonDigits(tt.input); got != tt.want {
			t.Fatalf("stripNonDigits(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{{
		input: "",
		want:  "''",
	}, {
		input: "simple",
		want:  "simple",
	}, {
		input: "has space",
		want:  "'has space'",
	}, {
		input: "it' s",
		want:  "'it'\\'' s'",
	}, {
		input: "dollar$sign",
		want:  "'dollar$sign'",
	}}

	for _, tt := range tests {
		if got := shellQuote(tt.input); got != tt.want {
			t.Fatalf("shellQuote(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatExports(t *testing.T) {
	exp := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	creds := RoleCredentials{
		AccessKeyID:     "AKIA123",
		SecretAccessKey: "secret",
		SessionToken:    "token",
		Expiration:      &exp,
	}
	got := formatExports(creds, "us-east-1", "profile with space")
	lines := strings.Split(got, "\n")
	if len(lines) < 6 {
		t.Fatalf("expected at least 6 export lines, got %d", len(lines))
	}
	if !strings.Contains(got, "AWS_PROFILE") {
		t.Fatalf("expected AWS_PROFILE export")
	}
	if !strings.Contains(got, "AWS_SESSION_EXPIRATION") {
		t.Fatalf("expected expiration export")
	}
}

func TestExpandPath(t *testing.T) {
	home := setTempHome(t)
	input := "~/.aws/config"
	got := expandPath(input)
	if !strings.HasPrefix(got, home) {
		t.Fatalf("expected expanded path to start with %q, got %q", home, got)
	}
	if got == input {
		t.Fatalf("expected path expansion for %q", input)
	}
}

func TestRunIdentityCheck(t *testing.T) {
	writeStubScripts(t, map[string]string{"aws": awsStubScript})
	t.Setenv("AWS_LOGIN_TEST_IDENTITY_JSON", `{"UserId":"abc","Account":"123","Arn":"arn:aws:sts::123:assumed-role"}`)
	buf := &bytes.Buffer{}
	runIdentityCheck("test-profile", "us-east-1", buf)
	output := buf.String()
	if !strings.Contains(output, "Current AWS identity") {
		t.Fatalf("expected identity output, got %q", output)
	}
	if !strings.Contains(output, "UserId: abc") {
		t.Fatalf("expected UserId in output, got %q", output)
	}
}

func TestShellInitScriptDefault(t *testing.T) {
	script := shellInitScript("bash")
	if !strings.Contains(script, "aws-login()") {
		t.Fatalf("expected bash shell function")
	}
	if !strings.Contains(script, "command aws-login --set-profile") {
		t.Fatalf("expected set-profile invocation")
	}
	if !strings.Contains(script, "doctor|--doctor") {
		t.Fatalf("expected doctor bypass in shell wrapper")
	}
}

func TestShellInitScriptFish(t *testing.T) {
	script := shellInitScript("fish")
	if !strings.Contains(script, "function aws-login") {
		t.Fatalf("expected fish function")
	}
	if !strings.Contains(script, "set -gx") {
		t.Fatalf("expected fish env export")
	}
	if !strings.Contains(script, "case doctor --doctor") {
		t.Fatalf("expected doctor bypass in fish wrapper")
	}
}
