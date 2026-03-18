package awslogin

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

const (
	awsConfigPath = "~/.aws/config"
)

func loadAWSConfig() (*ini.File, error) {
	path := expandPath(awsConfigPath)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, fmt.Errorf("unable to create %s", path)
		}
		if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
			return nil, fmt.Errorf("unable to create %s", path)
		}
	}
	cfg, err := ini.Load(path)
	if err != nil {
		restored, restoreErr := tryRestoreAWSConfig(path)
		if restoreErr == nil && restored {
			cfg, err = ini.Load(path)
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read %s", path)
		}
	}
	if err := validateAWSConfigFile(path); err != nil {
		restored, restoreErr := tryRestoreAWSConfig(path)
		if restoreErr == nil && restored {
			cfg, err = ini.Load(path)
			if err == nil {
				if validateErr := validateAWSConfigFile(path); validateErr == nil {
					return cfg, nil
				}
			}
		}
		return nil, fmt.Errorf("unable to read %s", path)
	}
	return cfg, nil
}

func tryRestoreAWSConfig(configPath string) (bool, error) {
	backupPath := expandPath(awsConfigBackupPath)
	if _, err := os.Stat(backupPath); err != nil {
		return false, err
	}
	if err := restoreFromBackup(configPath, backupPath); err != nil {
		return false, err
	}
	return true, nil
}

func validateAWSConfigFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if _, err := ini.Load(path); err != nil {
		return fmt.Errorf("unable to parse AWS config")
	}
	if !commandExists("aws") {
		return nil
	}
	cmd := exec.Command("aws", "configure", "list-profiles")
	cmd.Env = append(os.Environ(), "AWS_CONFIG_FILE="+path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = "aws cli could not parse config"
		}
		return fmt.Errorf(message)
	}
	return nil
}

func listSSOSessions(cfg *ini.File) map[string]SessionInfo {
	out := map[string]SessionInfo{}
	for _, section := range cfg.Sections() {
		name := section.Name()
		if !strings.HasPrefix(name, "sso-session ") {
			continue
		}
		sessionName := strings.TrimSpace(strings.TrimPrefix(name, "sso-session "))
		startURL := section.Key("sso_start_url").String()
		region := section.Key("sso_region").String()
		if sessionName != "" && startURL != "" && region != "" {
			out[sessionName] = SessionInfo{Name: sessionName, StartURL: startURL, Region: region}
		}
	}
	return out
}

func getProfileInfoIfExists(cfg *ini.File, profile string) (ProfileInfo, bool, error) {
	if strings.TrimSpace(profile) == "" {
		return ProfileInfo{}, false, nil
	}
	sectionName := "profile " + profile
	section, err := cfg.GetSection(sectionName)
	if err != nil {
		if profile == "default" {
			section, err = cfg.GetSection("default")
		}
	}
	if err != nil {
		return ProfileInfo{}, false, nil
	}
	return ProfileInfo{
		SSOSession: section.Key("sso_session").String(),
		SSOStart:   section.Key("sso_start_url").String(),
		SSORegion:  section.Key("sso_region").String(),
		Region:     section.Key("region").String(),
		AccountID:  section.Key("sso_account_id").String(),
		RoleName:   section.Key("sso_role_name").String(),
		EKSRoleARN: section.Key("eks_role_arn").String(),
	}, true, nil
}
