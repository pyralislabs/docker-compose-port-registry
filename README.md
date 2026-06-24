# compose-port-registry

A deterministic port-collision linter and allocator for workspaces containing many Docker Compose
projects.

> Status: implementation-grade bootstrap specification. No product code exists yet.

## Why

Multiple Compose projects commonly publish the same host port. Each file may be valid alone, but
starting projects together fails or exposes a service on an unintended interface.
`compose-port-registry` scans the tree as one workspace and reports those conflicts before runtime.

## Intended Behavior

```text
$ compose-port-registry scan ~/projects --suggest

COLLISION tcp 0.0.0.0:8080
  alpha/api    alpha/compose.yaml:12    8080 -> 80
  beta/web     beta/docker-compose.yml:9 8080 -> 3000

SUGGEST beta/web: replace published port 8080 with 4000
```

The tool will:

- discover Compose projects beneath one or more roots
- resolve effective Compose configurations, including selected overrides and interpolation
- normalize short and long port syntax into bind intervals
- detect collisions across projects, services, protocols, host IPs, and ranges
- provide deterministic next-free-port suggestions
- optionally apply a narrow, guarded set of fixes
- produce stable human and versioned JSON output

## Collision Semantics

A published binding is modeled as host IP scope, protocol, and host-port interval.

- Same host port over TCP and UDP: no conflict.
- Same host port on two different specific host IPs: no conflict.
- `0.0.0.0:8080` conflicts with every IPv4 bind on port `8080`.
- `[::]:8080` conflicts with every IPv6 bind on port `8080`.
- IPv4 and IPv6 wildcard binds are treated separately by default because dual-stack runtime
  behavior is host-dependent. A future strict dual-stack mode may conservatively overlap them.
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
`--file`, matching Compose file order. This avoids silently analyzing a configuration the user
would never run.

The loader follows Compose interpolation and merge behavior as documented in
[`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md). Unresolved or unsupported constructs are warnings
by default and errors under `--strict`.

## Planned CLI

```text
compose-port-registry scan [ROOT...] [flags]
compose-port-registry [ROOT...] [flags]        # scan is the default command
```

Examples:

```text
compose-port-registry .
compose-port-registry ~/projects --suggest --range 4000-4999
compose-port-registry --file compose.yaml --file compose.dev.yaml --project-dir .
compose-port-registry . --format json
compose-port-registry . --fix --dry-run
compose-port-registry . --fix --yes
```

`--fix` is not a general YAML formatter. V1 fixes only unambiguous literal published ports that
can be changed without altering interpolation variables, ranges, or long-syntax semantics. Each
edit must map to one unambiguous source node; a transaction may safely include several files.
Backups are on by default, writes are atomic, and a fresh scan must pass before the operation
succeeds. `--fix --dry-run` validates and reports the complete plan without committing changes.

## Documentation

- [`bootstrap.md`](bootstrap.md): product decision, tree, and public contracts
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md): Compose and collision semantics
- [`docs/CODE_STANDARDS.md`](docs/CODE_STANDARDS.md): implementation standards
- [`docs/TESTING.md`](docs/TESTING.md): fixture matrix and test strategy
- [`docs/ROADMAP.md`](docs/ROADMAP.md): phases and acceptance criteria

## License And Positioning

Recommended license: MIT. The project is a developer credibility and goodwill tool associated with
Pyralis Labs. It is infrastructure software, not a replacement for any monetized interactive site
tool.
