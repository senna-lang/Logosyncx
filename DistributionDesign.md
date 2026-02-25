# Distribution & Installation Design

> How `logos` gets onto users' machines — and stays up to date.

---

## Goals

| Goal | Description |
|------|-------------|
| Zero-dependency install | Users should not need Go, Node.js, or any runtime |
| One-command install | A single command installs the binary and puts it on `$PATH` |
| Cross-platform | macOS (arm64 / amd64), Linux (arm64 / amd64), Windows (amd64) |
| Update mechanism | Users can upgrade to a new release without re-running the full install |
| Update awareness | Running any `logos` command hints if a newer version is available |

---

## Distribution Channels

### 1. Homebrew Tap (primary for macOS / Linux)

```sh
brew install senna-lang/tap/logos
```

- Tap repository: `github.com/senna-lang/homebrew-tap`
- Formula file: `Formula/logos.rb`
- Updating: `brew upgrade logos`
- GoReleaser generates the formula and sends a PR to the tap automatically on each release

### 2. `curl | bash` installer (primary for Linux / CI)

```sh
curl -sSfL https://raw.githubusercontent.com/senna-lang/logosyncx/main/scripts/install.sh | bash
```

- Detects OS and architecture
- Downloads the correct pre-built binary from GitHub Releases
- Verifies the SHA256 checksum
- Installs to `~/.local/bin` (fallback: `/usr/local/bin` if writable)
- Ensures the install directory is on `$PATH` (prints a hint if not)

Pinning to a specific version:

```sh
curl -sSfL https://raw.githubusercontent.com/senna-lang/logosyncx/main/scripts/install.sh | \
  LOGOS_VERSION=v0.2.0 bash
```

### 3. GitHub Releases (direct binary download)

Pre-built archives published as GitHub Release assets for every tag:

| Asset name | Platform |
|------------|----------|
| `logos_darwin_arm64.tar.gz` | macOS Apple Silicon |
| `logos_darwin_amd64.tar.gz` | macOS Intel |
| `logos_linux_arm64.tar.gz` | Linux ARM64 |
| `logos_linux_amd64.tar.gz` | Linux x86-64 |
| `logos_windows_amd64.zip` | Windows x86-64 |
| `checksums.txt` | SHA256 hashes for all archives |

### 4. `go install` (for Go developers)

```sh
go install github.com/senna-lang/logosyncx@latest
```

Requires Go 1.21+. The resulting binary has version set to `(devel)` unless built with the release ldflags.

---

## Versioning

- **Scheme**: Semantic versioning — `vMAJOR.MINOR.PATCH`
- **Canonical source**: git tags (`v0.1.0`, `v0.2.0`, …)
- **Embedded at build time** via `go build -ldflags`:

```sh
go build -ldflags "-X github.com/senna-lang/logosyncx/internal/version.Version=v0.1.0" -o logos .
```

- **Default value**: `dev` (used when building locally without ldflags)
- **`logos version`** output:

```
logos v0.2.0 (darwin/arm64)
```

### New file: `internal/version/version.go`

```go
package version

// Version is set at build time via -ldflags.
// It defaults to "dev" for local builds.
var Version = "dev"
```

---

## Release Pipeline

### Tooling: GoReleaser

