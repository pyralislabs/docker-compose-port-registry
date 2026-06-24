# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 1.x     | :white_check_mark: |
| < 1.0   | :x:                |

## Reporting a Vulnerability

Compose files and `.env` files are untrusted input. The tool never executes Compose commands,
shell expansions, hooks, container images, or network requests during a scan.

To report a security vulnerability, please open a GitHub Security Advisory at:

https://github.com/pyralis-labs/compose-port-registry/security/advisories/new

Do not open public issues for security vulnerabilities.

## Security Properties

- **No remote execution**: The tool never connects to Docker, resolves hostnames, or executes
  shell commands during a scan.
- **No telemetry**: The tool makes no network calls. It is fully offline.
- **No secret exposure**: Environment variable *keys* may appear in error messages, but their
  *values* are never printed or included in reports.
- **Atomic mutations**: Fix transactions use same-directory temporary files, backups, and
  post-write verification. On failure, originals are restored from backups.
- **Stale-file detection**: Source file hashes are captured at plan time and verified before
  commit to reduce time-of-check/time-of-use risk.
- **Path traversal resistance**: Unsupported includes are rejected; file traversal is bounded
  by explicit discovery rules.

## CI Security

- GitHub Actions use least-privilege permissions (`contents: read` for CI, `contents: write`
  for release only).
- Third-party actions are pinned by commit hash.
- Dependency vulnerability scanning (`govulncheck`) runs in CI.
- Release artifacts include SHA-256 checksums for verification.
