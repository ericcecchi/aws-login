package main

import (
	"flag"
	"io"
	"os"
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
	versionFlag := fs.Bool("version", false, "Print version")
	versionShort := fs.Bool("v", false, "Print version")

	if err := fs.Parse(args); err != nil {
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
		Version:        *versionFlag || *versionShort,
	}, nil
}

func printUsage(w io.Writer) {
	_, _ = w.Write([]byte("Usage: aws-login [target] [role]\n"))
	_, _ = w.Write([]byte("       aws-login --account <id|name> --role <role>\n"))
	_, _ = w.Write([]byte("       aws-login --alias <name>\n"))
	_, _ = w.Write([]byte("       aws-login --print-env\n"))
	_, _ = w.Write([]byte("       aws-login --version\n"))
}

func logWriter(printEnv bool) io.Writer {
	if printEnv {
		return os.Stderr
	}
	return os.Stdout
}
