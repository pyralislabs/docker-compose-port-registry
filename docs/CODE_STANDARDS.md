# Code Standards

## Baseline

- Target the current stable Go release and the previous stable release in CI.
- Use standard-library facilities unless a dependency materially improves correctness.
- Required major dependencies are `compose-go/v2` for effective Compose behavior and `yaml.v3` for
  source-aware narrow edits.
- Pin dependency versions and review Compose-loader upgrades against fixtures.
- Keep the command entry point thin; business behavior belongs under `internal/`.

## Design Rules

- Domain types must not depend on CLI, renderer, or Compose-library types.
- Use explicit constructors and validation for intervals, protocols, host scopes, and source refs.
- Pass immutable configuration and a captured environment map into loaders.
- Pass `context.Context` through filesystem-scale operations.
- Inject filesystem, clock, and randomness only where needed; allocation itself uses no randomness.
- Prefer small interfaces owned by consumers. Do not create interfaces only for hypothetical reuse.
- Return errors with operation and path context while preserving typed causes.
- Never use panics for user-controlled input.

## Determinism

- Never depend on map iteration or filesystem traversal order.
- Canonicalize paths once and sort all projects, bindings, findings, warnings, and fixes.
- Keep the allocation tie-breaker in one tested comparator.
- Golden JSON must be byte-stable except for explicitly normalized tool-version fields.

## Parsing And Data Integrity

- Use structured YAML and Compose APIs; do not parse ports with broad regular expressions alone.
- Preserve original source location and syntax alongside the normalized binding.
- Validate ports as integers in `1-65535`; represent ranges as closed intervals.
- Treat omitted published ports distinctly from port `0`.
- Treat unknown protocols, hostnames, and unresolved interpolation as explicit unsupported or
  indeterminate states.
- Never infer an override stack during automatic discovery.

## Mutation Standards

- Read-only is the default mode.
- Fix eligibility is a first-class decision with a machine-readable refusal reason.
- Build and validate the whole edit plan before touching any original file.
- Back up by default, atomically replace, and verify after commit.
- Refuse edits when source hashes or metadata changed between scan and commit.
- Preserve permissions and line endings where practical.
- Limit source changes to the intended scalar nodes. A fix must not reserialize or reformat the
  entire document.
- Roll back all files when any file in the transaction fails.

## CLI And Output

- stdout contains requested results; stderr contains diagnostics.
- JSON mode emits one valid JSON document and no progress chatter.
- Exit codes follow `README.md`.
- CLI errors name the invalid argument and provide a corrective action.
- Public JSON fields use `snake_case`; Go identifiers use idiomatic names.
- Breaking JSON changes require a schema-version increment and migration notes.

## Error Categories

Use typed or sentinel categories for:

- invalid configuration
- discovery failure
- Compose load/interpolation/merge failure
- unsupported or indeterminate construct
- collision present
- allocation exhausted
- fix refused
- transaction/rollback/verification failure
- internal invariant violation

Only `internal/app` maps these categories to process exit codes.

## Logging And Privacy

- Default output is quiet beyond the requested report.
- A future verbose mode may describe discovery decisions but must never print environment values.
- Error messages may name environment variable keys, not their values.
- Do not add telemetry, analytics, or network calls.

## Quality Gates

Required locally and in CI:

```text
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
staticcheck ./...
govulncheck ./...
```

Also require:

- minimum meaningful coverage thresholds for collision, allocation, and fix packages
- no flaky tests
- golden-file review for JSON changes
- fixture additions for every supported Compose behavior
- successful builds for all release targets

## Documentation And Commits

- Document exported APIs and non-obvious invariants.
- Comments explain reasons and safety constraints, not syntax.
- Keep changes scoped; behavior changes include tests and documentation.
- Do not commit generated binaries, coverage artifacts, temporary files, backups, or fixture
  outputs outside approved golden files.
