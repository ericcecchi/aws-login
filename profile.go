package main

import (
	"fmt"
	"regexp"
	"strings"
)

func buildProfileName(account AccountInfo, role RoleInfo) string {
	rolePart := sanitizeProfilePart(role.RoleName)
	accountPart := sanitizeProfilePart(account.AccountName)
	if accountPart == "" {
		accountPart = account.AccountID
	}
	return fmt.Sprintf("aws-login-%s-%s", accountPart, rolePart)
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

func writeProfile(profileName, region string, creds RoleCredentials) error {
	config, err := loadAWSConfig()
	if err != nil {
		return err
	}
	credentials, credentialsPath, err := loadAWSCredentials()
	if err != nil {
		return err
	}

	configSection := "profile " + profileName
	cfgSection := config.Section(configSection)
	if region != "" {
		cfgSection.Key("region").SetValue(region)
	}
	cfgSection.Key("output").SetValue("json")

	credsSection := credentials.Section(profileName)
	credsSection.Key("aws_access_key_id").SetValue(creds.AccessKeyID)
	credsSection.Key("aws_secret_access_key").SetValue(creds.SecretAccessKey)
	credsSection.Key("aws_session_token").SetValue(creds.SessionToken)

	if err := config.SaveTo(expandPath(awsConfigPath)); err != nil {
		return err
	}
	if err := credentials.SaveTo(credentialsPath); err != nil {
		return err
	}
	return nil
}
