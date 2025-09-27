Project development guidelines (advanced)

Context and goal
- The immediate goal is to convert the contents of ShellBasedPackageManager/flutter_packagemanager_setup from shell/PowerShell into Go while keeping equivalent behavior on Windows, macOS, and Linux. The current Go module is go 1.25 (module name: awesomeProject). The repository also contains PowerShell (.ps1) and Bash (.sh) scripts used to install/configure Flutter dependencies and private packages.
- Treat the shell scripts as the source of truth for behavior and supported scenarios while designing the Go-based replacement.

Build and configuration (project-specific)
- Go version: go 1.25 per go.mod. Ensure your local Go toolchain matches or is newer, but stay within compatibility (1.25+). If corporate environments pin versions, use GOTOOLCHAIN=local or asdf/Go tool to select 1.25.
- Module layout: top-level main.go is a simple executable. For the rewrite, prefer a multi-cmd layout:
  - cmd/fpm (or cmd/packagemanager): main package implementing CLI that replaces the scripts.
  - internal/sbpm: core logic (cross-platform) for discovering, installing, and configuring packages.
  - internal/sbpm/platform/{windows,linux,macos}: platform-specific pieces behind interfaces.
  - internal/sbpm/pm: adapters for package managers (apt, yum/dnf, pacman, brew, choco, winget, etc.).
- Build commands:
  - Build the current main: go build ./
  - Run main: go run ./
  - Build future CLI (example): go build -o bin/fpm ./cmd/fpm
- Windows/PowerShell notes for local dev:
  - Use PowerShell when invoking scripts in ShellBasedPackageManager. For Go builds/tests, PowerShell is fine; the go tool works the same.
  - Paths: prefer Go’s filepath package; never hardcode path separators.

Testing: configuration, running, and adding new tests
- Running tests:
  - All packages: go test ./...
  - Verbose and filtered: go test -v ./... -run TestName -count=1
  - Race detector (non-Windows or supported Windows): go test -race ./...
  - Short mode for integration-heavy code: go test -short ./...
- Temporary verified example (smoke test):
  - A minimal test like below compiles and runs in this repository (verified). Don’t commit it permanently—use it as a template:
    package main
    import "testing"
    func Test_Smoke(t *testing.T) { t.Log("tests are wired up") }
  - Command used to run it: go test ./...
- Adding new tests for the Go rewrite:
  - Unit tests: Place *_test.go next to implementation files.
    - Mock external effects (package managers, network, file system) via small interfaces. Provide fakes in tests.
    - Use context.Context with timeouts; in tests, set short timeouts and inject stub executors.
  - Integration tests: Gate with -short by default. Only run heavy or privileged operations when a specific env var is set, e.g. SBPM_E2E=1.
    - Example pattern:
      if testing.Short() || os.Getenv("SBPM_E2E")=="" { t.Skip("e2e disabled") }
  - Command execution harness:
    - Centralize external command execution (see below) and inject an Executor interface to allow deterministic tests.
  - Windows vs. Unix tests:
    - Use build tags or files with _windows.go, _unix.go, _linux.go, _darwin.go when behavior diverges. Provide stub tests behind the same tags.
  - JSON-driven behavior:
    - For items like scripts/windows/private-packages.json, define Go structs and validate load/parse logic with golden tests.

Create and run a simple test (demonstration)
- We created a temporary Test_Smoke to validate tooling and ran: go test ./... (it passed). The temporary file was removed to keep the repo clean. Use the snippet above to bootstrap new tests as needed.

Guidance for converting shell-based package manager to Go
- High-level architecture
  - CLI command (cmd/fpm):
    - Subcommands reflect script entry points: detect, plan, install, configure, doctor, list.
    - Flags mirror script options (non-interactive, dry-run, verbose, target platform override, etc.).
  - Core domain (internal/sbpm):
    - Define domain types: Package, Source (system pkg mgr, URL, archive), Action (Install, Upgrade, Remove, Configure), Plan (ordered actions), Result (success/failure, logs).
    - Provide a Planner that takes desired state (from config JSON) and produces a Plan based on current system state.
  - Platform services (internal/sbpm/platform/...):
    - OS detection, privilege checks, filesystem, environment variables, PATH editing (Windows registry vs. shells), proxy settings, certificates.
  - Package manager adapters (internal/sbpm/pm):
    - Interfaces: type Manager interface { Name() string; Present(ctx, pkg) (bool, error); Install(ctx, pkg) error; UpdateIndex(ctx) error; }
    - Implementations: APT, DNF/YUM, Pacman, Brew, Chocolatey, Winget. Keep commands data-driven via templates where viable.

