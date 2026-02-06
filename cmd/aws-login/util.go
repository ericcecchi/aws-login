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

func mergeEnv(overrides map[string]string) []string {
	envMap := map[string]string{}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	for key, value := range overrides {
		envMap[key] = value
	}
	merged := make([]string, 0, len(envMap))
	for key, value := range envMap {
		merged = append(merged, fmt.Sprintf("%s=%s", key, value))
	}
	return merged
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

func runIdentityCheck(envVars map[string]string, w io.Writer) {
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--output", "json")
	cmd.Env = mergeEnv(envVars)
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
