# Bootstrap Plan

## Product Decision

Build `compose-port-registry` as a Go command-line application distributed as one
dependency-free binary. Go is preferred over Node because the tool is intended to run in CI,
pre-commit hooks, homelabs, and mixed-language workspaces without requiring a runtime.

The first release is a collision linter with deterministic allocation suggestions. A guarded
`--fix` workflow follows only after read-only behavior and fixtures are mature.

## Product Contract

Given one or more roots, the tool discovers Compose projects, resolves their effective published
ports, and reports bind conflicts between projects. It may suggest or safely apply replacement
published ports from an allowed range.

The conflict key is:

```text
effective host IP + protocol + published host port
```

Container target ports do not conflict. TCP and UDP do not conflict. Distinct specific host IPs do
not conflict. A wildcard bind conflicts with every specific address in the same IP family.

## Recommended Repository Tree

```text
.
|-- AGENTS.md
|-- README.md
|-- bootstrap.md
|-- cmd/
|   `-- compose-port-registry/
|       `-- main.go
|-- docs/
|   |-- ARCHITECTURE.md
|   |-- CODE_STANDARDS.md
|   |-- ROADMAP.md
|   `-- TESTING.md
|-- internal/
|   |-- app/             # command orchestration and exit-code mapping
|   |-- allocate/        # deterministic next-port selection
|   |-- collision/       # bind overlap and range-conflict engine
|   |-- compose/         # discovery, loading, interpolation, merge normalization
|   |-- config/          # CLI/config defaults and validation
|   |-- fix/             # edit plans, backups, atomic writes, verification
|   |-- model/           # normalized projects, bindings, findings
|   `-- report/          # text and versioned JSON output
|-- testdata/
|   |-- fixtures/
|   `-- golden/
|-- .github/
|   `-- workflows/
|       |-- ci.yml
|       `-- release.yml
|-- .goreleaser.yaml
|-- go.mod
|-- go.sum
|-- LICENSE
`-- SECURITY.md
```

Use `github.com/compose-spec/compose-go/v2` for Compose loading and semantics where practical.
Use `gopkg.in/yaml.v3` only for source-location-aware mutation. Do not build a second general
Compose parser.

## CLI Contract

Proposed command:

```text
compose-port-registry scan [ROOT...]
```

`scan` is the default command when omitted.

Core flags:

```text
--file PATH             explicit Compose file or repeated base/override files
--project-dir PATH      Compose project directory for explicit files
--env-file PATH         interpolation environment file; repeatable
--profile NAME          active Compose profile; repeatable
--range START-END       allocation range, default 4000-4999
--exclude GLOB          exclude path; repeatable
--format text|json      output format, default text
--suggest               include deterministic replacement suggestions
--fix                   apply supported suggestions
--dry-run               with --fix, validate and report the edit plan without committing
--backup-suffix VALUE   backup suffix, default .port-registry.bak
--no-backup             disable backups; requires --yes
--yes                   acknowledge mutation without interactive prompt
--strict                treat unsupported/indeterminate Compose constructs as errors
--version               print version
```

Exit codes:

| Code | Meaning |
| --- | --- |
| `0` | scan completed with no collisions |
| `1` | collisions found; no operational failure |
| `2` | invalid CLI/configuration |
| `3` | discovery, parse, interpolation, or merge failure |
| `4` | fix refused, partially failed, or post-write verification failed |
| `5` | internal error |

Human output goes to stdout. Diagnostics go to stderr. JSON mode writes exactly one JSON document
to stdout and diagnostics to stderr.

`--dry-run` requires `--fix`. It performs fix eligibility checks and full plan validation but never
replaces an original file or creates a persistent backup.

## JSON Contract

The report is versioned from day one:

```json
{
  "schema_version": "1",
  "tool_version": "0.1.0",
  "roots": ["/workspace"],
  "summary": {
    "projects": 2,
    "bindings": 4,
    "collisions": 1,
    "warnings": 0,
    "fixes_planned": 0,
    "fixes_applied": 0
  },
  "collisions": [
    {
      "id": "collision:tcp:ipv4-any:8080",
      "protocol": "tcp",
      "host_ip": "0.0.0.0",
      "published": {"start": 8080, "end": 8080},
      "bindings": []
    }
  ],
  "warnings": [],
  "fixes": []
}
```

All arrays are deterministically sorted. New optional fields may be added within schema version 1;
breaking changes require a new schema version.

## Bootstrap Sequence

- [x] 1. Initialize the Go module, command shell, version package, and exit-code contract.
- [x] 2. Implement discovery and normalized read-only scan behavior.
- [x] 3. Implement collision detection and deterministic text/JSON reports.
- [x] 4. Add allocator suggestions and fixture coverage.
- [x] 5. Add guarded `--fix` for the narrow supported mutation set.
- [x] 6. Add CI, cross-platform release automation, security policy, and signed checksums.

## Initial Non-Goals

- Reserving ports with a daemon or central server
- Inspecting currently listening OS sockets or running Docker containers
- Editing Swarm `deploy` ports, Kubernetes manifests, Helm charts, or Dockerfiles
- Reformatting whole Compose files
- Automatically resolving environment-dependent, templated, or generated Compose files
- Guaranteeing that a currently free port remains free at runtime
- Supporting arbitrary Compose extensions beyond what the Compose loader can normalize

## Bootstrap Definition Of Done

- [x] The seven requested planning files exist and agree on scope and semantics.
- [x] Implementation can start without unresolved decisions about package ownership, CLI behavior,
  collision semantics, mutation safety, test strategy, or release targets.
- [x] Product code is implemented and passes all quality gates.
