package awslogin

import (
	"bytes"
	"os"
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

func TestShellInitDir(t *testing.T) {
	setTempHome(t)
	dir, err := shellInitDir()
	if err != nil {
		t.Fatalf("shellInitDir failed: %v", err)
	}
	if !strings.Contains(dir, ".aws-login") {
		t.Fatalf("expected .aws-login in path, got %q", dir)
	}
}

func TestShellInitFile(t *testing.T) {
	setTempHome(t)
	tests := []struct {
		shell    string
		contains string
	}{
		{"bash", "init.sh"},
		{"zsh", "init.zsh"},
		{"fish", "init.fish"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			file, err := shellInitFile(tt.shell)
			if err != nil {
				t.Fatalf("shellInitFile failed: %v", err)
			}
			if !strings.Contains(file, tt.contains) {
				t.Fatalf("expected %s in path, got %q", tt.contains, file)
			}
		})
	}
}

func TestShellInitHookLine(t *testing.T) {
	setTempHome(t)
	tests := []struct {
		shell    string
		contains string
	}{
		{"bash", "source"},
		{"zsh", "source"},
		{"fish", "source"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			line, err := shellInitHookLine(tt.shell)
			if err != nil {
				t.Fatalf("shellInitHookLine failed: %v", err)
			}
			if !strings.Contains(line, tt.contains) {
				t.Fatalf("expected %q in hook line, got %q", tt.contains, line)
			}
			if !strings.Contains(line, ".aws-login") {
				t.Fatalf("expected .aws-login path in hook line, got %q", line)
			}
			// Hook line must use tilde path, not absolute.
			if !strings.HasPrefix(line, "source ~/") {
				t.Fatalf("expected tilde path in hook line, got %q", line)
			}
			// Tilde path should not be quoted (tilde expansion is not word-split).
			if strings.Contains(line, "'") {
				t.Fatalf("hook line should not use single quotes, got %q", line)
			}
		})
	}
}

func TestGenerateShellInitContent(t *testing.T) {
	tests := []struct {
		shell           string
		expectedContent string
	}{
		{"bash", "aws-login()"},
		{"zsh", "aws-login()"},
		{"fish", "function aws-login"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			content := generateShellInitContent(tt.shell)
			if !strings.Contains(content, tt.expectedContent) {
				t.Fatalf("expected %q in generated content", tt.expectedContent)
			}
			if !strings.Contains(content, "AWS Login shell initialization") {
				t.Fatalf("expected initialization comment in generated content")
			}
		})
	}
}

func TestShellRCFiles(t *testing.T) {
	setTempHome(t)
	tests := []struct {
		shell        string
		minExpected  int
		expectedName string
	}{
		{"bash", 1, ".bashrc"},
		{"zsh", 2, ".zshrc"},
		{"fish", 1, "config.fish"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			files, err := shellRCFiles(tt.shell)
			if err != nil {
				t.Fatalf("shellRCFiles failed: %v", err)
			}
			if len(files) < tt.minExpected {
				t.Fatalf("expected at least %d files, got %d", tt.minExpected, len(files))
			}
			found := false
			for _, f := range files {
				if strings.Contains(f, tt.expectedName) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected %q in rc files list, got %v", tt.expectedName, files)
			}
		})
	}
}

func TestEnsureShellInitFiles(t *testing.T) {
	setTempHome(t)
	err := ensureShellInitFiles()
	if err != nil {
		t.Fatalf("ensureShellInitFiles failed: %v", err)
	}

	shells := []string{"bash", "zsh", "fish"}
	for _, shell := range shells {
		file, _ := shellInitFile(shell)
		if _, err := fileStatOrDie(t, file); err != nil {
			t.Fatalf("expected file %s to exist, got: %v", file, err)
		}
		content := readFileOrDie(t, file)
		if !strings.Contains(content, "AWS Login shell initialization") {
			t.Fatalf("expected initialization comment in %s", file)
		}
	}
}

