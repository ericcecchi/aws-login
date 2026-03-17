# Agent Notes for aws-login

## Repo Overview
`aws-login` is a Go CLI that helps users log in with AWS SSO, writes short-lived credentials, and can configure EKS kube contexts.

## Build and Run
- Build: `go build .`
- Build binary into `bin/`: `make build`
- Install to GOBIN/GOPATH: `make install`

## Tests
Run: `go test ./...`
Tests use stubbed `aws` and `kubectl` binaries in temp PATHs to avoid real AWS calls.

## CLI Argument Design
- Two positional args: `aws-login <account> [role]`
  - First positional = account name or ID (same as `--account`)
  - Second positional = role name (same as `--role`)
  - Positional args conflict with their flag equivalents (error if both used)
- `--profile` flag selects a named AWS profile (no positional equivalent)
- `--set-profile` outputs `export AWS_PROFILE=<name>` to stdout for shell eval
- `--print-env` outputs full credential exports to stdout
- Shell init wrapper (`--shell-init`) uses `--set-profile` by default

## Key Source Files
- `cmd/aws-login/cli.go` — Argument parsing, usage text, normalizeArgs
- `cmd/aws-login/main.go` — Main flow: resolve session → login → configure profile
- `cmd/aws-login/types.go` — Args struct, data types
- `cmd/aws-login/resolve.go` — Account/role resolution (fuzzy matching)
- `cmd/aws-login/profile.go` — Profile naming and `aws configure set` calls
- `cmd/aws-login/aws.go` — AWS CLI wrappers (list accounts, roles, credentials)
- `cmd/aws-login/config.go` — AWS config file loading, SSO session discovery
- `cmd/aws-login/kube.go` — EKS cluster discovery and kube context switching
- `cmd/aws-login/mutation.go` — Mutation lock and backup/restore for config files
- `cmd/aws-login/util.go` — Shell init scripts, logging, formatting

## Release Workflow (Semantic Versioning)
- Releases are created automatically on every push to `main`.
- Versioning is driven by Conventional Commits via Semantic Release.
- Use commit messages like:
  - `feat: add support for profile aliases` (minor)
  - `fix: handle missing sso sessions` (patch)
  - `feat!: drop legacy awscli v1 support` (major)
  - `fix!: change default region behavior` (major)
  - `BREAKING CHANGE: remove deprecated flags` (major)

Workflow file: `.github/workflows/release.yml`.

## AI Skill
The file `.github/copilot-skill.md` contains comprehensive CLI documentation
designed for AI agent consumption. Keep it updated when CLI behavior changes.