- Replacing shell calls
  - Prefer native Go logic; only spawn external tools when necessary (the real package managers are external tools).
  - Use exec.CommandContext with Context timeouts and cancellation; capture stdout/stderr. Do not invoke via a shell layer (no "sh -c"/"powershell -Command") unless absolutely required for quoting semantics.
  - Normalize quoting: pass args as discrete elements. Avoid string-joined command lines.
  - Surface stderr/stdout to logs and to Result objects for troubleshooting.

- Privilege elevation and checks
  - Detection: On Unix, os.Geteuid()==0. On Windows, use golang.org/x/sys/windows to check admin token or attempt a privileged op and detect failure.
  - Elevation strategy: Document that the tool does not self-elevate; instruct users to run with sudo/Administrator. Optionally, add a helper that re-execs with elevation on supported platforms (defer implementation if not strictly needed).

- Configuration model
  - Parse the existing JSON (e.g., scripts/windows/private-packages.json) with a defined struct. Validate schema with jsonschema or custom validation.
  - Support per-platform conditionals in the config (include/exclude). Add an Evaluator that checks runtime OS/arch.
  - Keep configuration idempotent: running the tool twice should not break anything; it should converge to the same state.

- Interactive behavior and recommendations
  - The scripts include smart_recommendations.sh and multiselect.sh. For Go:
    - Provide a non-interactive default (CI-friendly) and an interactive TUI behind a flag using a lightweight library (optional) or simple stdin prompts.
    - Extract recommendation logic into pure functions so it’s testable.

- Logging, diagnostics, and dry runs
  - Provide leveled logs (info/debug/trace). For minimal dependencies, use log/slog.
  - Add --dry-run to print the Plan without executing actions.
  - Add --doctor to validate environment without changing state.
  - Persist an execution report (JSON) for debugging CI runs.

- Error handling and retries
  - Wrap external command errors with context (fmt.Errorf("apt install %s: %w", pkg, err)).
  - Add limited retries with backoff for transient network/package index errors.

- Cross-platform subtleties from current scripts
  - PATH management:
    - Windows: modify machine/user PATH via registry/APIs and broadcast WM_SETTINGCHANGE; avoid exceeding 2048-char limits in legacy contexts.
    - Unix: prefer /etc/paths.d or shell profile fragments in ~/.config for the user; avoid writing to arbitrary shell RC files; be explicit.
  - Certificates/proxy: honor standard env vars (HTTP_PROXY, HTTPS_PROXY, NO_PROXY). Consider reading OS-specific settings if required.
  - Archives: use Go’s archive/zip and compress/gzip + tar for archives handled by scripts.

