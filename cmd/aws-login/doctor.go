package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

func runDoctor(w io.Writer) error {
	logLine(w, "Running aws-login doctor...")
	lock, err := acquireMutationLock(45 * time.Second)
	if err != nil {
		return err
	}
	defer lock.release()

	issues := []string{}
	repairs := []string{}

	awsPath := expandPath(awsConfigPath)
	if err := validateAWSConfigFile(awsPath); err != nil {
		logLine(w, fmt.Sprintf("Detected invalid AWS config at %s", awsPath))
		restored, repairErr := restoreAndValidate(awsPath, expandPath(awsConfigBackupPath), validateAWSConfigFile)
		if repairErr != nil {
			issues = append(issues, fmt.Sprintf("aws config: %v", repairErr))
		} else if restored {
			repairs = append(repairs, "AWS config restored from backup")
		}
	} else {
		logLine(w, "AWS config looks healthy")
	}

	kubePath := expandPath(kubeConfigPath)
	if _, err := os.Stat(kubePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logLine(w, "Kube config not found; skipping")
		} else {
			issues = append(issues, fmt.Sprintf("kube config stat failed: %v", err))
		}
	} else if err := validateKubeConfigFile(kubePath); err != nil {
		logLine(w, fmt.Sprintf("Detected invalid kube config at %s", kubePath))
		restored, repairErr := restoreAndValidate(kubePath, expandPath(kubeConfigBackupPath), validateKubeConfigFile)
		if repairErr != nil {
			issues = append(issues, fmt.Sprintf("kube config: %v", repairErr))
		} else if restored {
			repairs = append(repairs, "Kube config restored from backup")
		}
	} else {
		logLine(w, "Kube config looks healthy")
	}

	if len(repairs) > 0 {
		for _, repair := range repairs {
			logLine(w, fmt.Sprintf("Repaired: %s", repair))
		}
	}

	if len(issues) > 0 {
		return fmt.Errorf(strings.Join(issues, "; "))
	}

	if len(repairs) == 0 {
		logLine(w, "Doctor found no issues")
	} else {
		logLine(w, "Doctor completed with repairs")
	}
	return nil
}

func validateKubeConfigFile(path string) error {
	if !commandExists("kubectl") {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if strings.TrimSpace(string(data)) == "" {
			return fmt.Errorf("kube config file is empty")
		}
		return nil
	}

	cmd := exec.Command("kubectl", "config", "view", "--kubeconfig", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = "kubectl failed to parse config"
		}
		return fmt.Errorf(message)
	}
	return nil
}

func restoreAndValidate(path, backupPath string, validate func(string) error) (bool, error) {
	if _, err := os.Stat(backupPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, fmt.Errorf("no backup available")
		}
		return false, err
	}
	if err := restoreFromBackup(path, backupPath); err != nil {
		return false, err
	}
	if err := validate(path); err != nil {
		return true, fmt.Errorf("restored backup but file is still invalid")
	}
	return true, nil
}
