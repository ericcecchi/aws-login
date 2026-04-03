# Agent Notes for aws-login

## Repo Overview

`aws-login` is a Go CLI that helps users log in with AWS SSO, writes short-lived credentials, and can configure EKS kube contexts. The binary wraps the AWS CLI — it never writes config files directly, instead delegating all mutations to `aws configure set` for consistency.

---

## Development Workflow

### Branch & Pull Request Policy

**Never commit or push directly to `main`.** All changes must go through a feature branch and an open pull request for review before merging.

```bash
# 1. Create a feature branch from main
git checkout main && git pull
git checkout -b feat/my-feature   # or fix/my-bug

# 2. Make changes, write tests, verify all tests pass
just test

# 3. Commit using Conventional Commits (see Release Workflow below)
git add .
git commit -m "feat: add support for profile aliases"

# 4. Push branch and open a pull request
git push -u origin feat/my-feature
# Then open a PR on GitHub targeting main
```

Pull requests require:

- All tests passing (`just test`)
- A descriptive title following Conventional Commits format
- Summary of what changed and why

---

## Build, Install & Test

```bash
just build    # Build binary into bin/aws-login
just install  # Install binary to GOBIN/GOPATH
just test     # Run all tests (go test ./...)
```

### Running Tests

**Tests must be written and run before every commit.** This repo avoids real AWS/kubectl calls in tests by injecting stub binaries into a temporary `PATH`.

```bash
just test
# or directly:
go test ./...
```

If tests fail, do not commit. Fix the failure or update tests to reflect intentional behavior changes.

---

## Testing Patterns

### Stub Binary Pattern

Tests in `testhelpers_test.go` inject fake `aws` and `kubectl` executables into a temp `PATH` so the full CLI flow runs without real AWS calls. Control test behavior via environment variables:

| Variable                            | Purpose                                                 |
| ----------------------------------- | ------------------------------------------------------- |
| `AWS_LOGIN_TEST_ACCOUNTS_JSON`      | JSON returned by `aws sso list-accounts`                |
| `AWS_LOGIN_TEST_ROLES_JSON`         | JSON returned by `aws sso list-account-roles`           |
| `AWS_LOGIN_TEST_CREDS_JSON`         | JSON returned by `aws sso get-role-credentials`         |
| `AWS_LOGIN_TEST_EKS_JSON`           | JSON returned by `aws eks list-clusters`                |
| `AWS_LOGIN_TEST_IDENTITY_JSON`      | JSON returned by `aws sts get-caller-identity`          |
| `AWS_LOGIN_TEST_CONFIGURE_SET_FILE` | File path where stub logs each `aws configure set` call |
| `AWS_LOGIN_TEST_SSO_CACHE_DIR`      | Directory where stub writes a fake SSO token cache file |

**Setup helpers:**

- `setTempHome(t)` — Sets `HOME` to a temp dir so `~/.aws` and `~/.aws-login` are isolated per test.
- `writeStubScripts(t)` — Creates stub `aws` and `kubectl` binaries in a temp dir and prepends it to `PATH`. Returns the temp dir.

**Always call `setTempHome` and `writeStubScripts` in any test that exercises real CLI flow.** This prevents test side effects on the developer's actual AWS config.

### Table-Driven Tests

Prefer table-driven tests for functions with multiple input/output variations. See `resolve_test.go` and `profile_test.go` for examples:

```go
tests := []struct {
    input string
    want  string
}{
    {"My Account", "my-account"},
    {"dev_admin",  "dev-admin"},
}
for _, tt := range tests {
    t.Run(tt.input, func(t *testing.T) {
        got := sanitizeProfilePart(tt.input)
        if got != tt.want {
            t.Fatalf("got %q, want %q", got, tt.want)
        }
    })
}
```

Use individual named test functions (not table-driven) when each test has significantly different setup, e.g., `cli_test.go`.

### Async / Background Behavior

For testing stale-while-revalidate caching (`cache.go`), poll with a deadline rather than `time.Sleep`:

```go
deadline := time.Now().Add(2 * time.Second)
for time.Now().Before(deadline) {
    if gotExpectedValue() {
        break
    }
    time.Sleep(25 * time.Millisecond)
}
```

### Assertions

Use `t.Fatalf` for hard failures that should stop a test immediately. Use `t.Errorf` when you want to accumulate multiple failures. This codebase favors `t.Fatalf` to keep failure messages focused.

---

## Code Organization

