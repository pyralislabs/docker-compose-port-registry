# Testing Strategy

## Principles

The main risk is false confidence: Compose syntax may parse while the analyzed configuration differs
from what Compose would run. Tests therefore emphasize fixtures, compatibility checks against the
pinned Compose loader, deterministic reports, and destructive-path safety.

Every supported behavior requires a fixture. Unsupported behavior requires a fixture proving the
warning or refusal.

## Test Layers

### Unit tests

- interval validation and intersection
- host-scope canonicalization and overlap
- protocol separation
- canonical comparators and stable IDs
- allocator winner selection, range preservation, exhaustion
- fix eligibility and refusal reasons
- exit-code mapping and JSON schema serialization

### Fixture integration tests

Load complete directory trees from `testdata/fixtures`, run the real scan pipeline, and compare
normalized results with golden JSON under `testdata/golden`.

### Mutation transaction tests

Copy fixtures to a temporary directory, run fixes, verify exact source diffs, rescan, then exercise
backup, rollback, stale-file, permission, and failure paths.

### CLI tests

Build and invoke the binary to verify stdout/stderr separation, exit codes, default command
behavior, repeated flags, invalid flags, JSON validity, and deterministic output.

### Fuzz/property tests

- port short-syntax tokenizer/parser boundaries
- intervals and collision grouping
- host-IP parsing
- YAML input through normalization
- invariant: allocator never returns an overlapping or out-of-range candidate
- invariant: a successful supported fix removes its planned collision without changing unrelated
  normalized bindings

## Required Fixture Matrix

### Discovery and projects

| Fixture | Expected behavior |
| --- | --- |
| each conventional base filename | discovered |
| nested projects | independent projects |
| two base filenames in one directory | ambiguity warning/error |
| `.git`, vendor, dependency trees | skipped |
| excluded glob | skipped |
| directory symlink loop | not followed |
| duplicate canonical explicit files | deduplicated or rejected clearly |
| unreadable file/directory | deterministic diagnostic |
| filenames with spaces and Unicode | handled without corruption |

### Compose syntax and loading

| Fixture | Expected behavior |
| --- | --- |
| short syntax `8080:80` | normalized TCP wildcard bind |
| quoted and unquoted YAML scalars | normalized consistently |
| host IP short syntax | specific scope retained |
| `/tcp`, `/udp`, mixed protocols | protocols retained |
| single and paired ranges | intervals validated |
| long syntax with string/integer published | normalized |
| omitted published port | excluded from static conflicts |
| `expose` only | ignored |
| YAML anchor/alias | loader-resolved and provenance/refusal recorded |
| extension field | accepted when loader resolves it |
| explicit base plus override | effective merged ports match Compose behavior |
| override reset/replace semantics | effective bindings correct |
| inactive/active profiles | included set correct |
| `extends` | resolved or explicitly unsupported |
| `include` | resolved only when pinned behavior is proven |
| Swarm port mode | warning plus documented normalization |
| malformed YAML | load failure |
| invalid port/range/protocol | clear failure |

### Interpolation

| Fixture | Expected behavior |
| --- | --- |
| process env value | highest-precedence value used |
| repeated `--env-file` | later explicit file wins |
| default `.env` | used only without explicit env file |
| default and required expressions | Compose-compatible result |
| escaped dollar | remains literal |
| service `env_file` | does not affect model interpolation |
| unresolved required variable | load failure |
| unresolved port value | indeterminate warning/error |
| secret-looking env value | never present in report |

### Collision semantics

| Fixture | Expected behavior |
| --- | --- |
| same project/service duplicate | collision |
| two projects same wildcard TCP port | collision |
| same port TCP vs UDP | no collision |
| same port on different specific IPv4s | no collision |
| IPv4 wildcard vs specific IPv4 | collision |
| IPv6 wildcard vs specific IPv6 | collision |
| IPv4 wildcard vs IPv6 wildcard | no collision by default |
| unspecified wildcard vs any specific | collision |
| equal canonical IPv6 spellings | collision |
| hostname host IP | indeterminate |
| range vs scalar intersection | exact overlap reported |
| range vs range partial/full overlap | exact overlap reported |
| adjacent non-overlapping ranges | no collision |
| target ports equal, published differ | no collision |
| three-way collision | one stable grouped finding |

### Allocation

| Fixture | Expected behavior |
| --- | --- |
| empty range | first port selected |
| occupied low ports | first gap selected |
| different protocol occupancy | candidate remains available |
| different specific IP occupancy | candidate remains available |
| wildcard occupancy | compatible specifics blocked |
| range binding | equal-width candidate selected |
| multiple collisions | canonical winner and stable sequential assignments |
| allocation range boundary | exact fit accepted |
| exhausted range | no suggestion and explicit diagnostic |
| reversed/invalid configured range | CLI/config error |
| repeated run | byte-identical plan |

### Fix safety

| Fixture | Expected behavior |
| --- | --- |
| supported literal short syntax | scalar-only edit succeeds |
| multiple supported files | all-or-nothing transaction |
| long syntax | refused in v1 |
| range | refused in v1 |
| interpolated published port | refused |
| override/anchor/alias origin | refused |
| ambiguous duplicate scalar | refused |
| existing backup path | refused |
| `--fix --dry-run` | validated plan, no original or persistent backup changes |
| `--dry-run` without `--fix` | CLI/config error |
| `--no-backup` without `--yes` | refused |
| read-only target | refused |
| file changes after plan | refused |
| temporary write failure | originals unchanged |
| parse/verification failure | rollback succeeds |
| backup restore failure | loud partial-failure report |
| CRLF input | line endings preserved where supported |
| permissions | preserved |
| symlink swap attempt | refused where detectable |

## Compose Compatibility Tests

Maintain a small suite that compares the normalized effective services/ports produced by the
project loader with the pinned Compose library's canonical model. Before upgrading `compose-go`,
run all fixtures and manually review any golden changes.

Optional CI may compare selected fixtures with `docker compose config --format json` when Docker
Compose is available. This is a compatibility signal, not a required runtime dependency.

## Cross-Platform Matrix

CI must test:

- Linux: full suite, race detector, fuzz smoke tests
- macOS: unit/integration tests and transaction behavior
- Windows: unit/integration tests, path handling, backup and atomic replacement behavior

Release-build verification targets:

- Linux `amd64`, `arm64`
- macOS `amd64`, `arm64`
- Windows `amd64`, `arm64`

## Coverage And Performance

Target at least 90% statement coverage for `collision`, `allocate`, and `fix`; coverage alone does
not replace fixture breadth.

Performance acceptance fixture:

- scan 10,000 Compose files / 100,000 normalized bindings within an agreed CI budget
- bounded memory growth
- deterministic output under randomized filesystem creation order

Set exact time and memory budgets after the first benchmark on CI hardware, then prevent material
regressions.

## Release Acceptance

A release candidate must pass all quality gates, all platform tests, a clean install/run smoke test
for every artifact, JSON schema/golden checks, vulnerability scanning, and checksum verification.
