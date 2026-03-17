package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func detectShell() string {
	if override := strings.TrimSpace(os.Getenv("AWS_LOGIN_SHELL")); override != "" {
		return strings.ToLower(override)
	}
	shell := strings.ToLower(filepath.Base(os.Getenv("SHELL")))
	if shell == "" {
		return "bash"
	}
	return shell
}

func shellInitScript(shell string) string {
	switch shell {
	case "fish":
		return `function aws-login
  if test (count $argv) -gt 0
    switch $argv[1]
      case doctor --doctor --version -v --shell-init -h --help
        command aws-login $argv
        return $status
    end
  end
  set -l out (command aws-login --set-profile $argv); or return $status
  for line in $out
    if test (string sub -l 7 $line) = "export "
      set -l kv (string sub -s 8 $line)
      set -l parts (string split -m 1 "=" $kv)
      if test (count $parts) -ge 2
        set -gx $parts[1] $parts[2]
      end
    end
  end
end
`
	default:
		return `aws-login() {
  if [ "$#" -gt 0 ]; then
    case "$1" in
      doctor|--doctor|--version|-v|--shell-init|-h|--help)
        command aws-login "$@"
        return
        ;;
    esac
  fi
  local out
  out="$(command aws-login --set-profile "$@")" || return
  eval "$out"
}
`
	}
}

func logLine(w io.Writer, message string) {
	fmt.Fprintln(w, message)
}

func commandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, strings.TrimPrefix(path, "~"))
		}
	}
	return path
}

func stripNonDigits(input string) string {
	var b strings.Builder
	for _, r := range input {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatExports(creds RoleCredentials, region, profile string) string {
	lines := []string{}
	if profile != "" {
		lines = append(lines, fmt.Sprintf("export AWS_PROFILE=%s", shellQuote(profile)))
	}
	lines = append(lines, fmt.Sprintf("export AWS_REGION=%s", shellQuote(region)))
	lines = append(lines, fmt.Sprintf("export AWS_DEFAULT_REGION=%s", shellQuote(region)))
	lines = append(lines, fmt.Sprintf("export AWS_ACCESS_KEY_ID=%s", shellQuote(creds.AccessKeyID)))
	lines = append(lines, fmt.Sprintf("export AWS_SECRET_ACCESS_KEY=%s", shellQuote(creds.SecretAccessKey)))
	lines = append(lines, fmt.Sprintf("export AWS_SESSION_TOKEN=%s", shellQuote(creds.SessionToken)))
	if creds.Expiration != nil {
		lines = append(lines, fmt.Sprintf("export AWS_SESSION_EXPIRATION=%s", shellQuote(creds.Expiration.Format(time.RFC3339))))
	}
	return strings.Join(lines, "\n")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	if !strings.ContainsAny(value, " \t\n'\"\\$`!") {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func runIdentityCheck(profileName, region string, w io.Writer) {
	args := []string{"sts", "get-caller-identity", "--output", "json"}
	if region != "" {
		args = append(args, "--region", region)
	}
	if profileName != "" {
		args = append(args, "--profile", profileName)
	}
	cmd := exec.Command("aws", args...)
	output, err := cmd.Output()
	if err != nil {
		logLine(w, "Could not retrieve identity")
		return
	}
	var data struct {
		UserID  string `json:"UserId"`
		Account string `json:"Account"`
		Arn     string `json:"Arn"`
	}
	if err := json.Unmarshal(output, &data); err != nil {
		return
	}
	logLine(w, "📋 Current AWS identity:")
	if data.UserID != "" {
		logLine(w, fmt.Sprintf("UserId: %s", data.UserID))
	}
	if data.Account != "" {
		logLine(w, fmt.Sprintf("Account: %s", data.Account))
	}
	if data.Arn != "" {
		logLine(w, fmt.Sprintf("Arn: %s", data.Arn))
	}
}
