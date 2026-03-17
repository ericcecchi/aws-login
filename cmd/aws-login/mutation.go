package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const (
	kubeConfigPath = "~/.kube/config"
)

var (
	mutationLockPath     = "~/.aws-login/locks/mutation.lock"
	awsConfigBackupPath  = "~/.aws-login/backups/aws-config.latest"
	kubeConfigBackupPath = "~/.aws-login/backups/kube-config.latest"
)

type mutationLock struct {
	path string
}

type mutationBackupState struct {
	awsBackupCreated  bool
	kubeBackupCreated bool
}

func withMutationGuard(includeKube bool, fn func() error) error {
	lock, err := acquireMutationLock(45 * time.Second)
	if err != nil {
		return err
	}
	defer lock.release()

	backupState, err := createMutationBackups(includeKube)
	if err != nil {
		return err
	}

	if err := fn(); err != nil {
		_ = backupState.restore(includeKube)
		return err
	}

	if err := validateMutationResults(includeKube); err != nil {
		_ = backupState.restore(includeKube)
		return fmt.Errorf("recovered from config corruption: %w", err)
	}

	return nil
}

func acquireMutationLock(timeout time.Duration) (mutationLock, error) {
	path := expandPath(mutationLockPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return mutationLock{}, fmt.Errorf("failed to create lock directory: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%d\n%d\n", os.Getpid(), time.Now().Unix())
			_ = f.Close()
			return mutationLock{path: path}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return mutationLock{}, fmt.Errorf("failed to acquire mutation lock: %w", err)
		}

		stale, staleErr := isStaleLock(path, 2*time.Minute)
		if staleErr == nil && stale {
			_ = os.Remove(path)
			continue
		}
		if time.Now().After(deadline) {
			return mutationLock{}, fmt.Errorf("another aws-login process is updating configs; timed out waiting for lock")
		}
		time.Sleep(200 * time.Millisecond)
	}
}

func isStaleLock(path string, maxAge time.Duration) (bool, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return time.Since(stat.ModTime()) > maxAge, nil
}

func (l mutationLock) release() {
	if l.path == "" {
		return
	}
	_ = os.Remove(l.path)
}

func createMutationBackups(includeKube bool) (mutationBackupState, error) {
	state := mutationBackupState{}

	if copied, err := backupFileIfExists(expandPath(awsConfigPath), expandPath(awsConfigBackupPath)); err != nil {
		return state, err
	} else {
		state.awsBackupCreated = copied
	}

	if includeKube {
		if copied, err := backupFileIfExists(expandPath(kubeConfigPath), expandPath(kubeConfigBackupPath)); err != nil {
			return state, err
		} else {
			state.kubeBackupCreated = copied
		}
	}

	return state, nil
}

func (s mutationBackupState) restore(includeKube bool) error {
	if s.awsBackupCreated {
		if err := restoreFromBackup(expandPath(awsConfigPath), expandPath(awsConfigBackupPath)); err != nil {
			return err
		}
	}
	if includeKube && s.kubeBackupCreated {
		if err := restoreFromBackup(expandPath(kubeConfigPath), expandPath(kubeConfigBackupPath)); err != nil {
			return err
		}
	}
	return nil
}

func validateMutationResults(includeKube bool) error {
	if err := validateAWSConfigFile(expandPath(awsConfigPath)); err != nil {
		return fmt.Errorf("aws config became unreadable")
	}

	if includeKube {
		path := expandPath(kubeConfigPath)
		if _, err := os.Stat(path); err == nil && commandExists("kubectl") {
			cmd := exec.Command("kubectl", "config", "view", "--kubeconfig", path)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("kube config became unreadable")
			}
		}
	}
	return nil
}

func backupFileIfExists(sourcePath, backupPath string) (bool, error) {
	if _, err := os.Stat(sourcePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		return false, err
	}
	if err := copyFile(sourcePath, backupPath, 0o600); err != nil {
		return false, err
	}
	return true, nil
}

func restoreFromBackup(destinationPath, backupPath string) error {
	if _, err := os.Stat(backupPath); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destinationPath), 0o755); err != nil {
		return err
	}
	return copyFile(backupPath, destinationPath, 0o600)
}

func copyFile(sourcePath, destinationPath string, mode os.FileMode) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(destinationPath, data, mode); err != nil {
		return err
	}
	return nil
}