```
main.go                          # Thin entry point: calls awslogin.Run()
internal/awslogin/
  cli.go          # Flag parsing (flag.FlagSet), normalizeArgs(), printUsage()
  main.go         # Orchestrator: resolve → login → discover → configure → output
  types.go        # Data types: Args, SessionInfo, AccountInfo, RoleInfo, etc.
  constants.go    # Package-level constants (version, cache paths)
  aws.go          # AWS CLI wrappers: listAccounts, listRoles, getRoleCredentials
  resolve.go      # Fuzzy account/role matching with multi-level fallback
  profile.go      # Profile naming (sanitize) and aws configure set calls
  config.go       # AWS config file loading via gopkg.in/ini.v1
  cache.go        # Stale-while-revalidate caching with background goroutine refresh
  mutation.go     # Filesystem mutex + backup/restore for safe config mutations
  kube.go         # EKS cluster discovery and kubectl context switching
  util.go         # Shell integration (--install/--uninstall), helpers, formatting
  doctor.go       # Health check and repair (--doctor flag)
  ui.go           # Interactive fuzzy selection (go-fuzzyfinder, generics)
  *_test.go       # Tests co-located with their source files
  testhelpers_test.go  # Shared test helpers (stub binaries, temp HOME)
```

**File placement rule:** Each concern lives in exactly one file. Add new behavior to the most relevant existing file rather than creating new files unless the concern is clearly distinct.

---

## Architecture & Design Patterns

### Dependency Injection via `io.Writer`

All logging is passed as an `io.Writer` argument rather than written to `os.Stderr` directly. This keeps functions testable and allows the caller to redirect output.

```go
// Good
func listAccounts(log io.Writer, token string) ([]AccountInfo, error) { ... }

// Avoid
func listAccounts(token string) ([]AccountInfo, error) {
    fmt.Fprintln(os.Stderr, "listing accounts...") // hard to test
}
```

### Error Handling

- Return `(value, error)` pairs. Never panic in normal flow.
- Wrap errors with context using `fmt.Errorf("doing X: %w", err)` so callers can `errors.Is`/`errors.As` the chain.
- Extract stderr from failed `exec.Cmd` runs to produce meaningful error messages:

  ```go
  var exitErr *exec.ExitError
  if errors.As(err, &exitErr) {
      return fmt.Errorf("aws command failed: %s", strings.TrimSpace(string(exitErr.Stderr)))
  }
  ```

- Graceful degradation: subsystems like kube context switching log a warning and continue rather than failing the entire login flow.
- At the top-level `Run()`, log the error and call `os.Exit(1)` — do not propagate to the Go runtime.

### Never Write Config Files Directly

All AWS config mutations go through `aws configure set`. Never write `~/.aws/config` or `~/.kube/config` directly. This ensures the AWS CLI's own validation and formatting is applied.

### Stale-While-Revalidate Caching

`cache.go` returns cached data immediately and refreshes in a background goroutine. Cache files live in `~/.aws-login/cache/` named by SHA1 hash of the cache key. This pattern keeps the CLI fast for repeated use while staying up to date.

```go
// Pattern: return stale, refresh async
go func() {
    fresh, err := listAccounts(log, token)
    if err == nil {
        writeCache(key, fresh)
    }
}()
return cachedAccounts, nil
```

### Filesystem Mutex + Backup/Restore

`mutation.go` uses an exclusive lock file (`~/.aws-login/locks/mutation.lock`) to serialize concurrent invocations. Before any mutation:

1. Acquire the lock (45s timeout, stale lock detection at 2min).
2. Back up `~/.aws/config` and `~/.kube/config`.
3. Run the mutation.
4. Validate by running `aws configure list-profiles` and `kubectl config view`.
5. If validation fails, automatically restore from backup.
6. Release the lock via `defer`.

### Fuzzy Matching with Multi-Level Fallback

`resolve.go` matches user input to accounts/roles with progressive specificity:

1. Exact account ID match (digits only)
2. Exact display name match
3. Normalized name match (lowercase, hyphens)
4. Partial/substring match

If multiple candidates match at any level, fall through to interactive selection (unless `--non-interactive` is set, in which case return an error).

### Generic Interactive Selection

`ui.go` exposes a single generic function:

```go
func chooseInteractive[T any](items []T, label func(T) string) (T, error)
```

Use this whenever the user needs to pick from a list. It checks for TTY presence and returns an error if stdin is not a terminal (enabling `--non-interactive` to work cleanly).

### CLI Argument Normalization

`cli.go` uses a custom `normalizeArgs()` pre-pass before `flag.FlagSet.Parse()` to support interspersed positionals and flags (e.g., `aws-login --print-env myaccount admin`). Positional and flag forms of the same argument are mutually exclusive and produce a clear error.

---

## CLI Argument Design

- **Two positionals:** `aws-login <account> [role]`
  - First positional = account name or ID (same as `--account`)
  - Second positional = role name (same as `--role`)
  - Positional args conflict with their flag equivalents — error if both are used
