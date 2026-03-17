package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireMutationLockSerializesWriters(t *testing.T) {
	setTempHome(t)

	first, err := acquireMutationLock(100 * time.Millisecond)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	defer first.release()

	if _, err := acquireMutationLock(250 * time.Millisecond); err == nil {
		t.Fatalf("expected second lock acquisition to time out")
	}

	first.release()
	second, err := acquireMutationLock(250 * time.Millisecond)
	if err != nil {
		t.Fatalf("expected lock after release: %v", err)
	}
	second.release()
}

func TestWithMutationGuardRestoresCorruptAWSConfig(t *testing.T) {
	home := setTempHome(t)
	configPath := filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	original := []byte("[profile ok]\nregion = us-east-1\n")
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatalf("write original config: %v", err)
	}

	err := withMutationGuard(false, func() error {
		return os.WriteFile(configPath, []byte("[profile broken\n"), 0o644)
	})
	if err == nil {
		t.Fatalf("expected validation failure from corrupted config")
	}

	restored, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read restored config: %v", err)
	}
	if string(restored) != string(original) {
		t.Fatalf("expected config to be restored from backup")
	}
}
