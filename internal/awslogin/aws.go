package awslogin

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/ini.v1"
)

func resolveSession(cfg *ini.File, w io.Writer, ssoSessionFlag, profileFlag string, nonInteractive bool) (SessionInfo, error) {
	cfg = ensureSSOConfigured(cfg, w, nonInteractive)
	if cfg == nil {
		return SessionInfo{}, fmt.Errorf("no AWS SSO sessions found; run 'aws configure sso-session'")
	}

	sessions := listSSOSessions(cfg)
	profileChoice := strings.TrimSpace(profileFlag)

	if ssoSessionFlag != "" {
		session, ok := sessions[ssoSessionFlag]
		if !ok {
			return SessionInfo{}, fmt.Errorf("SSO session '%s' not found in %s", ssoSessionFlag, awsConfigPath)
		}
		session.LoginArgs = []string{"--sso-session", ssoSessionFlag}
		return session, nil
	}

	if profileChoice != "" {
		info, found, err := getProfileInfoIfExists(cfg, profileChoice)
		if err != nil {
			return SessionInfo{}, err
		}
		if found {
			if info.SSOSession != "" {
				session, ok := sessions[info.SSOSession]
				if !ok {
					return SessionInfo{}, fmt.Errorf("SSO session '%s' not found for profile '%s'", info.SSOSession, profileChoice)
				}
				session.Name = info.SSOSession
				session.LoginArgs = []string{"--sso-session", info.SSOSession}
				return session, nil
			}
			if info.SSOStart != "" && info.SSORegion != "" {
				return SessionInfo{
					Name:      "",
					StartURL:  info.SSOStart,
					Region:    info.SSORegion,
					LoginArgs: []string{"--profile", profileChoice},
				}, nil
			}
			return SessionInfo{}, fmt.Errorf("profile '%s' is not an SSO profile", profileChoice)
		}
	}

	defaultInfo, found, err := getProfileInfoIfExists(cfg, "default")
	if err != nil {
		return SessionInfo{}, err
	}
	if found {
		if defaultInfo.SSOSession != "" {
			session, ok := sessions[defaultInfo.SSOSession]
			if !ok {
				return SessionInfo{}, fmt.Errorf("SSO session '%s' not found for profile 'default'", defaultInfo.SSOSession)
			}
			session.Name = defaultInfo.SSOSession
			session.LoginArgs = []string{"--sso-session", defaultInfo.SSOSession}
			return session, nil
		}
		if defaultInfo.SSOStart != "" && defaultInfo.SSORegion != "" {
			return SessionInfo{
				Name:      "",
				StartURL:  defaultInfo.SSOStart,
				Region:    defaultInfo.SSORegion,
				LoginArgs: []string{"--profile", "default"},
			}, nil
		}
	}

	if len(sessions) == 1 {
		for name, session := range sessions {
			session.Name = name
			session.LoginArgs = []string{"--sso-session", name}
			return session, nil
		}
	}

	if len(sessions) > 1 {
		if nonInteractive {
			return SessionInfo{}, fmt.Errorf("multiple SSO sessions available; specify --sso-session or --profile")
		}
		names := make([]string, 0, len(sessions))
		for name := range sessions {
			names = append(names, name)
		}
		sort.Strings(names)
		chosen, err := chooseInteractive(names, "AWS SSO session", func(name string) string {
			session := sessions[name]
			return fmt.Sprintf("%s (%s)", name, session.StartURL)
		})
		if err != nil {
			return SessionInfo{}, err
		}
		session := sessions[chosen]
		session.LoginArgs = []string{"--sso-session", chosen}
		return session, nil
	}

	return SessionInfo{}, fmt.Errorf("no AWS SSO sessions found in %s", awsConfigPath)
}

func ensureSSOConfigured(cfg *ini.File, w io.Writer, nonInteractive bool) *ini.File {
	if len(listSSOSessions(cfg)) > 0 {
		return cfg
	}

	if nonInteractive {
		return nil
	}

	if !commandExists("aws") {
		return nil
	}

	logLine(w, "No AWS SSO sessions found. Launching 'aws configure sso-session'...")
	cmd := exec.Command("aws", "configure", "sso-session")
	cmd.Stdout = w
	cmd.Stderr = w
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return nil
	}

	updated, err := loadAWSConfig()
	if err != nil {
		return nil
	}
	if len(listSSOSessions(updated)) == 0 {
		return nil
	}
	return updated
}

func ensureLoggedIn(session SessionInfo, w io.Writer) (string, error) {
	if token := findCachedToken(session.StartURL); token != "" {
		return token, nil
	}

	logLine(w, "🔐 Authenticating with AWS SSO...")
	cmd := exec.Command("aws", append([]string{"sso", "login"}, session.LoginArgs...)...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("AWS SSO login failed")
	}

	if token := findCachedToken(session.StartURL); token != "" {
		return token, nil
	}
	return "", fmt.Errorf("AWS SSO login succeeded but no access token found")
}

