package awslogin

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListKubeContexts(t *testing.T) {
	writeStubScripts(t, map[string]string{"kubectl": kubectlStubScript})
	t.Setenv("KUBECTL_TEST_CONTEXTS", "zeta\nalpha\n")
	contexts, err := listKubeContexts()
	if err != nil {
		t.Fatalf("listKubeContexts error: %v", err)
	}
	if len(contexts) != 2 || contexts[0] != "alpha" || contexts[1] != "zeta" {
		t.Fatalf("unexpected contexts: %v", contexts)
	}
}

func TestFilterContexts(t *testing.T) {
	contexts := []string{"ctx-123", "cluster-a", "other"}
	clusters := []string{"cluster-a", "cluster-b"}
	matches := filterContexts(contexts, "123", clusters)
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if matches[0] != "ctx-123" || matches[1] != "cluster-a" {
		t.Fatalf("unexpected matches: %v", matches)
	}
}

// TestMaybeSwitchKubeAutoSingleMatch verifies that when only one context matches
// the account, it is selected automatically without any interaction.
func TestMaybeSwitchKubeAutoSingleMatch(t *testing.T) {
	setTempHome(t)
	writeStubScripts(t, map[string]string{
		"aws":     awsStubScript,
		"kubectl": kubectlStubScript,
	})
	useContextFile := filepath.Join(t.TempDir(), "used.txt")
	t.Setenv("KUBECTL_TEST_USE_CONTEXT_FILE", useContextFile)
	// Only one context matches accountID "123"
	t.Setenv("KUBECTL_TEST_CONTEXTS", "zeta\naws-123\n")
	t.Setenv("AWS_LOGIN_TEST_EKS_JSON", `{"clusters":["cluster-b"]}`)

	buf := &bytes.Buffer{}
	maybeSwitchKubeAuto("123", "us-east-1", "", "test-profile", "", false, buf)

	data, err := os.ReadFile(useContextFile)
	if err != nil {
		t.Fatalf("expected use-context to be called: %v", err)
	}
	used := strings.TrimSpace(string(data))
	if used != "aws-123" {
		t.Fatalf("expected context aws-123, got %q", used)
	}
}

// TestMaybeSwitchKubeAutoWithSavedPreference verifies that when multiple contexts
// match and a preference has been saved for the account, the saved preference is used.
func TestMaybeSwitchKubeAutoWithSavedPreference(t *testing.T) {
	home := setTempHome(t)
	writeStubScripts(t, map[string]string{
		"aws":     awsStubScript,
		"kubectl": kubectlStubScript,
	})
	useContextFile := filepath.Join(t.TempDir(), "used.txt")
	t.Setenv("KUBECTL_TEST_USE_CONTEXT_FILE", useContextFile)
	t.Setenv("KUBECTL_TEST_CONTEXTS", "aws-123\ncluster-a\n")
	t.Setenv("AWS_LOGIN_TEST_EKS_JSON", `{"clusters":["cluster-a","cluster-b"]}`)

	// Pre-save a preference for account "123"
	prefsPath := filepath.Join(home, ".aws-login", "kube-prefs.json")
	if err := os.MkdirAll(filepath.Dir(prefsPath), 0o700); err != nil {
		t.Fatal(err)
	}
	prefs, _ := json.Marshal(map[string]string{"123": "cluster-a"})
	if err := os.WriteFile(prefsPath, prefs, 0o600); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	maybeSwitchKubeAuto("123", "us-east-1", "", "test-profile", "", false, buf)

	data, err := os.ReadFile(useContextFile)
	if err != nil {
		t.Fatalf("expected use-context to be called: %v", err)
	}
	used := strings.TrimSpace(string(data))
	if used != "cluster-a" {
		t.Fatalf("expected saved preference cluster-a, got %q", used)
	}
	if !strings.Contains(buf.String(), "saved Kubernetes context") {
		t.Fatalf("expected saved-preference message in output, got: %s", buf.String())
	}
}

