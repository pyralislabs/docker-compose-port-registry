# Architecture

## System Shape

`compose-port-registry` is a stateless Go CLI. A scan is a pure pipeline until an explicitly
requested fix is committed:

```text
CLI/config
  -> discover project inputs
  -> load effective Compose models
  -> normalize published bindings
  -> detect collisions
  -> allocate suggestions
  -> render report
  -> optionally plan, commit, and verify fixes
```

No registry server or persistent database is required. Determinism comes from canonical sorting and
a documented allocation algorithm.

## Package Responsibilities

| Package | Responsibility |
| --- | --- |
| `cmd/compose-port-registry` | minimal process entry point |
| `internal/app` | orchestrate commands and map typed failures to exit codes |
| `internal/config` | parse, validate, and freeze CLI/config values |
| `internal/compose` | discover projects, load Compose stacks, preserve source provenance |
| `internal/model` | implementation-independent normalized domain types |
| `internal/collision` | interval overlap and host-IP/protocol conflict logic |
| `internal/allocate` | deterministic replacement-port selection |
| `internal/fix` | edit planning, backups, atomic writes, rollback, verification |
| `internal/report` | deterministic text and versioned JSON renderers |

## Core Domain Model

Conceptual types:

```go
type Project struct {
    ID, Name, Directory string
    Files               []string
    Bindings            []Binding
}

type Binding struct {
    ProjectID, Service, Protocol string
    HostIP                      HostScope
    Published, Target           Interval
    Source                      SourceRef
    Mutability                  Mutability
}

type Interval struct {
    Start, End uint16
}
```

`SourceRef` records the source file, document position when available, original scalar or mapping,
and effective-file stack. `Mutability` explains whether and why a binding can be automatically
changed.

## Discovery And Project Identity

### Automatic tree scan

- Walk each requested root recursively.
- Skip `.git`, dependency/vendor directories, and user-provided exclude globs.
- Recognize conventional base filenames only.
- Treat each discovered base file as one independent project rooted at its directory.
- If multiple conventional base filenames exist in one directory, report ambiguity and require
  explicit `--file`; do not scan all of them as separate projects.
- Canonical project ID is the cleaned absolute project directory plus the ordered file list. Human
  display names use Compose `name`, then project directory basename.
- Symlink policy: do not follow directory symlinks by default. Explicit file symlinks are resolved
  once, and duplicate canonical files are deduplicated.

### Explicit Compose stack

Repeated `--file` values define one project in the exact supplied order. `--project-dir` controls
relative-path resolution and project identity. Automatic discovery and explicit files may not be
combined in one invocation in v1.

## Compose Loading Rules

Use `compose-go/v2` to model the Compose Specification. Pin its version and add compatibility
fixtures before upgrading.

### Interpolation precedence

For deterministic analysis, interpolation inputs are assembled in this order, highest precedence
first:

1. process environment captured once at startup
2. repeated explicit `--env-file` inputs, with later files overriding earlier files
3. default `.env` in the project directory when no `--env-file` was supplied

Compose `env_file` under a service configures the container and does not interpolate the Compose
model. Supported `${VAR}`, defaults, required values, alternatives, and escaped dollar behavior
follow the Compose loader. Any unresolved required value is a load failure. Unresolved optional
values that prevent a port from becoming concrete produce an indeterminate warning, or an error
under `--strict`.

The JSON report records variable names relevant to indeterminate bindings but never emits secret
values.

### Merge behavior

- Files are loaded and merged in supplied order using Compose Specification semantics.
- Relative paths resolve according to Compose loader behavior, rooted at the first/base file or
  explicit project directory as appropriate.
- `ports` are evaluated from the effective merged service, including reset/override tags supported
  by the pinned loader.
- Profiles: services without profiles and services in explicitly active `--profile` sets are
  included. Inactive profiled services are excluded and counted in diagnostics.
- YAML anchors, aliases, extension fields, and `extends` are accepted only to the extent the pinned
  Compose loader resolves them.
- `include` or other newer Compose features are supported only after compatibility fixtures prove
  loader behavior. Otherwise report them as unsupported/indeterminate.

Automatic discovery does not auto-merge `compose.override.yaml`, because doing so would guess the
user's runtime stack. Users pass override files explicitly.

## Port Normalization

Supported input forms include:

```yaml
ports:
  - "8080:80"
  - "127.0.0.1:8080:80/tcp"
  - "8000-8005:80-85"
  - target: 80
    published: "8080"
    host_ip: 127.0.0.1
    protocol: tcp
    mode: host
```

Rules:

- Default protocol is `tcp`.
- Omitted host IP becomes an address-family-unspecified wildcard scope.
- Omitted published port is an ephemeral runtime assignment and does not collide statically.
- Published and target ranges must normalize to valid intervals. Invalid ranges are parse errors.
- Range-length mismatch follows Compose loader validation; it is never silently expanded.
- `expose` is ignored because it does not publish a host port.
- Swarm `mode` is retained in provenance but v1 collision semantics remain host-bind based and
  report a warning when runtime behavior is not locally comparable.

