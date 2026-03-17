package main

import (
	"bytes"
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

func TestMaybeSwitchKubeAuto(t *testing.T) {
	writeStubScripts(t, map[string]string{
		"aws":     awsStubScript,
		"kubectl": kubectlStubScript,
	})
	useContextFile := filepath.Join(t.TempDir(), "used.txt")
	t.Setenv("KUBECTL_TEST_USE_CONTEXT_FILE", useContextFile)
	t.Setenv("KUBECTL_TEST_CONTEXTS", "zeta\naws-123\ncluster-a\n")
	t.Setenv("AWS_LOGIN_TEST_EKS_JSON", `{"clusters":["cluster-a","cluster-b"]}`)

	buf := &bytes.Buffer{}
	maybeSwitchKubeAuto("123", "us-east-1", "", "test-profile", "", buf)

	data, err := os.ReadFile(useContextFile)
	if err != nil {
		t.Fatalf("expected use-context to be called: %v", err)
	}
	used := strings.TrimSpace(string(data))
	if used != "aws-123" {
		t.Fatalf("expected context aws-123, got %q", used)
	}
}
