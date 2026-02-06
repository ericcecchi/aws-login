# aws-login

AWS SSO helper to discover accessible accounts and roles, then export credentials for the current shell.

## Usage

```bash
aws-login                 # interactive account + role selection
aws-login dev             # use alias from ~/.aws-login.toml
aws-login dev developer   # alias + role override
aws-login --account 123456789012 --role admin
aws-login --sso-session perch
```

## Install

From this repo:

```bash
cd ~/Projects/aws-login
go install ./cmd/aws-login
```

The binary will be installed into `GOBIN` or `GOPATH/bin`.
You can also run `make install`.

From GitHub:

```bash
go install github.com/ericcecchi/aws-login/cmd/aws-login@latest
```

If the repo is private, set `GOPRIVATE=github.com/ericcecchi/*` and ensure your
GitHub auth is configured.

## Configuration

The CLI reads optional config from `~/.aws-login.toml` (or `AWS_LOGIN_CONFIG`).
If it doesn't exist yet, the CLI will create a minimal config automatically.
Copy `config_sample.toml` from this package as a starting point to define
aliases, default role mappings, and regions. Kube context is now automatic.
Each login also writes a short-lived AWS profile named
`aws-login-<account>-<role>` into `~/.aws/config` and `~/.aws/credentials`.

Selections use fuzzy search via a built-in terminal picker (no extra system packages).

## Kubernetes

When you log in, the tool will:
- Discover EKS clusters in the selected account.
- Update your kubeconfig for those clusters.
- List contexts that match the account and switch to the first one found.
