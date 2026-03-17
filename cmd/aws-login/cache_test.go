package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestListAccountsCachedUsesStaleWhileRevalidate(t *testing.T) {
	setTempHome(t)
	writeStubScripts(t, map[string]string{"aws": awsStubScript})
	callsFile := filepath.Join(t.TempDir(), "accounts_calls.txt")
	t.Setenv("AWS_LOGIN_TEST_ACCOUNTS_CALLS_FILE", callsFile)
	t.Setenv("AWS_LOGIN_TEST_ACCOUNTS_JSON", `{"accountList":[{"accountId":"123","accountName":"Old"}]}`)

	first, err := listAccountsCached("token", "us-east-1", "session:test")
	if err != nil {
		t.Fatalf("listAccountsCached first call error: %v", err)
	}
	if len(first) != 1 || first[0].AccountID != "123" {
		t.Fatalf("unexpected first accounts result: %+v", first)
	}

	t.Setenv("AWS_LOGIN_TEST_ACCOUNTS_JSON", `{"accountList":[{"accountId":"456","accountName":"New"}]}`)
	second, err := listAccountsCached("token", "us-east-1", "session:test")
	if err != nil {
		t.Fatalf("listAccountsCached second call error: %v", err)
	}
	if len(second) != 1 || second[0].AccountID != "123" {
		t.Fatalf("expected stale cache on second call, got %+v", second)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		updated, err := listAccountsCached("token", "us-east-1", "session:test")
		if err == nil && len(updated) == 1 && updated[0].AccountID == "456" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("cache did not revalidate in time; latest value: %+v", updated)
		}
		time.Sleep(25 * time.Millisecond)
	}

	data, err := os.ReadFile(callsFile)
	if err != nil {
		t.Fatalf("read calls file: %v", err)
	}
	if lines := nonEmptyLineCount(string(data)); lines < 2 {
		t.Fatalf("expected refresh call to happen, got %d call(s)", lines)
	}
}

func TestListRolesCachedUsesStaleWhileRevalidate(t *testing.T) {
	setTempHome(t)
	writeStubScripts(t, map[string]string{"aws": awsStubScript})
	callsFile := filepath.Join(t.TempDir(), "roles_calls.txt")
	t.Setenv("AWS_LOGIN_TEST_ROLES_CALLS_FILE", callsFile)
	t.Setenv("AWS_LOGIN_TEST_ROLES_JSON", `{"roleList":[{"roleName":"Admin"}]}`)

	first, err := listRolesCached("token", "us-east-1", "123", "session:test")
	if err != nil {
		t.Fatalf("listRolesCached first call error: %v", err)
	}
	if len(first) != 1 || first[0].RoleName != "Admin" {
		t.Fatalf("unexpected first roles result: %+v", first)
	}

	t.Setenv("AWS_LOGIN_TEST_ROLES_JSON", `{"roleList":[{"roleName":"ReadOnly"}]}`)
	second, err := listRolesCached("token", "us-east-1", "123", "session:test")
	if err != nil {
		t.Fatalf("listRolesCached second call error: %v", err)
	}
	if len(second) != 1 || second[0].RoleName != "Admin" {
		t.Fatalf("expected stale cache on second call, got %+v", second)
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		updated, err := listRolesCached("token", "us-east-1", "123", "session:test")
		if err == nil && len(updated) == 1 && updated[0].RoleName == "ReadOnly" {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("cache did not revalidate in time; latest value: %+v", updated)
		}
		time.Sleep(25 * time.Millisecond)
	}

	data, err := os.ReadFile(callsFile)
	if err != nil {
		t.Fatalf("read calls file: %v", err)
	}
	if lines := nonEmptyLineCount(string(data)); lines < 2 {
		t.Fatalf("expected refresh call to happen, got %d call(s)", lines)
	}
}

func nonEmptyLineCount(value string) int {
	count := 0
	for _, line := range strings.Split(value, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}
