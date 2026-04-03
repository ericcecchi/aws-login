# Contributing to aws-login

Thank you for your interest in contributing! This document explains how to get set up, what to expect from the review process, and the conventions this project follows.

---

## Getting started

### Prerequisites

- [Go 1.21+](https://go.dev/dl/)
- [just](https://just.systems/) command runner
- AWS CLI v2

### Local setup

If you are an external contributor, **fork the repository first** on GitHub, then clone your fork:

```bash
# External contributors: fork on GitHub first, then clone your fork
git clone https://github.com/<your-username>/aws-login.git
cd aws-login
git remote add upstream https://github.com/ericcecchi/aws-login.git
just build        # build binary into bin/
just test         # run all tests
```

Maintainers with direct access can clone the repo directly:

```bash
git clone https://github.com/ericcecchi/aws-login.git
cd aws-login
just build
just test
```

---

## Workflow

### 1. Open or find an issue

Before writing code, check the [issue tracker](https://github.com/ericcecchi/aws-login/issues) to see if the bug or feature is already tracked. If not, open one first so the approach can be discussed before you invest time in implementation.

### 2. Create a feature branch

**Never commit directly to `main`.** Branch off of it instead.

**External contributors** (working from a fork):
```bash
# Keep your fork's main up to date with upstream first
git fetch upstream
git checkout main && git merge upstream/main
git checkout -b feat/my-feature   # or fix/my-bug
```

**Maintainers** (direct repo access):
```bash
git checkout main && git pull
git checkout -b feat/my-feature   # or fix/my-bug
```

### 3. Make your changes

- Keep changes focused. One logical change per pull request.
- Follow the [code conventions](#code-conventions) below.
- Add or update tests to cover the change (see [testing](#testing)).

### 4. Run tests

All tests must pass before opening a PR:

```bash
just test
```

### 5. Commit with Conventional Commits

This project uses [Conventional Commits](https://www.conventionalcommits.org/) to drive automated semantic versioning:

| Prefix | When to use | Version bump |
|--------|-------------|--------------|
| `feat:` | New behavior visible to users | minor |
| `fix:` | Bug fix | patch |
| `docs:` | Documentation only | none |
| `chore:` | Tooling, deps, config | none |
| `feat!:` / `fix!:` | Breaking change | major |

```bash
git commit -m "feat: add support for profile aliases"
```

Add a `BREAKING CHANGE:` footer or `!` suffix for changes that are not backwards-compatible.

### 6. Open a pull request

```bash
git push -u origin feat/my-feature
```

Then open a PR on GitHub targeting `ericcecchi/aws-login:main`. If you're working from a fork, GitHub will pre-fill the base repository — make sure it points to the upstream repo, not your fork. Fill out the pull request template — especially the **What**, **Why**, and **Considerations** sections. Link the issue the PR closes if one exists.

---

## Testing

Tests live next to the source files they cover (`foo_test.go` beside `foo.go`). They use stub `aws` and `kubectl` binaries injected into a temporary `PATH`, so no real AWS credentials are required.

- Use `setTempHome(t)` and `writeStubScripts(t)` in any test that exercises the full CLI flow.
- Prefer table-driven tests for functions with multiple input/output variations.
- Use `t.Fatalf` for hard failures; `t.Errorf` to accumulate multiple failures.
- See `testhelpers_test.go` and `AGENTS.md` for full details on the stub pattern and available test environment variables.

---

## Code conventions

- **One concern per file.** Add new behavior to the most relevant existing file; only create a new file if the concern is clearly distinct.
- **Dependency injection via `io.Writer`.** Pass a logger rather than writing to `os.Stderr` directly.
- **Never write config files directly.** All mutations to `~/.aws/config` go through `aws configure set`.
- **Return `(value, error)`.** Never panic in normal flow. Wrap errors with `fmt.Errorf("context: %w", err)`.
- **`defer` for cleanup.** Acquire a resource and defer its release on the same line.
- **Minimal dependencies.** Prefer the standard library. New direct dependencies need clear justification.
- **Naming:** `camelCase` for unexported, `PascalCase` for exported. Boolean helpers prefix with `is`/`has`/`can`. Functions use verb-object style: `listAccounts`, `updateKubeconfig`.

---

## Review process

- A maintainer will review your PR, leave feedback, and approve or request changes.
- Address review comments by pushing additional commits to the same branch.
- Once approved, the maintainer will merge. Do not squash or rebase after a PR is approved unless asked.
- Merging to `main` triggers an automatic release if commit messages warrant a version bump.

---

## Reporting bugs & requesting features

Use the issue templates:
- **Bug report** — for unexpected behavior or errors.
- **Feature request** — for new capabilities or improvements.
