package main

import (
	"bytes"
	"os"
	"testing"
)

func TestParseArgsPositional(t *testing.T) {
	args, err := parseArgs([]string{"acct", "role"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Target != "acct" || args.Role != "role" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestParseArgsFlags(t *testing.T) {
	args, err := parseArgs([]string{
		"--account", "123",
		"--role", "Admin",
		"--alias", "dev",
		"--profile", "prof",
		"--sso-session", "session",
		"--region", "us-east-1",
		"--kube-context", "ctx",
		"--no-kube",
		"--non-interactive",
		"--print-env",
		"--version",
	})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Account != "123" || args.Role != "Admin" || args.Alias != "dev" || args.Profile != "prof" {
		t.Fatalf("unexpected args: %+v", args)
	}
	if !args.NoKube || !args.NonInteractive || !args.PrintEnv || !args.Version {
		t.Fatalf("expected flags to be true: %+v", args)
	}
}

func TestParseArgsShortVersion(t *testing.T) {
	args, err := parseArgs([]string{"-v"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if !args.Version {
		t.Fatalf("expected Version=true for -v")
	}
}

func TestPrintUsage(t *testing.T) {
	buf := &bytes.Buffer{}
	printUsage(buf)
	if got := buf.String(); got == "" || got[0] != 'U' {
		t.Fatalf("expected usage output, got %q", got)
	}
}

func TestLogWriter(t *testing.T) {
	if logWriter(true) != os.Stderr {
		t.Fatalf("expected stderr writer")
	}
	if logWriter(false) != os.Stdout {
		t.Fatalf("expected stdout writer")
	}
}
