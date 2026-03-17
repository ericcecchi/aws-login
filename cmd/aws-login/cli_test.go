package main

import (
	"bytes"
	"os"
	"testing"
)

func TestParseArgsTwoPositionals(t *testing.T) {
	args, err := parseArgs([]string{"myaccount", "admin"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Account != "myaccount" || args.Role != "admin" {
		t.Fatalf("unexpected args: %+v", args)
	}
}

func TestParseArgsSinglePositionalAccount(t *testing.T) {
	args, err := parseArgs([]string{"myaccount"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Account != "myaccount" || args.Role != "" {
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
		"--set-profile",
		"--version",
	})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Account != "123" || args.Role != "Admin" || args.Profile != "prof" {
		t.Fatalf("unexpected args: %+v", args)
	}
	if !args.NoKube || !args.NonInteractive || !args.PrintEnv || !args.SetProfile || !args.Version {
		t.Fatalf("expected flags to be true: %+v", args)
	}
}

func TestParseArgsInterspersedFlags(t *testing.T) {
	args, err := parseArgs([]string{"myaccount", "--print-env", "admin"})
	if err != nil {
		t.Fatalf("parseArgs error: %v", err)
	}
	if args.Account != "myaccount" || args.Role != "admin" {
		t.Fatalf("unexpected positional args: %+v", args)
	}
	if !args.PrintEnv {
		t.Fatalf("expected PrintEnv=true")
	}
}

func TestParseArgsTooManyPositionals(t *testing.T) {
	_, err := parseArgs([]string{"acct", "role", "extra"})
	if err == nil {
		t.Fatalf("expected positional argument error")
	}
}

func TestParseArgsPositionalConflictsWithAccountFlag(t *testing.T) {
	_, err := parseArgs([]string{"myaccount", "--account", "otheraccount"})
	if err == nil {
		t.Fatalf("expected error when positional account conflicts with --account flag")
	}
}

func TestParseArgsPositionalConflictsWithRoleFlag(t *testing.T) {
	_, err := parseArgs([]string{"myaccount", "myrole", "--role", "otherrole"})
	if err == nil {
		t.Fatalf("expected error when positional role conflicts with --role flag")
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
	if args.Account != "" {
		t.Fatalf("expected empty account for doctor command, got %q", args.Account)
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
	if logWriter(Args{PrintEnv: true}) != os.Stderr {
		t.Fatalf("expected stderr writer for PrintEnv")
	}
	if logWriter(Args{SetProfile: true}) != os.Stderr {
		t.Fatalf("expected stderr writer for SetProfile")
	}
	if logWriter(Args{}) != os.Stdout {
		t.Fatalf("expected stdout writer")
	}
}
