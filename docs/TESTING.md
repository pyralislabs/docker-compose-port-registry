# Testing Strategy

## Principles

The main risk is false confidence: Compose syntax may parse while the analyzed configuration differs
from what Compose would run. Tests therefore emphasize fixtures, compatibility checks against the
pinned Compose loader, deterministic reports, and destructive-path safety.

Every supported behavior requires test coverage. Unsupported behavior requires a test proving the
warning or refusal.

## Test Layers

### Unit tests ✓

- interval validation and intersection
- host-scope canonicalization and overlap
- protocol separation
- canonical comparators and stable IDs
- allocator winner selection, range preservation, exhaustion
- fix eligibility and refusal reasons
- exit-code mapping and JSON schema serialization

### Integration tests ✓

Real Compose files are written to temporary directories, the full scan pipeline runs against them,
and results are asserted for correctness. These cover:

- discovery with conventional filenames, excludes, ambiguity detection
- Compose loading with short syntax, host IP, protocols, ranges, interpolation
- collision detection across projects, protocols, host scopes, ranges
- allocation with occupancy, exhaustion, protocol independence
- fix application, backup, rollback, dry-run, staleness detection

### Mutation transaction tests ✓

Copy fixtures to a temporary directory, run fixes, verify exact source diffs, rescan, then exercise
backup, rollback, stale-file, permission, and failure paths.

### CLI tests ✓

Tests construct an `App` with in-memory stdout/stderr, run the full pipeline, and verify exit codes,
output content, and JSON validity. These cover:

- exit code 0 (no collisions), 1 (collisions), 2 (invalid config), 3 (failure), 4 (fix refused)
- version output
- text and JSON output formats
- `--fix --dry-run` output
- `--no-backup` without `--yes`
- `--file` explicit project
- `--exclude` filtering
- combined `--file` and roots validation

### Fixture golden tests (planned)

Load complete directory trees from `testdata/fixtures`, run the real scan pipeline, and compare
normalized results with golden JSON under `testdata/golden`. These will provide byte-stable
regression coverage.

### Fuzz/property tests (planned)

- port short-syntax tokenizer/parser boundaries
- intervals and collision grouping
- host-IP parsing
- YAML input through normalization
- invariant: allocator never returns an overlapping or out-of-range candidate
- invariant: a successful supported fix removes its planned collision without changing unrelated
  normalized bindings

## Required Fixture Matrix

### Discovery and projects ✓

| Fixture | Expected behavior | Status |
| --- | --- | --- |
| each conventional base filename | discovered | ✓ tested |
| nested projects | independent projects | ✓ tested |
| two base filenames in one directory | ambiguity warning/error | ✓ tested |
| `.git`, vendor, dependency trees | skipped | ✓ tested |
| excluded glob | skipped | ✓ tested |
| directory symlink loop | not followed | not yet tested |
| duplicate canonical explicit files | deduplicated or rejected clearly | not yet tested |
| unreadable file/directory | deterministic diagnostic | ✓ tested |
| filenames with spaces and Unicode | handled | not yet tested |

### Compose syntax and loading ✓

| Fixture | Expected behavior | Status |
| --- | --- | --- |
| short syntax `8080:80` | normalized TCP wildcard bind | ✓ tested |
| quoted and unquoted YAML scalars | normalized consistently | ✓ tested |
| host IP short syntax | specific scope retained | ✓ tested |
| `/tcp`, `/udp`, mixed protocols | protocols retained | ✓ tested |
| single and paired ranges | intervals validated | ✓ tested |
| long syntax with string/integer published | normalized | not yet tested |
| omitted published port | excluded from static conflicts | ✓ tested |
| `expose` only | ignored | not yet tested |
| YAML anchor/alias | loader-resolved and provenance recorded | not yet tested |
| extension field | accepted when loader resolves it | not yet tested |
| explicit base plus override | effective merged ports match Compose | ✓ tested |
| override reset/replace semantics | effective bindings correct | not yet tested |
| inactive/active profiles | included set correct | not yet tested |
| `extends` | resolved or explicitly unsupported | not yet tested |
| `include` | resolved only when pinned behavior proven | not yet tested |
| Swarm port mode | warning plus documented normalization | not yet tested |
| malformed YAML | load failure | ✓ tested |
| invalid port/range/protocol | clear failure | ✓ tested |

### Interpolation ✓

