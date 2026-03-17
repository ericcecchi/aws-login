package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/ini.v1"
)

func buildProfileName(account AccountInfo, role RoleInfo) string {
	rolePart := sanitizeProfilePart(role.RoleName)
	accountID := sanitizeProfilePart(account.AccountID)
	if accountID == "" {
		accountID = "account"
	}
	return fmt.Sprintf("aws-login-%s-%s", accountID, rolePart)
}

func sanitizeProfilePart(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return ""
	}
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	clean := re.ReplaceAllString(lower, "-")
	clean = strings.Trim(clean, "-")
	clean = strings.ReplaceAll(clean, "--", "-")
	return clean
}

func configureProfile(profileName, region string, session SessionInfo, accountID, roleName string) error {
	if profileName == "" {
		return fmt.Errorf("profile name is required")
	}

	if session.Name != "" {
		if err := awsConfigureSet("profile."+profileName+".sso_session", session.Name); err != nil {
			return err
		}
	} else {
		if err := awsConfigureSet("profile."+profileName+".sso_start_url", session.StartURL); err != nil {
			return err
		}
		if err := awsConfigureSet("profile."+profileName+".sso_region", session.Region); err != nil {
			return err
		}
	}

	if err := awsConfigureSet("profile."+profileName+".sso_account_id", accountID); err != nil {
		return err
	}
	if err := awsConfigureSet("profile."+profileName+".sso_role_name", roleName); err != nil {
		return err
	}
	if err := awsConfigureSet("profile."+profileName+".region", region); err != nil {
		return err
	}
	if err := awsConfigureSet("profile."+profileName+".output", "json"); err != nil {
		return err
	}
	return nil
}

func ensureReusableSSOSession(cfg *ini.File, session SessionInfo) (SessionInfo, error) {
	if session.Name != "" {
		return session, nil
	}
	if strings.TrimSpace(session.StartURL) == "" || strings.TrimSpace(session.Region) == "" {
		return session, nil
	}

	sessions := listSSOSessions(cfg)
	names := make([]string, 0, len(sessions))
	for name := range sessions {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		existing := sessions[name]
		if existing.StartURL == session.StartURL && existing.Region == session.Region {
			session.Name = name
			session.LoginArgs = []string{"--sso-session", name}
			return session, nil
		}
	}

	base := "aws-login"
	chosen := base
	for i := 2; ; i++ {
		if _, ok := sessions[chosen]; !ok {
			break
		}
		chosen = fmt.Sprintf("%s-%d", base, i)
	}

	if err := awsConfigureSet("sso-session."+chosen+".sso_start_url", session.StartURL); err != nil {
		return SessionInfo{}, err
	}
	if err := awsConfigureSet("sso-session."+chosen+".sso_region", session.Region); err != nil {
		return SessionInfo{}, err
	}

	session.Name = chosen
	session.LoginArgs = []string{"--sso-session", chosen}
	return session, nil
}

func awsConfigureSet(setting, value string) error {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	cmd := exec.Command("aws", "configure", "set", setting, value)
	output, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = "unknown error"
		}
		return fmt.Errorf("aws configure set %s failed: %s", setting, message)
	}
	return nil
}
