# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Test Commands

```bash
# Build
go build ./cmd/restic
go run build.go              # Official build script with version info
go run build.go -T            # Build and run tests
go run build.go --tags debug  # Debug build with profiling support

# Run all tests
go test ./...
make test                     # Same as above

# Run single test
go test ./cmd/restic -run TestBackup

# Run tests with race detector
go test -race ./...

# Run FUSE tests (requires FUSE)
RESTIC_TEST_FUSE=true go test ./...

# Lint
golangci-lint run

# Check changelog entries
calens
```

## Code Structure

**Entry point:** `cmd/restic/main.go` - Cobra-based CLI with these command groups:
- Default commands: backup, restore, snapshots, forget, prune, check, init, key, etc.
- Advanced commands: debug, migrate, repair, recover, diff, stats

**Core internal packages:**

| Package | Responsibility |
|---------|----------------|
| `archiver` | Reads files, splits into chunks, saves to repository |
| `backend` | Storage backends (local, sftp, REST, S3, Azure, B2, GCS, Swift, rclone) |
| `checker` | Repository integrity verification |
| `crypto` | Encryption/decryption operations |
| `repository` | Pack/blob management, index, low-level repo operations |
| `restorer` | Restores data from repository to target |
| `restic` | Core data structures (ID, Snapshot, Tree, Blob) |
| `fs` | Filesystem abstraction with platform-specific implementations |
| `fuse` | FUSE mount for browsing repositories |
| `ui` | Terminal output, progress bars, JSON formatting |

**Architecture layers (from high to low):**
1. `cmd/restic` - CLI commands
2. `archiver` / `restorer` - Backup/restore logic
3. `repository` - Pack/blob handling
4. `backend` - Storage abstraction
5. `crypto` - Encryption (used throughout)

Backends must not import `internal/restic` or `internal/repository` (enforced by golangci-lint).

## Development Workflow

**Changelog entries:** For user-facing changes, add a file in `changelog/unreleased/issue-NNNN`:
- Start with `Bugfix:`, `Enhancement:`, or `Change:` (present tense, imperative mood)
- Include issue/PR URLs at the end
- Use `changelog/TEMPLATE` as reference

**Build tags:**
- `debug` / `profile` - Enable profiling options (`--cpu-profile`, `--mem-profile`, etc.)
- `selfupdate` - Enable `self-update` command
- `disable_grpc_modules` - Reduce binary size for GCS

**Testing notes:**
- Integration tests live alongside unit tests (`*_integration_test.go`)
- FUSE tests require `RESTIC_TEST_FUSE=true` and FUSE installed
- Cloud backend tests require credentials (see CI workflow for env vars)

**Code style:**
- Run `gofmt -w **/*.go` before committing
- Use `golangci-lint run` to catch issues pre-PR
- Commit messages: terse summary, blank line, detailed description
- Allow edits from maintainers on PRs

**Debugging:**
- Set `DEBUG_LOG=/tmp/debug.log` for debug output
- Set `RESTIC_DEBUG_STACKTRACE_SIGINT=true` (Windows) or press `Ctrl-\` (Unix) for stacktraces
- Use `--cpu-profile`, `--mem-profile`, `--trace-profile` with debug builds