- `--profile` — selects a named AWS profile (no positional equivalent)
- `--set-profile` — outputs `export AWS_PROFILE=<name>` to stdout for shell eval
- `--print-env` — outputs full credential exports to stdout
- `--non-interactive` — disables fuzzy selection; errors on ambiguous input
- `--no-kube` — skips kubectl context switching
- `--doctor` — health-check and repair mode
- `--install` — installs shell integration (one-time setup; modifies shell rc files)
- `--uninstall` — removes shell integration from shell rc files
- `--version` / `-v` — prints version

### Shell Integration

The recommended workflow is to run `aws-login --install` once, which:

1. Creates shell initialization scripts in `~/.aws-login/shell-init/`
2. Appends sourcing lines to shell rc files (`.bashrc`, `.zshrc`, `.zprofile`, `.config/fish/config.fish`)
3. Provides an `aws-login` wrapper function that automatically sets `AWS_PROFILE`

After installation, users simply run `aws-login account role` and the profile is set in their current shell. No eval or manual setup needed for subsequent commands.

---

## Go Conventions Used in This Repo

- **Go 1.21** — generics are available and used (see `ui.go`).
- **Naming:** unexported functions use `camelCase`; exported use `PascalCase`. Boolean helpers are prefixed `is`, `can`, `has` (e.g., `isTerminal`, `isStaleLock`). Functions follow verb-object naming: `listAccounts`, `updateKubeconfig`.
- **No global mutable state.** Constants and path vars are package-level but not mutated at runtime (except in tests via `setTempHome`).
- **Minimal dependencies.** Adding a new direct dependency requires clear justification. Use the standard library (`os/exec`, `encoding/json`, `flag`, `crypto/sha1`, etc.) wherever possible.
- **`defer` for cleanup.** Lock release, temp file cleanup, and similar teardown always use `defer` on the same line the resource is acquired.
- **Explicit over implicit.** No reflection, no magic registration. All behavior is traceable through direct function calls.
- **Test files co-located.** `foo_test.go` lives next to `foo.go`. Helpers shared across test files go in `testhelpers_test.go`.

---

## Release Workflow (Semantic Versioning)

Releases are created automatically on every push to `main`. Versioning is driven by **Conventional Commits** via Semantic Release.

| Commit format                           | Version bump |
| --------------------------------------- | ------------ |
| `feat: add support for profile aliases` | minor        |
| `fix: handle missing sso sessions`      | patch        |
| `feat!: drop legacy awscli v1 support`  | major        |
| `fix!: change default region behavior`  | major        |
| Body contains `BREAKING CHANGE: ...`    | major        |

Workflow file: `.github/workflows/release.yml`.

---

## Key Source Files

- `internal/awslogin/cli.go` — Argument parsing, usage text, normalizeArgs
- `internal/awslogin/main.go` — Main flow: resolve session → login → configure profile
- `internal/awslogin/types.go` — Args struct and all data types
- `internal/awslogin/resolve.go` — Account/role resolution with fuzzy matching
- `internal/awslogin/profile.go` — Profile naming and `aws configure set` calls
- `internal/awslogin/aws.go` — AWS CLI wrappers (list accounts, roles, credentials)
- `internal/awslogin/config.go` — AWS config file loading, SSO session discovery
- `internal/awslogin/kube.go` — EKS cluster discovery, kube context switching, and per-account context preference persistence (`~/.aws-login/kube-prefs.json`)
- `internal/awslogin/mutation.go` — Mutation lock and backup/restore for config files
- `internal/awslogin/cache.go` — Stale-while-revalidate account/role caching
- `internal/awslogin/util.go` — Shell init scripts, logging, formatting
- `internal/awslogin/doctor.go` — Config validation and repair
- `internal/awslogin/ui.go` — Generic interactive fuzzy selection

---

## AI Skill

The file `.agents/skills/aws-login-cli/SKILL.md` contains comprehensive CLI documentation designed for AI agent consumption.

**When to update `SKILL.md`:** Any change that affects how a user or agent would invoke the CLI or interpret its output requires a corresponding update to `SKILL.md`. This includes:

- New flags or positional arguments added or removed
- Changed flag names, default values, or behavior
- New or removed subcommands (e.g., `doctor`)
- Changes to output format (e.g., `--print-env`, `--set-profile`)
- Changes to profile naming conventions
- Changes to Kubernetes context switching behavior
- New error messages or exit codes that agents need to handle
- New onboarding or configuration flows

**How to update:** Edit `.agents/skills/aws-login-cli/SKILL.md` as part of the same PR that introduces the behavioral change. Treat `SKILL.md` as a user-facing document — write in clear, imperative language, and include updated command examples.