// TestMaybeSwitchKubeAutoNonInteractiveMultiple verifies that when multiple contexts
// match and there is no saved preference, non-interactive mode skips the switch.
func TestMaybeSwitchKubeAutoNonInteractiveMultiple(t *testing.T) {
	setTempHome(t)
	writeStubScripts(t, map[string]string{
		"aws":     awsStubScript,
		"kubectl": kubectlStubScript,
	})
	useContextFile := filepath.Join(t.TempDir(), "used.txt")
	t.Setenv("KUBECTL_TEST_USE_CONTEXT_FILE", useContextFile)
	t.Setenv("KUBECTL_TEST_CONTEXTS", "aws-123\ncluster-a\n")
	t.Setenv("AWS_LOGIN_TEST_EKS_JSON", `{"clusters":["cluster-a","cluster-b"]}`)

	buf := &bytes.Buffer{}
	maybeSwitchKubeAuto("123", "us-east-1", "", "test-profile", "", true, buf)

	if _, err := os.ReadFile(useContextFile); err == nil {
		t.Fatal("expected use-context not to be called in non-interactive mode")
	}
	if !strings.Contains(buf.String(), "Multiple Kubernetes contexts") {
		t.Fatalf("expected warning about multiple contexts, got: %s", buf.String())
	}
}

// TestMaybeSwitchKubeAutoExplicitContext verifies that an explicit --kube-context
// is used directly and saved as the account preference.
func TestMaybeSwitchKubeAutoExplicitContext(t *testing.T) {
	home := setTempHome(t)
	writeStubScripts(t, map[string]string{"kubectl": kubectlStubScript})
	useContextFile := filepath.Join(t.TempDir(), "used.txt")
	t.Setenv("KUBECTL_TEST_USE_CONTEXT_FILE", useContextFile)

	buf := &bytes.Buffer{}
	maybeSwitchKubeAuto("123", "us-east-1", "my-explicit-context", "test-profile", "", false, buf)

	data, err := os.ReadFile(useContextFile)
	if err != nil {
		t.Fatalf("expected use-context to be called: %v", err)
	}
	if strings.TrimSpace(string(data)) != "my-explicit-context" {
		t.Fatalf("expected my-explicit-context, got %q", string(data))
	}

	// Preference should have been saved.
	pref, err := loadKubePref("123")
	if err != nil {
		t.Fatalf("loadKubePref: %v (home=%s)", err, home)
	}
	if pref != "my-explicit-context" {
		t.Fatalf("expected saved preference my-explicit-context, got %q", pref)
	}
}

// TestSaveAndLoadKubePref verifies round-trip persistence of context preferences.
func TestSaveAndLoadKubePref(t *testing.T) {
	setTempHome(t)

	if err := saveKubePref("111", "ctx-a"); err != nil {
		t.Fatalf("saveKubePref: %v", err)
	}
	if err := saveKubePref("222", "ctx-b"); err != nil {
		t.Fatalf("saveKubePref: %v", err)
	}

	pref, err := loadKubePref("111")
	if err != nil || pref != "ctx-a" {
		t.Fatalf("expected ctx-a, got %q (err %v)", pref, err)
	}
	pref, err = loadKubePref("222")
	if err != nil || pref != "ctx-b" {
		t.Fatalf("expected ctx-b, got %q (err %v)", pref, err)
	}

	// Overwrite should update without clobbering other entries.
	if err := saveKubePref("111", "ctx-updated"); err != nil {
		t.Fatalf("saveKubePref update: %v", err)
	}
	pref, _ = loadKubePref("111")
	if pref != "ctx-updated" {
		t.Fatalf("expected ctx-updated, got %q", pref)
	}
	pref, _ = loadKubePref("222")
	if pref != "ctx-b" {
		t.Fatalf("expected ctx-b still intact, got %q", pref)
	}
}
