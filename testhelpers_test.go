package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setTempHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	return home
}

func writeStubScripts(t *testing.T, scripts map[string]string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("stub scripts use POSIX shell")
	}
	dir := t.TempDir()
	for name, body := range scripts {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
			t.Fatalf("write stub %s: %v", name, err)
		}
	}
	origPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+origPath)
	return dir
}

func mapFromEnvList(entries []string) map[string]string {
	out := make(map[string]string, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}

const awsStubScript = "#!/usr/bin/env bash\n" +
	"set -e\n" +
	"cmd=\"${1:-}\"\n" +
	"sub=\"${2:-}\"\n" +
	"if [[ \"$cmd\" == \"sso\" && \"$sub\" == \"list-accounts\" ]]; then\n" +
	"  printf '%s' \"${AWS_LOGIN_TEST_ACCOUNTS_JSON}\"\n" +
	"  exit 0\n" +
	"fi\n" +
	"if [[ \"$cmd\" == \"sso\" && \"$sub\" == \"list-account-roles\" ]]; then\n" +
	"  printf '%s' \"${AWS_LOGIN_TEST_ROLES_JSON}\"\n" +
	"  exit 0\n" +
	"fi\n" +
	"if [[ \"$cmd\" == \"sso\" && \"$sub\" == \"get-role-credentials\" ]]; then\n" +
	"  printf '%s' \"${AWS_LOGIN_TEST_CREDS_JSON}\"\n" +
	"  exit 0\n" +
	"fi\n" +
	"if [[ \"$cmd\" == \"sso\" && \"$sub\" == \"login\" ]]; then\n" +
	"  if [[ -n \"${AWS_LOGIN_TEST_SSO_CACHE_DIR:-}\" ]]; then\n" +
	"    mkdir -p \"$AWS_LOGIN_TEST_SSO_CACHE_DIR\"\n" +
	"    cat > \"$AWS_LOGIN_TEST_SSO_CACHE_DIR/token.json\" <<JSON\n" +
	"{\"startUrl\":\"${AWS_LOGIN_TEST_SSO_START_URL}\",\"accessToken\":\"${AWS_LOGIN_TEST_SSO_TOKEN}\",\"expiresAt\":\"${AWS_LOGIN_TEST_SSO_EXPIRY}\"}\n" +
	"JSON\n" +
	"  fi\n" +
	"  exit 0\n" +
	"fi\n" +
	"if [[ \"$cmd\" == \"eks\" && \"$sub\" == \"list-clusters\" ]]; then\n" +
	"  printf '%s' \"${AWS_LOGIN_TEST_EKS_JSON}\"\n" +
	"  exit 0\n" +
	"fi\n" +
	"if [[ \"$cmd\" == \"eks\" && \"$sub\" == \"update-kubeconfig\" ]]; then\n" +
	"  exit 0\n" +
	"fi\n" +
	"if [[ \"$cmd\" == \"sts\" && \"$sub\" == \"get-caller-identity\" ]]; then\n" +
	"  printf '%s' \"${AWS_LOGIN_TEST_IDENTITY_JSON}\"\n" +
	"  exit 0\n" +
	"fi\n" +
	"if [[ \"$cmd\" == \"configure\" && \"$sub\" == \"sso\" ]]; then\n" +
	"  exit 0\n" +
	"fi\n" +
	"echo \"unexpected aws args: $*\" >&2\n" +
	"exit 1\n"

const kubectlStubScript = "#!/usr/bin/env bash\n" +
	"set -e\n" +
	"cmd=\"${1:-}\"\n" +
	"sub=\"${2:-}\"\n" +
	"if [[ \"$cmd\" == \"config\" && \"$sub\" == \"get-contexts\" ]]; then\n" +
	"  printf '%s' \"${KUBECTL_TEST_CONTEXTS}\"\n" +
	"  exit 0\n" +
	"fi\n" +
	"if [[ \"$cmd\" == \"config\" && \"$sub\" == \"use-context\" ]]; then\n" +
	"  if [[ -n \"${KUBECTL_TEST_USE_CONTEXT_FILE:-}\" ]]; then\n" +
	"    echo \"${3:-}\" >> \"$KUBECTL_TEST_USE_CONTEXT_FILE\"\n" +
	"  fi\n" +
	"  exit 0\n" +
	"fi\n" +
	"echo \"unexpected kubectl args: $*\" >&2\n" +
	"exit 1\n"
