# Agent Notes for aws-login

## Repo Overview
`aws-login` is a Go CLI that helps users log in with AWS SSO, writes short-lived credentials, and can configure EKS kube contexts.

## Build and Run
- Build: `go build .`
- Build binary into `bin/`: `make build`
- Install to GOBIN/GOPATH: `make install`

## Tests
There are no automated tests in this repo at the moment.

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
