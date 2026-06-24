# Roadmap

## Product Goal

Ship a trustworthy, deterministic, single-binary Compose port-collision linter with guarded
automatic mutation.

## Current Status: v0.1.0

All bootstrap phases are complete. The current release covers read-only discovery, normalization,
collision detection, deterministic suggestions, and guarded `--fix` for literal short-syntax ports.

## Completed Phases

### Phase 0: Repository Foundation ✓

- Go module and package tree matching the architecture plan
- CLI shell, version command, configuration validation, exit codes
- CI for formatting, vetting, tests, race detector
- MIT license, security policy, GoReleaser release automation

### Phase 1: Read-Only Discovery And Normalization ✓

- Recursive conventional-file discovery and explicit ordered `--file` stacks
- Exclusions, symlink policy, ambiguity diagnostics
- Compose loading via `compose-go/v2`, interpolation, profiles, merge behavior
- Normalized binding model with source provenance
- Deterministic text and schema-versioned JSON inventory output

### Phase 2: Collision Linter ✓

- Protocol-aware, host-scope-aware interval collision engine
- Exact overlap findings across projects, services, and ranges
- Stable grouping and source-rich human reports
- Exit code `1` for collisions

### Phase 3: Deterministic Suggestions ✓

- Configurable allocation range (default `4000-4999`)
- Deterministic winner selection and equal-width next-port allocation
- Allocation exhaustion diagnostics
- Suggestion records in text and JSON

### Phase 4: Guarded Fixes ✓

- Edit eligibility/refusal engine with typed mutability
- Scalar-only YAML edits for the documented v1 mutable set
- Backup-by-default, same-directory temporary writes, atomic replacement, rollback
- Stale-file detection and post-write verification
- Dry-run edit plan in text and JSON

## Phase 5: Stable Release (Current)

Focus:

- Documentation polished against actual behavior
- shell completion/manpage if justified
- GoReleaser artifacts and checksums
- signed release provenance where supported
- Homebrew/Scoop/package-manager publication after artifact workflow is stable
- compatibility policy and deprecation process

Acceptance criteria:

- all documented behavior has fixtures
- no high/critical known dependency vulnerabilities
- artifacts smoke-tested on all target platforms
- security reporting and release verification instructions published
- at least one release cycle validates JSON backward compatibility

`v1.0.0` definition:

- scan, collision, suggestion, and guarded-fix contracts are reliable
- no open correctness issue that can corrupt a Compose file
- platform-specific transaction limitations are documented and tested

## Future Candidates, Not Commitments

- configuration file for workspace policy and reserved ranges
- GitHub annotations and SARIF output
- strict dual-stack host-overlap mode
- runtime comparison against Docker or listening sockets
- controlled support for long-syntax and range fixes
- plugin/library API

Each candidate requires a written architecture update and fixtures before implementation.

## Security And Release Policy

- Treat all scanned files as untrusted.
- Never execute Compose, shells, hooks, containers, or network requests during a scan.
- Run `govulncheck`, dependency review, static analysis, and secret scanning in CI.
- Use least-privilege GitHub Actions permissions and pin third-party actions by commit.
- Build reproducible release artifacts with checksums for Linux/macOS/Windows on `amd64` and
  `arm64`.
- `SECURITY.md` is published.

## Explicit Non-Goals Through V1

- central port reservation service or daemon
- distributed locking between users
- runtime guarantee that Docker can bind a suggested port
- automatic changes to environment files, generated Compose files, overrides, anchors, aliases,
  includes, long syntax, or ranges
- Docker Swarm/Kubernetes orchestration validation
- a GUI or hosted service
- telemetry

## Definition Of Done For Any Feature

A feature is done only when:

- behavior and failure modes are documented
- package boundaries remain coherent
- unit, fixture, CLI, and platform tests appropriate to the risk pass
- text and JSON outputs are deterministic
- security/privacy implications are addressed
- unsupported cases fail visibly
- mutation changes include backup, rollback, and verification coverage
- CI and release behavior remain green
- user-facing docs match the shipped implementation