func findCachedToken(startURL string) string {
	cacheDir := expandPath(awsSSOCacheDir)
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return ""
	}

	var bestToken string
	var bestExpiry time.Time
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(cacheDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var cacheEntry SSOCacheEntry
		if err := json.Unmarshal(data, &cacheEntry); err != nil {
			continue
		}
		if cacheEntry.StartURL != startURL {
			continue
		}
		expiry := parseExpiry(cacheEntry.ExpiresAt)
		if expiry.IsZero() || expiry.Before(time.Now().UTC()) {
			continue
		}
		if bestToken == "" || expiry.After(bestExpiry) {
			bestToken = cacheEntry.AccessToken
			bestExpiry = expiry
		}
	}
	return bestToken
}

func parseExpiry(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	normalized := strings.TrimSpace(value)
	if strings.HasSuffix(normalized, "UTC") {
		normalized = strings.TrimSuffix(normalized, "UTC") + "Z"
	}
	if strings.HasSuffix(normalized, "Z") {
		if t, err := time.Parse(time.RFC3339, normalized); err == nil {
			return t
		}
	}
	if t, err := time.Parse(time.RFC3339, normalized); err == nil {
		return t
	}
	return time.Time{}
}

func listAccounts(accessToken, region string) ([]AccountInfo, error) {
	payload, err := awsCLIJSON([]string{"sso", "list-accounts", "--access-token", accessToken, "--region", region})
	if err != nil {
		return nil, err
	}
	var response struct {
		AccountList []struct {
			AccountID    string `json:"accountId"`
			AccountName  string `json:"accountName"`
			EmailAddress string `json:"emailAddress"`
		} `json:"accountList"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse account list")
	}
	accounts := make([]AccountInfo, 0, len(response.AccountList))
	for _, acct := range response.AccountList {
		if acct.AccountID == "" {
			continue
		}
		name := acct.AccountName
		if name == "" {
			name = acct.AccountID
		}
		accounts = append(accounts, AccountInfo{AccountID: acct.AccountID, AccountName: name, Email: acct.EmailAddress})
	}
	return accounts, nil
}

func listRoles(accessToken, region, accountID string) ([]RoleInfo, error) {
	payload, err := awsCLIJSON([]string{"sso", "list-account-roles", "--access-token", accessToken, "--region", region, "--account-id", accountID})
	if err != nil {
		return nil, err
	}
	var response struct {
		RoleList []struct {
			RoleName string `json:"roleName"`
		} `json:"roleList"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return nil, fmt.Errorf("failed to parse role list")
	}
	roles := make([]RoleInfo, 0, len(response.RoleList))
	for _, role := range response.RoleList {
		if role.RoleName == "" {
			continue
		}
		roles = append(roles, RoleInfo{RoleName: role.RoleName})
	}
	return roles, nil
}

func getRoleCredentials(accessToken, region, accountID, roleName string) (RoleCredentials, error) {
	payload, err := awsCLIJSON([]string{
		"sso", "get-role-credentials",
		"--access-token", accessToken,
		"--region", region,
		"--account-id", accountID,
		"--role-name", roleName,
	})
	if err != nil {
		return RoleCredentials{}, err
	}
	var response struct {
		RoleCredentials struct {
			AccessKeyID     string `json:"accessKeyId"`
			SecretAccessKey string `json:"secretAccessKey"`
			SessionToken    string `json:"sessionToken"`
			Expiration      int64  `json:"expiration"`
		} `json:"roleCredentials"`
	}
	if err := json.Unmarshal(payload, &response); err != nil {
		return RoleCredentials{}, fmt.Errorf("failed to parse role credentials")
	}
	if response.RoleCredentials.AccessKeyID == "" {
		return RoleCredentials{}, fmt.Errorf("failed to retrieve role credentials")
	}
	var expiration *time.Time
	if response.RoleCredentials.Expiration > 0 {
		exp := time.UnixMilli(response.RoleCredentials.Expiration).UTC()
		expiration = &exp
	}
	return RoleCredentials{
		AccessKeyID:     response.RoleCredentials.AccessKeyID,
		SecretAccessKey: response.RoleCredentials.SecretAccessKey,
		SessionToken:    response.RoleCredentials.SessionToken,
		Expiration:      expiration,
	}, nil
}

func awsCLIJSON(args []string) ([]byte, error) {
	cmd := exec.Command("aws", append(args, "--output", "json", "--no-cli-pager")...)
	output, err := cmd.Output()
	if err == nil {
		return output, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return nil, fmt.Errorf("aws command failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
	}
	return nil, fmt.Errorf("aws command failed")
}
