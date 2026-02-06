package main

import (
	"flag"
	"io"
	"os"
	"strings"
)

func parseArgs(args []string) (Args, error) {
	fs := flag.NewFlagSet("aws-login", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	accountFlag := fs.String("account", "", "Account name or ID")
	roleFlag := fs.String("role", "", "Role name")
	aliasFlag := fs.String("alias", "", "Alias defined in config")
	profileFlag := fs.String("profile", "", "AWS profile to use")
	ssoSessionFlag := fs.String("sso-session", "", "AWS SSO session name")
	regionFlag := fs.String("region", "", "AWS region to export")
	kubeContextFlag := fs.String("kube-context", "", "kubectl context to switch to")
	noKube := fs.Bool("no-kube", false, "Skip kubectl context switching")
	nonInteractive := fs.Bool("non-interactive", false, "Fail instead of prompting")
	printEnv := fs.Bool("print-env", false, "Print export statements to stdout")
	shellInit := fs.Bool("shell-init", false, "Print shell integration script")
	versionFlag := fs.Bool("version", false, "Print version")
	versionShort := fs.Bool("v", false, "Print version")

	if err := fs.Parse(normalizeArgs(args)); err != nil {
		return Args{}, err
	}

	positional := fs.Args()
	var target, roleArg string
	if len(positional) > 0 {
		target = positional[0]
	}
	if len(positional) > 1 {
		roleArg = positional[1]
	}

	role := *roleFlag
	if role == "" {
		role = roleArg
	}

	return Args{
		Target:         target,
		Role:           role,
		Account:        *accountFlag,
		RoleFlag:       *roleFlag,
		Alias:          *aliasFlag,
		Profile:        *profileFlag,
		SSOSession:     *ssoSessionFlag,
		Region:         *regionFlag,
		KubeContext:    *kubeContextFlag,
		NoKube:         *noKube,
		NonInteractive: *nonInteractive,
		PrintEnv:       *printEnv,
		ShellInit:      *shellInit,
		Version:        *versionFlag || *versionShort,
	}, nil
}

func printUsage(w io.Writer) {
	_, _ = w.Write([]byte("Usage: aws-login [target] [role]\n"))
	_, _ = w.Write([]byte("       aws-login --account <id|name> --role <role>\n"))
	_, _ = w.Write([]byte("       aws-login --alias <name>\n"))
	_, _ = w.Write([]byte("       aws-login --print-env\n"))
	_, _ = w.Write([]byte("       aws-login --shell-init\n"))
	_, _ = w.Write([]byte("       aws-login --version\n"))
}

func logWriter(printEnv bool) io.Writer {
	if printEnv {
		return os.Stderr
	}
	return os.Stdout
}

func normalizeArgs(args []string) []string {
	flagWithValue := map[string]struct{}{
		"--account":      {},
		"--role":         {},
		"--alias":        {},
		"--profile":      {},
		"--sso-session":  {},
		"--region":       {},
		"--kube-context": {},
	}
	flagBool := map[string]struct{}{
		"--no-kube":         {},
		"--non-interactive": {},
		"--print-env":       {},
		"--shell-init":      {},
		"--version":         {},
	}

	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positionals = append(positionals, args[i+1:]...)
			break
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			name := arg
			if strings.HasPrefix(arg, "--") {
				if idx := strings.Index(arg, "="); idx != -1 {
					name = arg[:idx]
				}
			}
			if name == "-v" {
				flags = append(flags, arg)
				continue
			}
			if _, ok := flagWithValue[name]; ok {
				flags = append(flags, arg)
				if !strings.Contains(arg, "=") && i+1 < len(args) {
					flags = append(flags, args[i+1])
					i++
				}
				continue
			}
			if _, ok := flagBool[name]; ok {
				flags = append(flags, arg)
				continue
			}
		}
		positionals = append(positionals, arg)
	}

	return append(flags, positionals...)
}