## Host-IP Overlap

Normalize host scopes into:

- `any-unspecified`
- `ipv4-any` (`0.0.0.0`)
- `ipv6-any` (`::`)
- specific IPv4
- specific IPv6
- unresolved/unsupported hostname

Two bindings collide only when protocols match, published intervals intersect, and host scopes
overlap.

- `any-unspecified` overlaps every concrete IP scope.
- An IPv4 wildcard overlaps all IPv4 specifics and IPv4 wildcard.
- An IPv6 wildcard overlaps all IPv6 specifics and IPv6 wildcard.
- Two specific IPs overlap only when equal after canonical parsing.
- IPv4 and IPv6 do not overlap by default.
- Hostnames in `host_ip` are indeterminate, not DNS-resolved.

This is a static conservative model, not a promise about OS-specific dual-stack socket behavior.

## Collision Engine

Sort normalized bindings by protocol, host-scope sort key, published start/end, project ID, service,
and source. Compare interval intersections within compatible protocol and host scopes.

A collision finding groups all mutually conflicting bindings for a concrete overlapping interval.
Findings must be stable independent of filesystem traversal order.

Ranges are first-class. For example, `8000-8010` conflicts with `8005`, and the reported overlap is
`8005-8005`.

## Deterministic Allocation

Allocation uses the configured inclusive range, default `4000-4999`.

1. Build an occupied interval set from every concrete binding in the scan, not only collisions.
2. Preserve one winner for each collision group: the lexicographically first binding by canonical
   project ID, service name, source file, source position, and original published interval.
3. Process remaining conflicting bindings in that same canonical order.
4. Preserve the binding's interval width.
5. Search candidate starts ascending from range start.
6. A candidate is valid only if the full candidate interval is inside the allocation range and
   does not overlap an occupied binding with compatible protocol and host scope.
7. Reserve each accepted candidate immediately before processing the next binding.
8. If no candidate fits, emit `allocation_exhausted`; never wrap, reuse, or partially allocate.

Suggestions are advisory and include the assumptions used. Running the same command against the
same inputs and environment produces the same plan.

## Fix Safety

### V1 mutable set

V1 may mutate only a literal, single-port, short-syntax mapping in one source base file, such as:

```yaml
- "8080:80"
- "127.0.0.1:8080:80/tcp"
```

V1 refuses to mutate:

- ranges
- interpolated values
- long syntax
- values originating from an override, anchor, alias, `extends`, or included/generated file
- ambiguous duplicate scalars
- read-only files
- a file changed after scan

### Transaction

1. Scan and produce a complete edit plan.
2. Require `--yes` for non-interactive mutation.
3. Hash every source file and capture permissions before edits.
4. Create backups beside originals by default; refuse to overwrite existing backups.
5. Parse with `yaml.v3`, locate exact scalar nodes, and make scalar-only edits.
6. Write each candidate file to a same-directory temporary file, flush, sync, and preserve mode.
7. Reparse all temporary results and run the full scan against the planned result.
8. When `--dry-run` is set, delete temporary files, emit the validated plan, and stop successfully.
9. Atomically rename originals to backups and temporary files to originals.
10. Run a fresh post-commit scan.
11. On commit or verification failure, restore every changed original from backup and return exit
    code `4`.

Cross-filesystem atomicity is impossible; same-directory temporary files avoid that case. Windows
rename behavior needs dedicated tests and may require a platform-specific replacement helper.

`--no-backup` requires `--yes`, but still uses temporary files and post-write verification.
`--dry-run` requires `--fix`, creates no persistent backup, and never replaces an original file.

## Reporting And Compatibility

Text output is optimized for people and may gain presentation improvements. JSON is the automation
contract and includes:

- schema/tool version
- normalized scan inputs without secret values
- summary counts
- projects and bindings with provenance
- collisions and exact overlap intervals
- warnings/unsupported constructs
- suggested/planned/applied fixes
- outcome and exit-code meaning

Sort every output collection explicitly. Paths in JSON are absolute by default; a future
`--relative-paths` option may make reports portable.

## Security Model

Compose files and `.env` files are untrusted input.

- Never execute Compose commands, shell expansions, hooks, or container images.
- Never connect to Docker or resolve hostnames in v1.
- Bound file size, YAML nesting, alias expansion, file count, and traversal work.
- Do not print environment values or `.env` contents.
- Reject path escapes introduced by unsupported includes rather than reading arbitrary files.
- Open mutation targets without following swapped symlinks where the platform permits.
- Validate file hashes immediately before commit to reduce time-of-check/time-of-use risk.
- Run dependency vulnerability scanning in CI and publish checksums for releases.
