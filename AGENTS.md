# AGENTS.md

Instructions for agents working in `git-projects/docker-compose-port-registry/`.

## Read First

1. `bootstrap.md`
2. `docs/ARCHITECTURE.md`
3. `docs/CODE_STANDARDS.md`
4. `docs/TESTING.md`
5. `docs/ROADMAP.md`

This folder is the complete ownership boundary for this project. Do not modify files outside it
unless the user explicitly expands scope.

## Product Rules

- The product is a Go single-binary CLI.
- Prefer Compose Specification behavior through `compose-go`; do not invent approximate Compose
  semantics when the library can provide them.
- Detection is broad; mutation is deliberately narrow and must fail closed.
- Never silently choose values for unresolved interpolation.
- Never rewrite a Compose file unless the requested fix is unambiguous, backed up by default,
  atomically installed, and verified by a fresh scan.
- Output ordering and allocation decisions must be deterministic across runs and platforms.
- JSON output is a public, versioned contract.
- Keep v1 focused on static Compose-file analysis. Do not add daemon behavior or runtime Docker
  inspection without an approved architecture change.

## Implementation Rules

- Keep packages aligned with the tree in `bootstrap.md`.
- Keep normalized domain types in `internal/model`; Compose-library types must not leak through the
  application.
- Preserve source provenance for every binding: project, service, source file, line/column where
  available, and original syntax.
- Represent ports and ranges as validated integer intervals, never ad hoc strings.
- Treat host IP overlap explicitly, including wildcard IPv4 and IPv6 binds.
- Use typed errors and map them to the documented exit codes at the application boundary.
- Avoid global mutable state and hidden environment reads.
- Add fixtures for every new Compose syntax or collision behavior.
- Do not weaken mutation safeguards to make a test pass.

## Required Checks Before Completion

```text
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
```

When implemented, CI must also run linting, fixture/golden tests, vulnerability scanning, and
cross-platform build verification.

## Documentation Discipline

Update the relevant documents when behavior changes:

- CLI or JSON contract: `README.md`, `bootstrap.md`, and architecture
- Compose or collision semantics: architecture and testing matrix
- Engineering policy: code standards
- Scope, sequencing, or acceptance criteria: roadmap

Do not claim support for behavior that lacks fixtures.
