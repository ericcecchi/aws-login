# aws-login

`aws-login` is a fast, zero-config AWS SSO helper that discovers every account and role you can access, then configures AWS profiles for SSO use. It also auto-configures Kubernetes contexts for EKS so you can start working immediately.

## Highlights

- Interactive fuzzy selection for accounts and roles.
- Automatic onboarding if you have never run `aws configure sso`.
- Configures AWS profiles through `aws configure set` (no direct config-file writes).
- Auto‑discovers EKS clusters and switches kube context.
- Caches account and role lookups in `~/.aws-login/cache` using stale-while-revalidate.
- Uses a cross-terminal mutation lock to prevent concurrent config writes.
- Backs up and auto-recovers AWS/Kubernetes config files if corruption is detected.
- No extra system dependencies.

## Requirements

- AWS CLI v2 (`aws`) with SSO support.
- `kubectl` for Kubernetes context automation.
- Go 1.21+ for building from source.

## Install

From GitHub:

```bash
go install github.com/ericcecchi/aws-login@latest
```

From a local clone:

```bash
git clone https://github.com/ericcecchi/aws-login.git
cd aws-login
go install .
```

The binary is installed into `GOBIN` or `GOPATH/bin`.

## Quickstart

Interactive login with fuzzy selection:

```bash
aws-login
```

Pick a specific account and role:

```bash
aws-login myaccount admin
aws-login 123456789012 ReadOnly
```

Or use flags:

```bash
aws-login --account 123456789012 --role admin
```

Use a named profile:

```bash
aws-login --profile dev
```

Export `AWS_PROFILE` for the current shell:

```bash
eval "$(aws-login --set-profile myaccount admin)"
```

Print full credential exports for the current shell:

```bash
eval "$(aws-login --print-env)"
```

Run health check and auto-repair for config corruption:

```bash
aws-login doctor
```

## Shell Integration

To use `aws-login` as a convenient shell wrapper that automatically sets `AWS_PROFILE`, install the shell integration once:

```bash
aws-login --install
```

This updates your shell configuration files (`.bashrc`, `.zshrc`, `.zprofile`, etc.) to automatically load the `aws-login` wrapper function when you start a new shell. After installation, simply use:

```bash
aws-login myaccount admin
```

The `AWS_PROFILE` environment variable will be automatically set in your current shell session. No `eval` needed.

> **To activate immediately** without restarting your shell, source the init file once:
> ```bash
> source ~/.aws-login/shell-init/init.zsh   # zsh
> source ~/.aws-login/shell-init/init.sh    # bash
> source ~/.aws-login/shell-init/init.fish  # fish
> ```
> The `--install` output also prints the exact command for your current shell.

To uninstall, run:

```bash
aws-login --uninstall
```

**Note:** For full credential exports (access key, secret, token), use `--print-env`:

```bash
eval "$(aws-login --print-env myaccount admin)"
```

Or use `--set-profile` to export only the profile name:

```bash
eval "$(aws-login --set-profile myaccount admin)"
```

## Configuration

`aws-login` wraps the existing AWS CLI configuration:

- `~/.aws/config` for SSO session/profile metadata
- AWS CLI credential resolution and caches for temporary credentials

Example profile in `~/.aws/config`:

```ini
[profile dev]
sso_session = my-sso
sso_account_id = 123456789012
sso_role_name = admin
region = us-east-1
output = json
```

## Profiles

Each login writes/updates an AWS profile named:

`<account-name>-<role>`

For example, an account named `prod` with role `admin` produces a profile named `prod-admin`.

Profile configuration is written via AWS CLI commands, and credentials are resolved by AWS CLI at runtime.

You can then use:

```bash
aws s3 ls --profile prod-admin
```

## Kubernetes

On successful login, the tool:

- Lists EKS clusters for the selected account.
- Runs `aws eks update-kubeconfig` for each cluster.
- Lists matching kube contexts and switches to the first match.

If you want to skip Kubernetes setup:

```bash
aws-login --no-kube
```

## Onboarding Flow

If `~/.aws/config` is missing or no SSO sessions are configured, the tool will:

- Create the AWS config file if needed.
- Launch `aws configure sso`.
- Continue login once the SSO session exists.

## Troubleshooting

- **No TTY available**: Use `--account` and `--role` or `--non-interactive`.
- **No SSO sessions found**: Run `aws configure sso` and try again.
- **No EKS clusters**: The tool will skip kube context switching.
- **Concurrent runs blocked**: Wait for the other `aws-login` process to finish; config updates are serialized to avoid corruption.
- **Recover from corruption**: Run `aws-login doctor` to validate and restore configs from backups in `~/.aws-login/backups`.

## Agent Skill

An AI agent skill for `aws-login` is available in the [open agent skills ecosystem](https://skills.sh/). Install it with the skills CLI so your AI assistant knows how to use `aws-login` to authenticate, switch accounts, and configure profiles:

```bash
npx skills add ericcecchi/aws-login
```

Once installed, agents will automatically invoke `aws-login` when AWS authentication is needed.

## Development

```bash
# Build the binary into bin/aws-login
just build
# Run tests
just test
# Install locally from source
just install
```

## Conventional Commits

Releases are automated with Semantic Release and require Conventional Commit messages.
Use these formats so versions are bumped correctly:

- `feat: add support for profile aliases` (minor)
- `fix: handle missing sso sessions` (patch)
- `feat!: drop legacy awscli v1 support` (major)
- `fix!: change default region behavior` (major)

For breaking changes, add `!` after the type or include a footer like:

```
BREAKING CHANGE: remove deprecated flags
```

## License

MIT. See `License`.
