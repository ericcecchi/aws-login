package awslogin

import (
	"fmt"
	"os"
	"strings"
)

func Run() {
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

	if args.Doctor {
		doctorWriter := logWriter(args)
		if err := runDoctor(doctorWriter); err != nil {
			logLine(os.Stderr, fmt.Sprintf("Doctor failed: %v", err))
			os.Exit(1)
		}
		return
	}

	writer := logWriter(args)

	if !commandExists("aws") {
		logLine(writer, "Error: AWS CLI is not installed or not in PATH")
		os.Exit(1)
	}

	awsConfig, err := loadAWSConfig()
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	selectedAccount := strings.TrimSpace(args.Account)
	selectedRole := strings.TrimSpace(args.Role)
	kubeContext := strings.TrimSpace(args.KubeContext)
	regionOverride := strings.TrimSpace(args.Region)
	profileName := strings.TrimSpace(args.Profile)

	profileInfo := ProfileInfo{}
	profileFound := false
	if profileName != "" {
		info, found, err := getProfileInfoIfExists(awsConfig, profileName)
		if err != nil {
			logLine(writer, fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		profileInfo = info
		profileFound = found
		if !found {
			logLine(writer, fmt.Sprintf("ℹ️  Profile '%s' not found; will create it", profileName))
		}
	}

	if profileFound {
		if selectedAccount == "" && profileInfo.AccountID != "" {
			selectedAccount = profileInfo.AccountID
		}
		if selectedRole == "" && profileInfo.RoleName != "" {
			selectedRole = profileInfo.RoleName
		}
	}

	if args.NonInteractive && (selectedAccount == "" || selectedRole == "") {
		logLine(writer, "Error: missing account or role in non-interactive mode")
		os.Exit(1)
	}

	session, err := resolveSession(awsConfig, writer, args.SSOSession, profileName, args.NonInteractive)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
	session, err = ensureReusableSSOSession(awsConfig, session)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	accessToken, err := ensureLoggedIn(session, writer)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	cacheKey := sessionCacheKey(session)
	accounts, err := listAccountsCached(accessToken, session.Region, cacheKey)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	account, err := resolveAccount(accounts, selectedAccount, writer, args.NonInteractive)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	roles, err := listRolesCached(accessToken, session.Region, account.AccountID, cacheKey)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	role, err := resolveRole(roles, selectedRole, writer, args.NonInteractive)
	if err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}

	if profileName == "" {
		profileName = buildProfileName(account, role)
	}

	// For auto-named profiles, load existing profile info so we can preserve
	// any previously stored region rather than defaulting to the SSO session region.
	if !profileFound {
		if info, found, err := getProfileInfoIfExists(awsConfig, profileName); err == nil && found {
			profileInfo = info
		}
	}

	region := regionOverride
	if region == "" {
		region = profileInfo.Region
	}
	if region == "" {
		region = session.Region
	}

	if err := withMutationGuard(!args.NoKube, func() error {
		if err := configureProfile(profileName, region, session, account.AccountID, role.RoleName); err != nil {
			return err
		}
		if !args.NoKube {
			maybeSwitchKubeAuto(account.AccountID, region, kubeContext, profileName, profileInfo.EKSRoleARN, args.NonInteractive, writer)
		}
		return nil
	}); err != nil {
		logLine(writer, fmt.Sprintf("Error: %v", err))
		os.Exit(1)
	}
	logLine(writer, fmt.Sprintf("✓ Configured AWS profile %s", profileName))
	logLine(writer, fmt.Sprintf("✅ Ready for %s (%s) as %s", account.AccountName, account.AccountID, role.RoleName))

	runIdentityCheck(profileName, region, writer)

	if args.PrintEnv {
		creds, err := getRoleCredentials(accessToken, session.Region, account.AccountID, role.RoleName)
		if err != nil {
			logLine(writer, fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		fmt.Print(formatExports(creds, region, profileName))
	} else if args.SetProfile {
		fmt.Printf("export AWS_PROFILE=%s\n", shellQuote(profileName))
	}
}

