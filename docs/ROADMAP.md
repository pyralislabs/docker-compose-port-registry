# Roadmap

## Product Goal

Ship a trustworthy, deterministic, single-binary Compose port-collision linter. Read-only analysis
must be mature before automatic mutation is enabled.

## Phase 0: Repository Foundation

Deliver:

- Go module and package tree from `bootstrap.md`
- thin CLI shell, version command, configuration validation, exit codes
- CI for formatting, vetting, tests, race detector, static analysis, and vulnerability scan
- MIT license, contribution guide, security policy, and release automation skeleton

Acceptance criteria:

- binary builds on Linux, macOS, and Windows
- documented command and exit-code contract is tested
- CI is required and green
- no scanning behavior is claimed yet

## Phase 1: Read-Only Discovery And Normalization

Deliver:

- recursive conventional-file discovery and explicit ordered `--file` stacks
- exclusions, symlink policy, ambiguity diagnostics
- Compose loading, interpolation, profiles, and merge behavior
- normalized binding model with source provenance
- deterministic text and schema-versioned JSON inventory output

Acceptance criteria:

- required discovery, Compose syntax, merge, profile, and interpolation fixtures pass
- unsupported/indeterminate constructs are visible and strict mode promotes them to errors
- reports never expose environment values
- repeated scans produce byte-stable JSON

## Phase 2: Collision Linter

Deliver:

- protocol-aware, host-scope-aware interval collision engine
- exact overlap findings across projects, services, and ranges
- stable grouping and source-rich human reports
- documented exit code `1` for collisions

Acceptance criteria:

- complete collision fixture matrix passes on all platforms
- no known false negatives inside the documented support boundary
- wildcard and specific-IP behavior matches architecture rules
- 10,000-file benchmark establishes an acceptable baseline

This phase is the first useful public release candidate (`v0.1.0`).

## Phase 3: Deterministic Suggestions

Deliver:

- configurable allocation range
- deterministic winner selection and equal-width next-port allocation
- allocation exhaustion diagnostics
- suggestion records in text and JSON

Acceptance criteria:

- allocator property tests prove no overlap and no out-of-range results
- suggestions are identical across repeated runs and platforms
- all existing concrete bindings and earlier suggestions are reserved
- exhausted ranges never cause reuse or partial allocation

This phase is suitable for `v0.2.0`.

## Phase 4: Guarded Fixes

Deliver:

- edit eligibility/refusal engine
- scalar-only YAML edits for the documented v1 mutable set
- backup-by-default, same-directory temporary writes, atomic replacement, rollback
- stale-file detection and mandatory post-write verification
- dry-run edit plan in text and JSON

`--fix` always implies a complete dry-run plan before commit. Without `--yes`, an interactive
terminal asks for confirmation; non-interactive sessions refuse.

Acceptance criteria:

- every fix-safety fixture passes on Linux, macOS, and Windows
- exact diffs show no unrelated formatting changes
- induced failures leave originals unchanged or restore them from backups
- successful fixes remove planned collisions and preserve unrelated normalized bindings
- unsupported or ambiguous cases fail closed with machine-readable reasons

This phase is suitable for `v0.3.0`; do not label fixes stable before real-world soak time.

## Phase 5: Stable Release

Deliver:

- documentation polished against actual behavior
- shell completion/manpage if justified
- GoReleaser artifacts and checksums
- signed release provenance where supported
- Homebrew/Scoop/package-manager publication only after artifact workflow is stable
- compatibility policy and deprecation process

Acceptance criteria:

- all documented behavior has fixtures
- no high/critical known dependency vulnerabilities
- artifacts smoke-tested on all target platforms
- security reporting and release verification instructions published
- at least one release cycle validates JSON backward compatibility

Stable `v1.0.0` definition:

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
- Publish a `SECURITY.md` before the first public release.

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