| Fixture | Expected behavior | Status |
| --- | --- | --- |
| process env value | highest-precedence value used | ✓ tested |
| repeated `--env-file` | later explicit file wins | ✓ tested |
| default `.env` | used only without explicit env file | ✓ tested |
| default and required expressions | Compose-compatible result | not yet tested |
| escaped dollar | remains literal | not yet tested |
| service `env_file` | does not affect model interpolation | not yet tested |
| unresolved required variable | load failure | not yet tested |
| unresolved port value | indeterminate warning/error | not yet tested |
| secret-looking env value | never present in report | ✓ by design |

### Collision semantics ✓

| Fixture | Expected behavior | Status |
| --- | --- | --- |
| same project/service duplicate | collision | ✓ tested |
| two projects same wildcard TCP port | collision | ✓ tested |
| same port TCP vs UDP | no collision | ✓ tested |
| same port on different specific IPv4s | no collision | ✓ tested |
| IPv4 wildcard vs specific IPv4 | collision | ✓ tested |
| IPv6 wildcard vs specific IPv6 | collision | ✓ tested |
| IPv4 wildcard vs IPv6 wildcard | no collision by default | ✓ tested |
| unspecified wildcard vs any specific | collision | ✓ tested |
| equal canonical IPv6 spellings | collision | ✓ tested |
| hostname host IP | indeterminate | ✓ tested |
| range vs scalar intersection | exact overlap reported | ✓ tested |
| range vs range partial/full overlap | exact overlap reported | ✓ tested |
| adjacent non-overlapping ranges | no collision | ✓ tested |
| target ports equal, published differ | no collision | ✓ tested |
| three-way collision | one stable grouped finding | ✓ tested |

### Allocation ✓

| Fixture | Expected behavior | Status |
| --- | --- | --- |
| empty range | first port selected | ✓ tested |
| occupied low ports | first gap selected | ✓ tested |
| different protocol occupancy | candidate remains available | ✓ tested |
| different specific IP occupancy | candidate remains available | ✓ tested |
| wildcard occupancy | compatible specifics blocked | ✓ tested |
| range binding | equal-width candidate selected | ✓ tested |
| multiple collisions | canonical winner and stable sequential | ✓ tested |
| allocation range boundary | exact fit accepted | ✓ tested |
| exhausted range | no suggestion, explicit diagnostic | ✓ tested |
| reversed/invalid configured range | CLI/config error | ✓ tested |
| repeated run | byte-identical plan | ✓ by design |

### Fix safety ✓

| Fixture | Expected behavior | Status |
| --- | --- | --- |
| supported literal short syntax | scalar-only edit succeeds | ✓ tested |
| multiple supported files | all-or-nothing transaction | not yet tested |
| long syntax | refused in v1 | ✓ tested |
| range | refused in v1 | ✓ tested |
| interpolated published port | refused | ✓ tested |
| override/anchor/alias origin | refused | ✓ tested |
| ambiguous duplicate scalar | refused | not yet tested |
| existing backup path | refused | ✓ tested |
| `--fix --dry-run` | validated plan, no changes | ✓ tested |
| `--dry-run` without `--fix` | CLI/config error | ✓ tested |
| `--no-backup` without `--yes` | refused | ✓ tested |
| read-only target | refused | ✓ tested |
| file changes after plan | refused | not yet tested |
| temporary write failure | originals unchanged | not yet tested |
| parse/verification failure | rollback succeeds | ✓ tested |
| backup restore failure | loud partial-failure report | not yet tested |
| CRLF input | line endings preserved | not yet tested |
| permissions | preserved | ✓ tested |
| symlink swap attempt | refused where detectable | not yet tested |

## Coverage And Performance

Current statement coverage target for `collision`, `allocate`, and `fix` packages is >85%.

Performance acceptance criteria for future benchmarks:

- scan 10,000 Compose files / 100,000 normalized bindings within an agreed CI budget
- bounded memory growth
- deterministic output under randomized filesystem creation order

## Cross-Platform Matrix

CI should eventually test:

- Linux: full suite, race detector, fuzz smoke tests
- macOS: unit/integration tests and transaction behavior
- Windows: unit/integration tests, path handling, backup and atomic replacement

Release-build targets:

- Linux `amd64`, `arm64`
- macOS `amd64`, `arm64`
- Windows `amd64`, `arm64`

## Release Acceptance

A release candidate must pass all quality gates, a clean install/run smoke test for every artifact,
JSON schema/golden checks, vulnerability scanning, and checksum verification.
