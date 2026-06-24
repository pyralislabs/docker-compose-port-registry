# compose-port-registry

A deterministic port-collision linter and allocator for workspaces containing many Docker Compose
projects.

[![CI](https://github.com/pyralis-labs/compose-port-registry/workflows/CI/badge.svg)](https://github.com/pyralis-labs/compose-port-registry/actions)
[![Release](https://github.com/pyralis-labs/compose-port-registry/workflows/Release/badge.svg)](https://github.com/pyralis-labs/compose-port-registry/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/pyralis-labs/compose-port-registry)](https://goreportcard.com/report/github.com/pyralis-labs/compose-port-registry)
[![Go Reference](https://pkg.go.dev/badge/github.com/pyralis-labs/compose-port-registry.svg)](https://pkg.go.dev/github.com/pyralis-labs/compose-port-registry)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## Why

Multiple Compose projects commonly publish the same host port. Each file may be valid alone, but
starting projects together fails or exposes a service on an unintended interface.
`compose-port-registry` scans the tree as one workspace and reports those conflicts before runtime.

## Quick Start

```text
# Scan the current directory for collisions
compose-port-registry .

# Scan with suggestions for resolution
compose-port-registry ~/projects --suggest

# JSON output for automation
compose-port-registry . --format json

# Dry-run a fix plan
compose-port-registry . --fix --dry-run

# Apply suggested fixes
compose-port-registry . --fix --yes
```

## Example Output

```text
$ compose-port-registry ~/projects --suggest

COLLISION tcp 0.0.0.0:8080
  alpha/api    /home/user/projects/alpha/compose.yaml    8080 -> 80
  beta/web     /home/user/projects/beta/docker-compose.yml 8080 -> 3000

PLANNED beta/web: 8080:3000 -> 4000:3000

Found 1 collision(s).
```

## CLI

```text
compose-port-registry [ROOT...] [flags]
compose-port-registry --file PATH [--file PATH...] [flags]
```

Flags:

| Flag | Default | Description |
| --- | --- | --- |
| `--file` | | Explicit Compose file; repeatable for override stacks |
| `--project-dir` | | Compose project directory for explicit files |
| `--env-file` | | Interpolation environment file; repeatable |
| `--profile` | | Active Compose profile; repeatable |
| `--range` | `4000-4999` | Allocation port range (START-END) |
| `--exclude` | | Exclude path glob; repeatable |
| `--format` | `text` | Output format: `text` or `json` |
| `--suggest` | `false` | Include deterministic replacement suggestions |
| `--fix` | `false` | Apply supported suggestions |
| `--dry-run` | `false` | Validate and report the edit plan without committing |
| `--backup-suffix` | `.port-registry.bak` | Backup file suffix |
| `--no-backup` | `false` | Disable backups; requires `--yes` |
| `--yes` | `false` | Acknowledge mutation without interactive prompt |
| `--strict` | `false` | Treat unsupported constructs as errors |
| `--version` | | Print version |

Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | Scan completed with no collisions |
| `1` | Collisions found; no operational failure |
| `2` | Invalid CLI/configuration |
| `3` | Discovery, parse, interpolation, or merge failure |
| `4` | Fix refused, partially failed, or post-write verification failed |
| `5` | Internal error |

## Collision Semantics

A published binding is modeled as host IP scope, protocol, and host-port interval.

- Same host port over TCP and UDP: no conflict.
- Same host port on two different specific host IPs: no conflict.
- `0.0.0.0:8080` conflicts with every IPv4 bind on port `8080`.
- `[::]:8080` conflicts with every IPv6 bind on port `8080`.
- IPv4 and IPv6 wildcard binds are treated separately by default.
- Any intersection between published ranges is a collision.
- Target/container ports are reported but do not determine host collisions.
- Duplicate effective bindings inside one project are collisions too.

## Compose Scope

Discovery recognizes these conventional base files:

```text
compose.yaml
compose.yml
docker-compose.yaml
docker-compose.yml
```

Automatic discovery treats each base file's directory as a project and does not guess which
override files should be applied. Override stacks must be passed explicitly with repeated
`--file`, matching Compose file order.

The loader follows Compose interpolation and merge behavior via `compose-go/v2`. Unresolved or
unsupported constructs are warnings by default and errors under `--strict`.

## Fix Safety

`--fix` is not a general YAML formatter. V1 fixes only unambiguous literal published ports that
can be changed without altering interpolation variables, ranges, or long-syntax semantics. Each
edit must map to one unambiguous source node. Refused mutations include:

- ranges
- interpolated values
- long syntax
- values from override files
- YAML anchor/alias origins
- ambiguous duplicate scalars
- read-only files
- files changed after scan

Backups are on by default, writes are atomic, and a fresh scan must pass before the operation
succeeds. `--fix --dry-run` validates and reports the complete plan without committing changes.

## Documentation

- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md): Compose and collision semantics
- [`docs/CODE_STANDARDS.md`](docs/CODE_STANDARDS.md): implementation standards
- [`docs/TESTING.md`](docs/TESTING.md): fixture matrix and test strategy
- [`docs/ROADMAP.md`](docs/ROADMAP.md): phases and acceptance criteria

## License

MIT &mdash; see [LICENSE](LICENSE).
