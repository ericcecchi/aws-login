# aws-login

`aws-login` is a fast, zero-config AWS SSO helper that discovers every account and role you can access, then exports short‑lived credentials for your current shell. It also auto‑configures Kubernetes contexts for EKS so you can start working immediately.

## Highlights

- Interactive fuzzy selection for accounts and roles.
- Automatic onboarding if you have never run `aws configure sso`.
- Writes a short‑lived AWS profile per account/role for easy `--profile` use.
- Auto‑discovers EKS clusters and switches kube context.
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

If the repo is private, set:

```bash
export GOPRIVATE=github.com/ericcecchi/*
```

From a local clone:

```bash
cd ~/Projects/aws-login
go install .
```

The binary is installed into `GOBIN` or `GOPATH/bin`.

## Quickstart

Interactive login with fuzzy selection:

```bash
aws-login
```

Pick a specific account/role:

```bash
aws-login --account 123456789012 --role admin
```

Use a named alias from `~/.aws-login.toml`:

```bash
aws-login dev
aws-login dev developer
```

Print exports for the current shell:

```bash
eval "$(aws-login --print-env)"
```

## Shell Integration

To make environment updates automatic, add the shell wrapper once:

```bash
eval "$(aws-login --shell-init)"
```

This defines an `aws-login` shell function that runs the binary and applies the exported AWS credentials to your current shell session.
For non-bash shells, set `AWS_LOGIN_SHELL` to `fish` before running the command.

## Configuration

`aws-login` reads optional config from `~/.aws-login.toml` (or `AWS_LOGIN_CONFIG`).
If the file does not exist, a minimal config is created automatically.

Start with `config_sample.toml` to define aliases and defaults.

Example:

```toml
[defaults]
sso_session = "my-sso"

[aliases.dev]
account_id = "123456789012"
default_role = "admin"
roles = ["admin", "read"]
region = "us-east-1"
```

## Profiles

Each login writes a short‑lived AWS profile named:

`aws-login-<account>-<role>`

These are saved into:

- `~/.aws/config`
- `~/.aws/credentials`

You can then use:

```bash
aws s3 ls --profile aws-login-myaccount-admin
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

## Development

```bash
go build .
go test ./...
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