[GoReleaser](https://goreleaser.com) is the de facto standard for Go CLI distribution. It handles:

- Cross-compilation via GOOS/GOARCH matrix
- Archive creation (`.tar.gz` for Unix, `.zip` for Windows)
- SHA256 `checksums.txt` generation
- GitHub Release creation and asset upload
- Homebrew formula generation and tap PR

Config file: `.goreleaser.yaml` in the repository root.

### Trigger: pushing a semver tag

```sh
git tag v0.2.0
git push origin v0.2.0
```

This triggers the GitHub Actions release workflow.

### GitHub Actions workflow: `.github/workflows/release.yml`

```
Trigger: push tag matching v*.*.*
Steps:
  1. Checkout (with full history for GoReleaser changelog)
  2. Set up Go
  3. Run GoReleaser (goreleaser release --clean)
     - Builds all platform binaries
     - Creates archives + checksums.txt
     - Creates/updates GitHub Release
     - Opens PR on homebrew-tap with updated formula
```

### `.goreleaser.yaml` key sections

```yaml
builds:
  - binary: logos
    ldflags:
      - -s -w
      - -X github.com/senna-lang/logosyncx/internal/version.Version={{.Version}}
    goos: [darwin, linux, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: checksums.txt

brews:
  - repository:
      owner: senna-lang
      name: homebrew-tap
    homepage: https://github.com/senna-lang/logosyncx
    description: "Shared AI agent conversation context — stored in git"
    install: |
      bin.install "logos"
    test: |
      system "#{bin}/logos", "version"
```

---

## `scripts/install.sh`

The install script is the entry point for the `curl | bash` channel. Key responsibilities:

1. **Detect platform** — `uname -s` (OS) and `uname -m` (arch), map to GoReleaser asset names
2. **Resolve version** — use `$LOGOS_VERSION` env var if set, otherwise query GitHub API for the latest tag
3. **Download archive** — from `https://github.com/senna-lang/logosyncx/releases/download/<version>/logos_<os>_<arch>.tar.gz`
4. **Verify checksum** — download `checksums.txt`, extract the line for the target archive, compare with `sha256sum` / `shasum -a 256`
5. **Install binary** — extract and place `logos` in `$INSTALL_DIR` (default: `~/.local/bin`)
6. **PATH hint** — if the install directory is not on `$PATH`, print a shell-specific export hint

```
Supported OS:        Darwin, Linux
Supported arch:      x86_64 → amd64, arm64 / aarch64 → arm64
Dependencies:        curl (or wget), tar (or unzip), shasum / sha256sum
```

---

## `logos update` Command

Allows users to self-update the installed binary without going through a package manager.

### Usage

```sh
logos update            # check for a newer version and install it if found
logos update --check    # check only; print status, do not install
```

### Implementation (`cmd/update.go`)

```
1. GET https://api.github.com/repos/senna-lang/logosyncx/releases/latest
   → parse .tag_name (e.g. "v0.2.0")

2. Compare tag_name with internal/version.Version
   - If equal or current is newer  →  "Already up to date."
   - If current is "dev"           →  warn "Cannot update a dev build." and exit

3. Find the matching release asset for runtime.GOOS + runtime.GOARCH

4. Download the archive to a temp directory

5. Verify SHA256 against checksums.txt in the same release

6. Extract the logos binary from the archive

7. Determine the path of the running binary (os.Executable())

8. Atomic replace:
   a. Write new binary to <current-path>.new
   b. os.Rename(<current-path>.new, <current-path>)
   (Rename is atomic on Unix; on Windows use the move-and-delete pattern)

9. Print: "Updated logos to v0.2.0"
```

### Error handling

| Situation | Behaviour |
|-----------|-----------|
| No network | Print error, exit 1 |
| GitHub API rate-limited | Print error with retry hint, exit 1 |
| Checksum mismatch | Delete temp file, print error, exit 1 |
| Binary not writable (e.g. `/usr/local/bin` without sudo) | Print error with `sudo logos update` hint |
| Windows running binary cannot be replaced in-place | Extract alongside, print rename instruction |

---

## Background Update Notification

Every time `logos` runs any command, it optionally checks in the background whether a newer version exists and prints a one-line hint **after** the command output (never before, to avoid breaking piped JSON output).

### Rules

| Rule | Detail |
|------|--------|
| At most once per day | Last check timestamp stored in `~/.config/logosyncx/last-update-check` |
| Non-blocking | Spawned as a goroutine; the main command does not wait for it |
| Suppressed for `--json` output | No notice printed when `--json` flag is active (breaks piped parsing) |
| Suppressed when `LOGOS_NO_UPDATE_CHECK=1` | Opt-out for CI environments |
| Suppressed for `dev` builds | No point notifying on local builds |

### Output example

```
$ logos ls
DATE                 TOPIC                    TAGS
2025-02-20 10:30    auth-refactor            auth, jwt

A new version of logos is available: v0.3.0
Run 'logos update' to upgrade.
```

### Cache file format

```
v0.3.0 2025-03-01T09:00:00Z
```

One line: `<latest-version> <check-time-RFC3339>`. Written atomically.

---

## Homebrew Tap Repository Structure

Repository: `github.com/senna-lang/homebrew-tap`

```
homebrew-tap/
└── Formula/
    └── logos.rb      ← generated and updated by GoReleaser on each release
```

GoReleaser opens a pull request against this repository automatically. Merging the PR publishes the new version to Homebrew users.

---

## Files to Create / Modify

| Path | Action | Description |
|------|--------|-------------|
| `internal/version/version.go` | Create | Version variable, set via ldflags |
| `cmd/update.go` | Create | `logos update` command |
| `cmd/root.go` | Modify | Wire background update check into PersistentPostRun |
| `main.go` | Modify | Nothing needed; version package is imported transitively |
| `.goreleaser.yaml` | Create | GoReleaser configuration |
| `.github/workflows/release.yml` | Create | Release pipeline |
| `.github/workflows/ci.yml` | Create | PR / push CI (test + lint) |
| `scripts/install.sh` | Create | curl-pipe installer |
| `Makefile` | Modify | Add `release-dry-run` and `snapshot` targets |
| `README.md` | Modify | Replace "Build from source" section with Homebrew / curl instructions |

---

## Makefile Additions

```makefile
## snapshot: build a local snapshot with GoReleaser (no publish)
snapshot:
	goreleaser build --snapshot --clean

## release-dry-run: full release dry run (no publish, no git tag required)
release-dry-run:
	goreleaser release --snapshot --clean

## release: tag and push to trigger the release pipeline
release: fmt lint test
	@echo "Current version: $$(git describe --tags --abbrev=0 2>/dev/null || echo 'none')"
	@read -p "New version tag (e.g. v0.2.0): " tag; \
	  git tag $$tag && git push origin $$tag
```

---

## Updated Installation Section for `README.md`

```markdown
## Installation

### Homebrew (macOS / Linux)

    brew install senna-lang/tap/logos

### curl | bash (Linux / macOS / CI)

    curl -sSfL https://raw.githubusercontent.com/senna-lang/logosyncx/main/scripts/install.sh | bash

### Direct download

Download the pre-built binary for your platform from the
[latest GitHub Release](https://github.com/senna-lang/logosyncx/releases/latest).

### go install (Go developers)

    go install github.com/senna-lang/logosyncx@latest

### Updating

    logos update          # self-update to the latest release
    brew upgrade logos    # if installed via Homebrew
```

---

## Rollout Plan

| Step | Work item | Notes |
|------|-----------|-------|
| 1 | Create `internal/version/version.go` | Tiny, no behaviour change |
| 2 | Update `main.go` / `cmd/root.go` to use version package | Update `logos version` output |
| 3 | Create `.goreleaser.yaml` | Test with `make snapshot` |
| 4 | Create `.github/workflows/release.yml` | Can test with a `v0.0.x` pre-release tag |
| 5 | Create `scripts/install.sh` | Test on macOS and Linux |
| 6 | Create `homebrew-tap` repository | One-time setup |
| 7 | Implement `cmd/update.go` | After at least one real release exists to test against |
| 8 | Add background update check | After `update` command is stable |
| 9 | Update `README.md` | After install path is confirmed working end-to-end |

---

## Open Questions

**1. Install directory for `install.sh`**
`~/.local/bin` is writable without sudo but not always on `$PATH` by default (especially on macOS).
Alternative: prompt the user or default to `/usr/local/bin` with a sudo fallback.

**2. Windows support priority**
Windows users are less likely to use AI agent CLIs. Ship the binary in GitHub Releases, but
defer the installer script and Chocolatey/Scoop packages until there is user demand.

**3. Signed binaries (macOS Gatekeeper)**
Unsigned binaries on macOS require users to run `xattr -dr com.apple.quarantine $(which logos)`
after install. Code-signing via an Apple Developer certificate and `goreleaser`'s `notarize`
hook eliminates this but requires a paid Apple Developer account. Defer until post-1.0.

**4. Update check in CI**
Background update checks should be disabled in CI to avoid flaky output and unnecessary
GitHub API calls. `LOGOS_NO_UPDATE_CHECK=1` is the escape hatch; document it prominently.