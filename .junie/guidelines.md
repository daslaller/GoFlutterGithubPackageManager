**Main Goal**
This is a project to convert the exiting shell/ps based package manager to a Go binary for the commandline which uses bubbletea for the UI.
This package manager is for both Windows, Linux and MacOS.
Therefore, we maintain an entrypoint that works for both Linux/Macos and one in PowerShell for Windows.
The entrypoint in shell is used soley as a oneline installer which adds the binary to the PATH.
After path install it will be invoked as flutter-pm.
Take care to audit and have feature parity with the linux/macos version, its the source of truth.

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




---

Build/Configuration Instructions (Optimized)

- Go toolchain: Tested with go1.25.1 (windows/amd64). Use Go >= 1.23 for cross-platform, 1.25 preferred.
- No CGO: Ensure static builds and easy cross-compile.
  - PowerShell: setx CGO_ENABLED 0 (new shells) or $Env:CGO_ENABLED = "0" (current shell)
  - Bash: export CGO_ENABLED=0
- Reproducible builds:
  - Use -trimpath and -ldflags "-s -w" to strip symbols.
  - Recommended default build flags:
    - PowerShell: go build -trimpath -ldflags "-s -w" -mod=readonly ./...
    - Bash: GOFLAGS="-trimpath -mod=readonly" go build -ldflags="-s -w" ./...
- Target binary name: flutter-pm (mobilxpm alias optional). The shell/PS one-line installers should download the binary into PATH and then invoke flutter-pm.
- Cross-compilation matrix (no cgo):
  - Windows:   GOOS=windows GOARCH=amd64|arm64 → flutter-pm.exe
  - macOS:     GOOS=darwin  GOARCH=amd64|arm64 → flutter-pm
  - Linux:     GOOS=linux   GOARCH=amd64|arm64 → flutter-pm
  - Example (PowerShell):
    - $Env:GOOS = "linux"; $Env:GOARCH = "amd64"; go build -trimpath -ldflags "-s -w" -o dist/flutter-pm-linux-amd64 .
    - Remove-Item Env:GOOS, Env:GOARCH
- Release artifacts (recommended via goreleaser):
  - flutter-pm_{darwin,linux,windows}_{amd64,arm64}.{tar.gz|zip}
  - SHA256SUMS.txt
- External CLIs the app will call at runtime (must be discoverable on PATH): git, dart and/or flutter, gh (optional), unzip/zip (optional). The shell/macos scripts are the source of truth for expected behavior.


Testing Information

General guidance
- Keep close parity with the Linux/macOS shell implementation. Validate new Go behavior by comparing against ShellBasedPackageManager/scripts/linux-macos/linux_macos_full.sh in a sandboxed project.
- Prefer "go test" with table-driven unit tests for core logic. Integration tests may spin up temp git repos (git init --bare) and fake dart executables by PATH shim to record arguments.
- Use Windows-safe paths (filepath package) and avoid rename-on-open patterns when writing files.

Where to put tests
- Place unit tests adjacent to the package they test: package foo → foo_test.go.
- For integration tests that touch the file system or spawn processes, keep them under internal/<module> to avoid exporting test helpers.

How to run tests
- All packages: go test ./...
- With verbose output: go test -v ./...
- Single package: go test ./internal/core
- Race detector (when applicable and not using CGO): go test -race ./...

Adding a new test (verified example)
- We validated this flow locally before writing these docs by temporarily creating a tiny package and test, running go test, and then removing the files. You can follow the same steps to bootstrap tests in a new package:
  1) Create a small function and a corresponding test file:
     - File: internal/demo/math.go
       package demo
       
       // Add returns the sum of a and b.
       func Add(a, b int) int { return a + b }
     - File: internal/demo/math_test.go
       package demo
       
       import "testing"
       
       func TestAdd(t *testing.T) {
         if got := Add(2, 3); got != 5 {
           t.Fatalf("Add(2,3) = %d; want 5", got)
         }
       }
  2) Run the tests from the project root:
     - PowerShell: go test ./...
     - Bash: go test ./...
  3) Clean up (if these were just demonstration files):
     - PowerShell: Remove-Item -Recurse -Force .\internal\demo
     - Bash: rm -rf internal/demo
- Expected output for the example: ok  awesomeProject/internal/demo  (timing)

CI suggestions
- Add a workflow to run on push/PR: setup-go (>=1.23), go test ./..., and archive JSON logs if your binary supports --json. For Windows runners, prefer powershell terminal.


Additional Development Information

- Source of truth: The Linux/macOS implementation (ShellBasedPackageManager/scripts/linux-macos/linux_macos_full.sh) defines functional behavior. The Go binary must preserve semantics and flags. When behavior diverges, align Go with the shell script.
- Installers: Keep ShellBasedPackageManager/flutter_packagemanager_setup/install/install.sh and install.ps1 as bootstrap-only one-line installers that fetch the latest flutter-pm binary to PATH, then exec it. Do not re-implement functionality in the installers.
- Git fidelity: Always call the git CLI (handle LFS, submodules, and credentials) rather than using libraries that may miss edge cases.
- Pub operations: Use dart pub add or flutter pub add; avoid editing pubspec.yaml directly. Before any modification, create a timestamped backup. Afterwards, run pub get.
- Discovery: Implement "nearest pubspec" by walking up from CWD. Also provide a local scan of common roots (~/Development, ~/Projects, ~/dev; configurable).
- Stale detection: Heuristic (lockfile age > 24h) plus precise check comparing git ls-remote ref against the sha in pubspec.lock for git deps.
- Recommendations: Start with rules to pin floating branches, suggest shallow/sparse clones for big repos, share caches, and propose safe pub upgrades.
- TUI first: Default entry launches a Bubble Tea TUI with steps DetectProject → ChooseSource → ListRepos → EditSpecs → Confirm → Execute → Summary/Recos. Offer CLI subcommands for CI: add, sync, status, reco.
- Cross-platform hardening: Use filepath for paths, avoid rename-on-open on Windows, detect tool availability (git/dart/flutter/gh) and print exact install hints (brew/apt/winget).
- Logs and dry-run: Provide --dry-run and --json flags built-in. Emit exact commands executed (optionally under --explain) to ease parity checks with the macOS/Linux scripts.
- Testing to evaluate everything: For each new feature, write tests first, run them locally on Windows and at least one Unix environment, and compare results to the shell script behavior. Only ship changes when tests and parity checks are green.

Housekeeping for this documentation task
- The example test workflow above was executed and passed with go1.25.1. Any temporary files created for validation should be removed before committing. Only this file (.junie/guidelines.md) should persist as part of this task’s output.
