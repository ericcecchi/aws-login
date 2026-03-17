package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunDoctorRestoresCorruptAWSConfig(t *testing.T) {
	home := setTempHome(t)
	awsConfig := filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(awsConfig), 0o755); err != nil {
		t.Fatalf("mkdir aws dir: %v", err)
	}
	if err := os.WriteFile(awsConfig, []byte("[profile broken\n"), 0o644); err != nil {
		t.Fatalf("write corrupt aws config: %v", err)
	}

	backupPath := expandPath(awsConfigBackupPath)
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup dir: %v", err)
	}
	restored := []byte("[profile restored]\nregion = us-east-1\n")
	if err := os.WriteFile(backupPath, restored, 0o644); err != nil {
		t.Fatalf("write backup config: %v", err)
	}

	buf := &bytes.Buffer{}
	if err := runDoctor(buf); err != nil {
		t.Fatalf("runDoctor error: %v", err)
	}

	data, err := os.ReadFile(awsConfig)
	if err != nil {
		t.Fatalf("read aws config: %v", err)
	}
	if string(data) != string(restored) {
		t.Fatalf("expected aws config restored from backup")
	}
	if !strings.Contains(buf.String(), "Doctor completed with repairs") {
		t.Fatalf("expected repair output, got: %s", buf.String())
	}
}

func TestRunDoctorFailsWithoutBackupForCorruptAWSConfig(t *testing.T) {
	home := setTempHome(t)
	awsConfig := filepath.Join(home, ".aws", "config")
	if err := os.MkdirAll(filepath.Dir(awsConfig), 0o755); err != nil {
		t.Fatalf("mkdir aws dir: %v", err)
	}
	if err := os.WriteFile(awsConfig, []byte("[profile broken\n"), 0o644); err != nil {
		t.Fatalf("write corrupt aws config: %v", err)
	}

	err := runDoctor(&bytes.Buffer{})
	if err == nil {
		t.Fatalf("expected runDoctor error when no backup exists")
	}
	if !strings.Contains(err.Error(), "no backup available") {
		t.Fatalf("expected missing backup error, got: %v", err)
	}
}
