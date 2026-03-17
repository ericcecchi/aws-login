package main

import (
	"bytes"
	"os"
	"testing"
)

func TestParseArgsPositional(t *testing.T) {
	args, err := parseArgs([]string{"dev"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Target != "dev" || args.Role != "" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestParseArgsFlags(t *testing.T) {
	args, err := parseArgs([]string{
		"--account", "123",
		"--role", "Admin",
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
	if args.Account != "123" || args.Role != "Admin" || args.Profile != "prof" {
		t.Fatalf("unexpected args: %+v", args)
	}
	if !args.NoKube || !args.NonInteractive || !args.PrintEnv || !args.Version {
		t.Fatalf("expected flags to be true: %+v", args)
	}
}

func TestParseArgsInterspersedFlags(t *testing.T) {
	args, err := parseArgs([]string{"dev", "--print-env"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Target != "dev" || args.Role != "" {
		t.Fatalf("unexpected positional args: %+v", args)
	}
	if !args.PrintEnv {
		t.Fatalf("expected PrintEnv=true")
	}
}

func TestParseArgsTooManyPositionals(t *testing.T) {
	_, err := parseArgs([]string{"dev", "admin"})
	if err == nil {
		t.Fatalf("expected positional argument error")
	}
}

func TestParseArgsDoctorPositional(t *testing.T) {
	args, err := parseArgs([]string{"doctor"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if !args.Doctor {
		t.Fatalf("expected Doctor=true")
	}
	if args.Target != "" {
		t.Fatalf("expected empty target for doctor command, got %q", args.Target)
	}
}

func TestParseArgsDoctorFlag(t *testing.T) {
	args, err := parseArgs([]string{"--doctor"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if !args.Doctor {
		t.Fatalf("expected Doctor=true")
	}
}

func TestParseArgsDoctorRejectsExtraPositionals(t *testing.T) {
	_, err := parseArgs([]string{"doctor", "extra"})
	if err == nil {
		t.Fatalf("expected doctor positional error")
	}
}

func TestParseArgsShellInit(t *testing.T) {
	args, err := parseArgs([]string{"--shell-init"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if !args.ShellInit {
		t.Fatalf("expected ShellInit=true")
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
