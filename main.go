package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	args, err := parseArgs(os.Args[1:])
	if err != nil {
		printUsage(os.Stderr)
		os.Exit(1)
	}

	if args.Version {
		fmt.Println(version)
		return
	}

	if args.ShellInit {
		fmt.Print(shellInitScript(detectShell()))
		return
	}

	writer := logWriter(args.PrintEnv)

	if !commandExists("aws") {
		logLine(writer, "Error: AWS CLI is not installed or not in PATH")
		os.Exit(1)
	}

	configPath := os.Getenv("AWS_LOGIN_CONFIG")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	configPath = expandPath(configPath)

	awsConfig, err := loadAWSConfig()
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	maybeBootstrapConfig(configPath, awsConfig, writer)
	userConfig, err := loadUserConfig(configPath)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	aliasName := strings.TrimSpace(args.Alias)
	if aliasName == "" && args.Target != "" {
		if _, ok := userConfig.Aliases[args.Target]; ok {
			aliasName = args.Target
			args.Target = ""
		}
	}

	aliasConfig, hasAlias := userConfig.Aliases[aliasName]
	if aliasName != "" && !hasAlias {
		logLine(writer, fmt.Sprintf("Error: alias '%s' not found", aliasName))
		os.Exit(1)
	}

	selectedAccount := strings.TrimSpace(args.Account)
	if selectedAccount == "" {
		selectedAccount = strings.TrimSpace(args.Target)
	}
	selectedRole := strings.TrimSpace(args.Role)

	var profileName string
	kubeContext := strings.TrimSpace(args.KubeContext)
	regionOverride := strings.TrimSpace(args.Region)

	if hasAlias {
		aliasAccount, aliasRole, aliasProfile, aliasKube, aliasRegion, err := resolveAlias(aliasConfig, selectedRole)
		if err != nil {
			logLine(writer, fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		selectedAccount = aliasAccount
		selectedRole = aliasRole
		profileName = aliasProfile
		if kubeContext == "" {
			kubeContext = aliasKube
		}
		if regionOverride == "" {
			regionOverride = aliasRegion
		}
	}

	if args.NonInteractive && (selectedAccount == "" || selectedRole == "") {
		logLine(writer, "Error: missing account or role in non-interactive mode")
		os.Exit(1)
	}

	session, err := resolveSession(awsConfig, userConfig, writer, args.SSOSession, args.Profile, args.NonInteractive)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	accessToken, err := ensureLoggedIn(session, writer)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	accounts, err := listAccounts(accessToken, session.Region)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	account, err := resolveAccount(accounts, selectedAccount, writer, args.NonInteractive)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	roles, err := listRoles(accessToken, session.Region, account.AccountID)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	role, err := resolveRole(roles, selectedRole, writer, args.NonInteractive)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	profileInfo := ProfileInfo{}
	if args.Profile != "" {
		info, err := getProfileInfo(awsConfig, args.Profile)
		if err != nil {
			logLine(writer, fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		profileInfo = info
		if profileName == "" && info.AccountID == account.AccountID && info.RoleName == role.RoleName {
			profileName = args.Profile
		}
	}

	creds, err := getRoleCredentials(accessToken, session.Region, account.AccountID, role.RoleName)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	region := regionOverride
	if region == "" {
		region = profileInfo.Region
	}
	if region == "" {
		region = session.Region
	}

	if profileName == "" {
		profileName = buildProfileName(account, role)
	}
	if err := writeProfile(profileName, region, creds); err != nil {
		logLine(writer, fmt.Sprintf("Warning: failed to write AWS profile: %v", err))
	} else {
		logLine(writer, fmt.Sprintf("✓ Wrote AWS profile %s", profileName))
	}

	logLine(writer, fmt.Sprintf("✅ Ready for %s (%s) as %s", account.AccountName, account.AccountID, role.RoleName))

	envVars := map[string]string{
		"AWS_REGION":            region,
		"AWS_DEFAULT_REGION":    region,
		"AWS_ACCESS_KEY_ID":     creds.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY": creds.SecretAccessKey,
		"AWS_SESSION_TOKEN":     creds.SessionToken,
	}
	if creds.Expiration != nil {
		envVars["AWS_SESSION_EXPIRATION"] = creds.Expiration.Local().Format(time.RFC3339)
	}
	if profileName != "" {
		envVars["AWS_PROFILE"] = profileName
	}

	if !args.NoKube {
		maybeSwitchKubeAuto(account.AccountID, region, kubeContext, envVars, writer)
	}

	runIdentityCheck(envVars, writer)

	if args.PrintEnv {
		fmt.Print(formatExports(creds, region, profileName))
	}
}
