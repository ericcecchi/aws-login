---
name: aws-login-cli
description: Fast, zero-config AWS SSO login helper that discovers accounts and roles, configures AWS profiles, and auto-configures EKS Kubernetes contexts. Use when authenticating with AWS SSO, switching between AWS accounts or roles, setting up AWS profiles for CLI usage, configuring Kubernetes contexts for EKS clusters, or exporting AWS_PROFILE for tools that support named profiles.
---

# aws-login CLI

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
| `--kube-context <name>` | Switch to this context and save as account default |
| `--no-kube` | Skip Kubernetes context switching |
| `--non-interactive` | Fail instead of prompting (for CI/scripts) |
| `--print-env` | Print full credential export statements to stdout |
| `--set-profile` | Print `export AWS_PROFILE=<name>` to stdout |
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

## Shell Integration

Run once to install the shell wrapper:

```bash
aws-login --install
```

This updates your shell rc files (`.bashrc`, `.zshrc`, `.zprofile`, `config.fish`) to automatically load an `aws-login` wrapper that sets `AWS_PROFILE` in your current shell after login. No `eval` needed.

To remove:

```bash
aws-login --uninstall
```

## Kubernetes Integration

After login, `aws-login` automatically:

1. Lists EKS clusters for the selected account
2. Updates kubeconfig for each cluster
3. Filters contexts to those matching the account (by account ID or cluster name)
4. **If one context matches** — switches to it automatically
5. **If multiple contexts match**:
   - Checks `~/.aws-login/kube-prefs.json` for a saved preference for this account
   - If a preference exists and is still valid, uses it (logs "Using saved Kubernetes context: …")
   - If no preference (or it is stale), prompts interactively via fuzzy finder and saves the choice for next time
   - In `--non-interactive` mode, skips the switch with a warning instead of prompting
6. **`--kube-context <name>`** — bypasses discovery, switches directly, and saves as preference

Skip entirely with `--no-kube`. Force a one-time context with `--kube-context`.

Preferences are stored per account ID in `~/.aws-login/kube-prefs.json`.

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