- Mapping existing files to Go components
  - scripts/windows/windows_full_standalone.ps1 → cmd/fpm + internal/sbpm/platform/windows
  - scripts/linux-macos/linux_macos_full.sh → cmd/fpm + internal/sbpm/platform/{linux,macos}
  - scripts/shared/*.sh (utils, recommendations, multiselect) → internal/sbpm/{interactive,recommend}
  - install/install.{ps1,sh} and run.{ps1,sh} → replaced by a single cross-platform fpm executable.

Developer workflow (specific to this repo)
- Formatting and vetting:
  - go fmt ./...
  - go vet ./...
- Optional (if you add it): golangci-lint run
- Running the app during the transition:
  - Until the CLI exists, main.go is a placeholder. Use it to spike isolated pieces (but prefer adding code under internal/ and minimal main wrappers).
- Build tags and files:
  - Put platform-specific implementations in files named *_windows.go, *_linux.go, *_darwin.go with matching //go:build tags.

Testing strategy for the package-manager adapters
- Create an Executor interface:
  type Executor interface { Run(ctx context.Context, name string, args ...string) (stdout, stderr []byte, err error) }
  - Provide a DefaultExecutor using exec.CommandContext; provide a FakeExecutor for tests.
- State discovery should be separated from installation so tests can cover planning without performing changes.
- Use golden files for complex output (plans, reports) under testdata/.

How to run tests in CI and locally (project-tailored)
- Local: go test ./... -v -race (when supported).
- Windows PowerShell: go test ./...
- Use environment variables to enable integrations: SBPM_E2E=1 go test ./internal/sbpm/... -run E2E -v

Notes about deleting temporary files for this guidance
- Any example tests created purely to validate instructions should be removed after running them. The repository should only retain .junie/guidelines.md as part of this task.

Troubleshooting
- go: unknown revision errors: run go mod tidy (ensure you are online or behind a configured proxy).
- Permission denied during tests that touch the system: mark as integration and skip by default; require explicit opt-in.
- PowerShell vs Bash quoting differences: avoid shell wrapping; pass arguments directly with exec.CommandContext.

Next steps for the conversion
1) Establish internal/sbpm skeleton with interfaces and no-op adapters. Add unit tests for planning logic. 
2) Implement Windows adapter with winget/choco detection and basic install flow. 
3) Implement Linux adapter with apt/dnf/pacman detection. 
4) Implement macOS adapter with brew. 
5) Replace shell installers with the Go CLI and deprecate scripts.

**General Step By Step Guide**
Phase 0 — Decide the shape (non-negotiables)

Goals

One static Go binary (flutter-pm or mobilxpm) with a terminal-only UI (Bubble Tea).

Keep your one-line installers; they just download the binary to PATH and verify git & flutter|dart.
GitHub

Parity with current flows: local project discovery, “run from nested dir and detect nearest pubspec”, GitHub clone/fetch, multi-select repos, add as git deps (via dart|flutter pub add), backups, and a recommendation pass.
GitHub

Principles

Call the git CLI for fidelity (LFS, submodules, creds).

Use dart|flutter pub add (avoid YAML surgery).

Dry-run + JSON logs built-in.

No CGO; cross-compile easily.

Phase 1 — Map features to modules

From your README/features and scripts: install scripts, multiselect UI, smart recommendations, project discovery (local + GitHub fetch), backups, safety checks.
GitHub

Proposed layout

flutter-pm/
cmd/                     # cobra commands (optional; or single entry cmd)
root.go
tui.go                 # `flutter-pm` launches Bubble Tea by default
add.go                 # optional non-TUI subcommands
sync.go
status.go
internal/
core/                  # business logic (UI-independent)
env.go               # flags, json logs, dry run, concurrency
discovery.go         # nearest pubspec; scan common dirs
gh.go                # GitHub auth checks (via gh CLI) & clone targets
pub.go               # add/sync via dart|flutter
stale.go             # stale detection (lock age + ls-remote, etc.)
reco.go              # recommendations (your heuristics)
backup.go            # pubspec backup/restore
git.go               # thin git wrapper
tui/                   # Bubble Tea state machine
model.go
update.go
view.go
components/          # lists, forms, spinners
installers/              # keep your current install.{sh,ps1} updated
main.go
go.mod

Phase 2 — Data contracts (what the UI and core pass around)

Project{ path, pubspecPath }

RepoCandidate{ owner, name, url, privacy, desc } // from GitHub or direct URL entry

PkgSpec{ name, url, ref, subdir }

ActionResult{ ok, message, err, logs }

Reco{ message, severity, rationale }

Keep these UI-agnostic so TUI and CLI subcommands can reuse.

Phase 3 — Core logic (parity with scripts)
3.1 Discovery

Nearest pubspec: walk up from CWD until pubspec.yaml found (like your nested-dir detection).
GitHub

Local scan: search in ~/Development, ~/Projects, ~/dev (configurable).
GitHub

3.2 GitHub integration

Detect gh and auth (gh auth status). If absent, show actionable hint (brew/apt/winget) but allow manual URL input.
GitHub

List user/org repos (paged) or accept pasted URLs.

Clone target repos to a chosen location (or temp) if you support “GitHub Fetch”.

3.3 Selection UX → PkgSpec

Multi-select repos (Bubble Tea list + checkboxes), then a small form per selection: package name (default repo name), ref (default main), subdir optional. That mirrors your current flow.
GitHub

3.4 Add deps safely

Prefer dart pub add NAME --git-url URL [--git-ref ref --git-path subdir] (fallback to flutter pub add if Dart missing).

Before modify: backup pubspec (e.g., .pubspec.backup.YYYYMMDDHHmm), like your script.
GitHub

After batch add: pub get (via flutter pub get or dart pub get).

3.5 Stale detection

Quick heuristic: if git deps exist and pubspec.lock age > 24h, mark “stale candidates”.
GitHub

Precise: for each git dep, run git ls-remote URL ref and compare the commit to what’s in pubspec.lock (the lock records resolved sha when git deps are used). Flag mismatches as stale.

3.6 Recommendations

Seed with rules you already describe:

Suggest pinning floating branches to a tag/sha.

Suggest sparse/shallow clone for heavy repos.

Suggest using mirrors/shared cache if multiple projects pull the same repo.

Suggest pub upgrade --major-versions preview where safe. (You already hint at quality-first recommendations; you can bring those forward here.)
GitHub

Phase 4 — Bubble Tea TUI (terminal-only)

Model (top-level):

type model struct {
step Step
msgs []string
// discovery
projects []Project
selectedProject int
// source selection
source SourceMode // LocalScan | GitHub | ManualURL
repos   []RepoCandidate
picks   map[int]bool
// per-pick package spec editing
editIdx int
edits   []PkgSpec
// run queue + progress
jobs    []PkgSpec
results []ActionResult
// recomms
recos   []Reco
err     error
}


Update loop:

Steps: DetectProject → ChooseSource → ListRepos → EditSpecs → Confirm → Execute → Summary/Recos.

Use tea.Cmd to run long ops: discovery scan, GH fetch, pub add, pub get, git ls-remote.

Stream output lines into msgs pane; show per-job progress.

View:

Left: main list/form pane (projects/repos/specs).

Right (optional): log/output pane (scrollable).

Footer: keybinds (↑/↓, space, enter, q, tab to move focus).

Keep simple ANSI, no mouse, no real GUI.

Components:

List (repos/projects) with checkbox support.

Form (name/ref/subdir).

Spinner/Progress lines while executing.

Phase 5 — CLI surface (optional but useful)

Alongside the TUI default entry, expose parity commands for CI/automation:

flutter-pm          # launches TUI
flutter-pm add --name NAME --git URL [--ref ... --path ...] --root .
flutter-pm sync --root .
flutter-pm status --json
flutter-pm reco --json


This mirrors your current script affordances and lets power users bypass TUI.

Phase 6 — Installers & releases

Keep your install/run scripts (Bash/PS) but make them fetch the latest binary from GitHub Releases and put it in PATH (your README already advertises this pattern).
GitHub

Use goreleaser to produce:

flutter-pm_{darwin,linux,windows}_{amd64,arm64}.{tar.gz|zip}

SHA256SUMS.txt

Add an in-app self-update later (nice-to-have).

Phase 7 — Cross-platform hardening

Paths via filepath only; no hardcoded separators.

Windows: avoid rename-on-open; do atomic write+replace patterns for backups.

Detect git, dart or flutter, gh and render exact install hints per OS (brew/apt/winget), like your README’s prerequisite section.
GitHub

Phase 8 — Testing strategy

Golden parity tests

For a small matrix of scenarios (git dep add, stale detection, recommendations), run:

current Bash/PS in a temp sandbox repo (create a minimal Flutter project scaffolding),

the new Go binary against the same inputs,

compare structured logs (--json) and final pubspec.yaml/lockfile.

Unit tests

discovery_test.go (nearest pubspec, scanning roots)

stale_test.go (fake ls-remote responses)

pub_test.go (spawn a fake dart that records args)

Integration

Spin temporary git remotes (with git init --bare) to validate ls-remote, add, and pub get paths offline.

Phase 9 — Incremental migration plan

Scaffold core (env, git, pub, discovery) + a minimal TUI that just detects project and runs pub get.

Port Add flow (multi-select → edit specs → pub add).

Port Stale detection (heuristic → precise ls-remote compare).

Port Recommendations rules you already have.

Wire installers to download the binary; mark scripts “bootstrap only.”

Add status/add/sync CLI subcommands for CI.

Release alpha → dogfood on your machines → cut 1.0 when parity holds.

Phase 10 — Nice extras (after parity)

--explain (print exact git/pub commands).

--jobs N worker pool for batch operations.

--json logs for CI, --quiet for minimal.

Cache/mirror strategy if you want to accelerate multi-project pulls.

Minimal starting code (drop-in snippets)

pub.go (core call; mirrors your current approach):

tool := findFirstOnPath("dart", "flutter")
if tool == "" { return Err("neither 'dart' nor 'flutter' found") }
args := []string{"pub","add", name, "--git-url", url}
if ref != "" { args = append(args, "--git-ref", ref) }
if sub != "" { args = append(args, "--git-path", sub) }
runCmd(tool, args, withDir(root))


stale.go (precise check):

// parsed from lock: deps[name].source == "git" -> deps[name].resolved.ref/sha
upstream := gitLsRemote(url, ref)   // returns sha
if upstream != lockSha { markStale(name, upstream, lockSha) }


Bubble Tea model step switch:

switch m.step {
case DetectProject: // run discovery
case ChooseSource:
case ListRepos:
case EditSpecs:
case Execute: // enqueue tea.Cmd for each PkgSpec
case Summary:
}

