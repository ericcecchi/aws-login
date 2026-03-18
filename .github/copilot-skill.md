# aws-login CLI Skill

> A fast, zero-config AWS SSO login helper that discovers accounts and roles, configures AWS profiles, and auto-configures EKS Kubernetes contexts.

## When to Use

Use `aws-login` when you need to:

- Authenticate with AWS SSO and get short-lived credentials
- Switch between AWS accounts and roles
- Set up AWS profiles for CLI usage
- Configure Kubernetes contexts for EKS clusters
- Export `AWS_PROFILE` for tools that support named profiles

## Commands

### Login with account and role (recommended)

```bash
# Two positional args: <account> <role>
aws-login myaccount admin
aws-login 123456789012 ReadOnly

# Or with flags
aws-login --account myaccount --role admin
```

### Login with a named profile

```bash
aws-login --profile dev
```

### Interactive login (prompts for account and role)

```bash
aws-login
```

### Export credentials to current shell

```bash
eval "$(aws-login --print-env myaccount admin)"
```

### Export AWS_PROFILE to current shell

```bash
eval "$(aws-login --set-profile myaccount admin)"
```

### Health check and auto-repair

```bash
aws-login doctor
```

### Print version

```bash
aws-login --version
```

## All Flags

| Flag | Description |
|---|---|
| `--account <id\|name>` | Account name or ID (alternative to first positional arg) |
| `--role <name>` | Role name (alternative to second positional arg) |
| `--profile <name>` | Use a specific AWS profile name instead of auto-generated |
| `--sso-session <name>` | AWS SSO session name |
| `--region <region>` | AWS region override |
| `--kube-context <name>` | Explicit kubectl context to switch to |
| `--no-kube` | Skip Kubernetes context switching |
| `--non-interactive` | Fail instead of prompting (for CI/scripts) |
| `--print-env` | Print full credential export statements to stdout |
| `--set-profile` | Print `export AWS_PROFILE=<name>` to stdout |
| `--shell-init` | Print shell integration script for eval |
| `--doctor` | Validate and repair AWS/Kubernetes config files |
| `--version`, `-v` | Print version |

## Positional Arguments

```
aws-login <account> [role]
```

- **First positional arg**: Account name or ID (same as `--account`)
- **Second positional arg**: Role name (same as `--role`)
- Positional args cannot be combined with their flag equivalents

## Non-Interactive / Scripting Usage

For automation and scripts, use `--non-interactive` to prevent prompts:

```bash
aws-login --non-interactive --account 123456789012 --role admin
```

Or with positional args:

```bash
aws-login --non-interactive 123456789012 admin
```

## Profile Naming

When no `--profile` is specified, profiles are auto-named as:

```
<account-name>-<role>
```

Example: `prod-admin`

On each run, legacy profiles with the old `aws-login-<account-id>-<role>` format are automatically cleaned up.

## Shell Integration

Add to your shell rc file for automatic `AWS_PROFILE` setting:

```bash
eval "$(aws-login --shell-init)"
```

This wraps `aws-login` so that `AWS_PROFILE` is automatically exported to your shell after login.

## Kubernetes Integration

After login, `aws-login` automatically:

1. Lists EKS clusters for the selected account
2. Updates kubeconfig for each cluster
3. Switches to the first matching kube context

Skip with `--no-kube`.

## Requirements

- AWS CLI v2 (`aws`) with SSO support
- `kubectl` for Kubernetes context automation (optional)

## Examples for AI Agents

```bash
# Log in to a specific account and role
aws-login --non-interactive --account prod --role admin --no-kube

# Log in and set AWS_PROFILE in the environment
eval "$(aws-login --set-profile --non-interactive --account 123456789012 --role ReadOnly)"

# Log in and export full credentials
eval "$(aws-login --print-env --non-interactive --account staging --role developer)"

# Use a named profile
aws-login --non-interactive --profile my-dev-profile

# Check health
aws-login doctor
```