func TestInstallShellIntegration(t *testing.T) {
	home := setTempHome(t)

	// Create rc files
	zshrc := home + "/.zshrc"
	if err := createFileWithContent(zshrc, "# existing zshrc\n"); err != nil {
		t.Fatalf("failed to create test rc file: %v", err)
	}

	buf := &bytes.Buffer{}
	err := installShellIntegration("zsh", buf)
	if err != nil {
		t.Fatalf("installShellIntegration failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "✓ Updated") {
		t.Fatalf("expected success message, got: %q", output)
	}
	if !strings.Contains(output, "✅ Shell integration installed") {
		t.Fatalf("expected completion message")
	}
	if !strings.Contains(output, "To activate in your current session") {
		t.Fatalf("expected source-to-activate hint, got: %q", output)
	}

	// Verify hook was added with tilde path
	content := readFileOrDie(t, zshrc)
	if !strings.Contains(content, "source ~/") || !strings.Contains(content, ".aws-login") {
		t.Fatalf("expected tilde hook line in zshrc, got: %s", content)
	}
}

func TestInstallShellIntegrationCreatesMissingPrimaryRCFile(t *testing.T) {
	home := setTempHome(t)

	// Fish config dir does not exist — install should create it
	buf := &bytes.Buffer{}
	err := installShellIntegration("fish", buf)
	if err != nil {
		t.Fatalf("installShellIntegration failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "✓ Created") {
		t.Fatalf("expected 'Created' message for missing rc file, got: %q", output)
	}

	fishRC := home + "/.config/fish/config.fish"
	content := readFileOrDie(t, fishRC)
	if !strings.Contains(content, "source ~/") || !strings.Contains(content, ".aws-login") {
		t.Fatalf("expected tilde hook line in created config.fish, got: %s", content)
	}
}

func TestInstallShellIntegrationIdempotent(t *testing.T) {
	home := setTempHome(t)

	// Create rc file
	zshrc := home + "/.zshrc"
	if err := createFileWithContent(zshrc, "# existing zshrc\n"); err != nil {
		t.Fatalf("failed to create test rc file: %v", err)
	}

	buf1 := &bytes.Buffer{}
	err := installShellIntegration("zsh", buf1)
	if err != nil {
		t.Fatalf("first install failed: %v", err)
	}

	contentAfterFirst := readFileOrDie(t, zshrc)

	// Install again - should be idempotent
	buf2 := &bytes.Buffer{}
	err = installShellIntegration("zsh", buf2)
	if err != nil {
		t.Fatalf("second install failed: %v", err)
	}

	output2 := buf2.String()
	if !strings.Contains(output2, "already installed") {
		t.Fatalf("expected idempotency message, got: %q", output2)
	}

	contentAfterSecond := readFileOrDie(t, zshrc)
	if contentAfterFirst != contentAfterSecond {
		t.Fatalf("file changed after second install - not idempotent")
	}
}

func TestUninstallShellIntegration(t *testing.T) {
	home := setTempHome(t)

	// Create and install first
	zshrc := home + "/.zshrc"
	if err := createFileWithContent(zshrc, "# existing zshrc\n"); err != nil {
		t.Fatalf("failed to create test rc file: %v", err)
	}

	buf1 := &bytes.Buffer{}
	err := installShellIntegration("zsh", buf1)
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}

	contentAfterInstall := readFileOrDie(t, zshrc)
	if !strings.Contains(contentAfterInstall, "source") {
		t.Fatalf("expected hook to be installed")
	}

	// Now uninstall
	buf2 := &bytes.Buffer{}
	err = uninstallShellIntegration("zsh", buf2)
	if err != nil {
		t.Fatalf("uninstall failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "✓ Updated") {
		t.Fatalf("expected success message, got: %q", output)
	}

	contentAfterUninstall := readFileOrDie(t, zshrc)
	if strings.Contains(contentAfterUninstall, ".aws-login/shell-init") {
		t.Fatalf("expected hook to be removed, but found in: %s", contentAfterUninstall)
	}
}

// Test helpers
func fileStatOrDie(t *testing.T, path string) (interface{}, error) {
	t.Helper()
	info, err := fileInfo(path)
	return info, err
}

func fileInfo(path string) (interface{}, error) {
	return os.Stat(path)
}

func readFileOrDie(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(content)
}

func createFileWithContent(path string, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
