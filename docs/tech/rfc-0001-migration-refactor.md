# RFC-0001: Migration System Refactor

## Problem

The current Oracle migration system has fundamental DX and correctness issues:

1. **Two-phase workflow**: Developer must run `oracle migrate generate` to create a
   baseline snapshot BEFORE making schema changes, then run it again after. Forgetting
   step 1 means no migration is generated.
2. **Broken cross-type dependencies**: Snapshot packages filter colocated types by
   namespace + output path, missing types the migrated struct actually references.
   Snapshots don't compile.
3. **Disconnected state**: `schemas/.snapshots/` is a separate artifact from generated
   code. Gets out of sync.
4. **Import path bugs**: `repoRoot` passed as `""` in several places.

## Design

### Single command

`oracle sync` generates types, codecs, and migrations in one pass. The `go/migrate`
plugin handles its own state internally. No separate `oracle migrate generate` step
required for normal workflow.

### State tracking via `state.json`

Each migratable type's output directory gets a `migrations/` folder. The latest version
directory contains `state.json` — field fingerprints for every type reachable from the
migrated struct.

```json
{
  "schemas.schematic.Schematic": [
    ["key", "schemas.schematic.Key"],
    ["name", "string"],
    ["data", "json"],
    ["snapshot", "bool"]
  ],
  "schemas.schematic.Key": {"alias": "uuid"},
  "schemas.spatial.XY": [
    ["x", "float64"],
    ["y", "float64"]
  ]
}
```

Structs store `[field_name, canonical_typeref_string]` pairs. Aliases store
`{"alias": "target"}`. Distincts store `{"distinct": "base"}`. Enums store
`{"enum": ["value1", "value2"]}`.

### Plugin logic

When `go/migrate` runs, for each migratable type:

1. Compute its Go output path (e.g., `core/pkg/service/schematic`)
2. Scan `{goPath}/migrations/` for the latest version directory
3. If no `state.json` exists → initial generation
4. If `state.json` exists → compute current fingerprints, compare
5. If nothing changed → emit nothing
6. If something changed → generate migration

The plugin reads `state.json` from disk via `req.RepoRoot`. No `OldResolutions`. No
`schemas/.snapshots/`. The plugin is self-contained.

### Initial generation (no state.json found)

Output:

```
{goPath}/
  migrate.gen.go                    # SchematicMigrations() — just CodecTransition
  migrations/
    v1/
      v1.gen.go                     # Frozen type snapshot (self-contained)
      codec.gen.go                  # Frozen binary codec
      state.json                    # Field fingerprints for diffing
```

### Diff generation (state.json exists, changes detected)

1. Load `state.json` from latest version (e.g., v1)
2. Compute current fingerprints from resolution table
3. Compare — find which types changed
4. `deps.Build(currentResolutions).AffectedEntries(changedTypes)` to find affected
   migratable entries
5. Classify each as skeleton (direct change) or propagation (transitive dependency
   changed)
6. Compute field-level diff from old fingerprints vs current

Output:

```
{goPath}/
  migrate.gen.go                    # Updated — adds TypedMigration v1→v2
  migrations/
    v1/                             # FROZEN. Never touched.
      v1.gen.go
      codec.gen.go
      state.json
    v2/
      v2.gen.go                     # Frozen snapshot of NEW types
      codec.gen.go                  # Frozen codec for new types
      auto.gen.go                   # AutoSchematic — field mapping with TODOs
      migrate.go                    # PostSchematic — stub for developer
      state.json                    # Current fingerprints for next diff
```

### Reachability-based snapshots

Every snapshot includes ALL types reachable from the migrated struct via BFS: field
types, extends clauses, alias targets, distinct bases — recursively. No namespace or
output path filtering. If `Schematic` references `Viewport` which references `XY` from
`schemas/spatial.oracle`, all three end up in the snapshot.

The same BFS determines what goes into `state.json`.

### Diff without old resolutions

```
old fingerprints (state.json)  vs  current fingerprints (resolution table)
         |
   changed qualified type names
         |
   deps.Build(current).AffectedEntries(changed)
         |
   affected migratable entries (skeleton or propagation)
         |
   field-level diff per entry (for auto.gen.go)
```

### Generated migrate.gen.go

```go
func SchematicMigrations(codec gorp.Codec[Schematic]) []gorp.Migration {
    return []gorp.Migration{
        // Version migrations in chronological order
        gorp.NewTypedMigration[v1.Schematic, v2.Schematic](
            "v1_to_v2",
            v1.SchematicCodec, v2.SchematicCodec,
            v2.AutoSchematic, v2.PostSchematic,
        ),
        // CodecTransition always last
        gorp.NewCodecTransition[v1.Key, v1.Schematic](
            "msgpack_to_binary",
            v1.SchematicCodec,
        ),
    }
}
```

Versions discovered by scanning `migrations/v*/` directories. Chained in order.
CodecTransition always references v1 types and comes last.

### oracle migrate check (CI)

Compares current fingerprints against latest `state.json`. If different, schemas changed
without a migration.

---

## Implementation

### What gets deleted

- `schemas/.snapshots/` directory
- `snapshot.Create()`, `snapshot.Check()`, `snapshot.LatestVersion()` — or rewritten
- `analyzer.SnapshotFileLoader`
- `loadSnapshotResolutions()` helper in `generate.go`
- `generateOpts.OldResolutions` and `generateOpts.SnapshotVersion`
- Passing old resolutions through the plugin request

### What gets built

| Component | File | Purpose |
|---|---|---|
| state.json serialization | `oracle/plugin/go/migrate/state.go` | Read/write fingerprints |
| `collectReachableTypes` | `oracle/plugin/go/migrate/reachable.go` | BFS from migrated struct |
| State-based diff | `oracle/plugin/go/migrate/migrate.go` | Diff state.json vs current |
| Rewritten `Generate()` | `oracle/plugin/go/migrate/migrate.go` | Self-contained, reads state.json from disk |
| Rewritten `migrate check` | `oracle/cmd/migrate.go` | Fingerprint comparison |

### What stays the same

- `gorp/migrate.go` runtime (Migration interface, NewTypedMigration, NewCodecTransition,
  NewRawMigration)
- `diff.FieldDiff`, `diff.TypeDiff`, `diff.ChangeKind` types
- `deps.Build()` and `AffectedEntries()`
- Templates (v1Template, autoTemplate, postTemplate, migrateTemplate)
- `gomarshal.GenerateCodec` for codec generation
- Service-side registration pattern (`SchematicMigrations(cfg.Codec)`)
- Ranger's manual migration pattern (append custom migration)

---

## Detailed Steps

### Step 1: state.json format and serialization

**New file**: `oracle/plugin/go/migrate/state.go`

Define the state format:

```go
// TypeFingerprint stores the shape of a single type for diff comparison.
type TypeFingerprint struct {
    // For structs: list of (field_name, canonical_typeref) pairs.
    Fields [][2]string `json:"fields,omitempty"`
    // For aliases: the target type canonical string.
    Alias string `json:"alias,omitempty"`
    // For distinct types: the base type canonical string.
    Distinct string `json:"distinct,omitempty"`
    // For enums: the list of value names.
    Enum []string `json:"enum,omitempty"`
}

// MigrationState is the content of state.json.
type MigrationState struct {
    // Types maps qualified type names to their fingerprints.
    Types map[string]TypeFingerprint `json:"types"`
}
```

Functions:
- `writeMigrationState(path string, state MigrationState) ([]byte, error)` — returns
  JSON bytes (plugin returns files, doesn't write directly)
- `readMigrationState(repoRoot, goPath string, version int) (MigrationState, error)` —
  reads from disk
- `computeFingerprints(entry resolution.Type, table *resolution.Table) MigrationState` —
  computes fingerprints for the entry + all reachable types

### Step 2: Reachability-based type collection

**New file**: `oracle/plugin/go/migrate/reachable.go`

```go
func collectReachableTypes(
    entry resolution.Type,
    table *resolution.Table,
) []resolution.Type
```

BFS algorithm:
1. Seed queue with all TypeRefs from entry's fields (via `UnifiedFields`) and extends
2. For each ref:
   - Skip primitives, `Array`, `Map`, type params (but follow their type args)
   - Resolve from table (try qualified name, then namespace-local lookup)
   - If not visited: mark visited, add to result
   - If struct: enqueue field types + extends
   - If alias: enqueue target
   - If distinct: enqueue base
   - If enum: nothing to follow
3. Return `table.TopologicalSort(result)`

Exclude the entry itself and its key field's type (they're rendered separately by the
v1 template).

This function is used in two places:
- `generateV1File` — to determine which types go in the snapshot package
- `computeFingerprints` — to determine which types go in state.json

### Step 3: State-based diff computation

**In**: `oracle/plugin/go/migrate/migrate.go`

New function:

```go
func diffFromState(
    oldState MigrationState,
    currentTable *resolution.Table,
) (changedTypes []string, entryDiffs map[string]*diff.TypeDiff)
```

Algorithm:
1. Compute current fingerprints for ALL struct types in the resolution table
2. Compare each type in `oldState.Types` against current:
   - Type in old but not current → changed
   - Type in current but not old → changed
   - Fields/alias/distinct/enum differ → changed
3. For changed struct types that are migratable, compute field-level `diff.TypeDiff`
   (FieldAdded, FieldRemoved, FieldTypeChanged, FieldUnchanged)
4. Return both the list of changed type names AND the per-entry field diffs

### Step 4: Rewrite `Generate()` entry point

**In**: `oracle/plugin/go/migrate/migrate.go`

New flow:

```go
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
    resp := &plugin.Response{}

    // Group migratable entries by goPath.
    entriesByPath := groupMigratableEntries(req)

    for goPath, entries := range entriesByPath {
        latestVersion := scanLatestVersion(req.RepoRoot, goPath)

        if latestVersion == 0 {
            // No migrations exist. Initial generation.
            files, err := p.generateInitial(goPath, entries, req)
            // append to resp.Files...
        } else {
            // Load state from latest version.
            oldState, err := readMigrationState(req.RepoRoot, goPath, latestVersion)
            // Compute diff.
            changedTypes, entryDiffs := diffFromState(oldState, req.Resolutions)
            if len(changedTypes) == 0 {
                continue // Nothing changed for this path.
            }
            // Generate new version.
            files, err := p.generateMigration(
                goPath, entries, req, latestVersion, changedTypes, entryDiffs,
            )
            // append to resp.Files...
        }
    }
    return resp, nil
}
```

`scanLatestVersion` reads `{repoRoot}/{goPath}/migrations/` and finds the highest `vN`
directory containing a `state.json`.

### Step 5: Rewrite `generateInitial`

Generates for a single goPath:
- `migrations/v1/v1.gen.go` — frozen snapshot using reachability
- `migrations/v1/codec.gen.go` — frozen codec (NEW: wasn't generated before)
- `migrations/v1/state.json` — fingerprints
- `migrate.gen.go` — just CodecTransition

Uses `collectReachableTypes` for snapshot and `computeFingerprints` for state.

### Step 6: Implement `generateMigration` (replaces `generateWithDiff`)

Generates for a single goPath when changes detected:
- `migrations/vN/vN.gen.go` — snapshot from CURRENT resolutions + reachability
- `migrations/vN/codec.gen.go` — codec from current resolutions
- `migrations/vN/auto.gen.go` — field mapping from diff
- `migrations/vN/migrate.go` — PostMigrate stub
- `migrations/vN/state.json` — current fingerprints
- `migrate.gen.go` — updated with full migration chain

Key difference from old `generateWithDiff`: does NOT regenerate v1 or any previous
version. Only generates new version directory + updated migrate.gen.go.

### Step 7: Update `generateV1File` to use reachability

Replace the three filtering loops (aliases filtered by namespace+path, distincts
filtered by namespace+path, structs filtered by namespace+path) with:

```go
reachable := collectReachableTypes(entry.Type, table)
// Partition by form type for template.
var aliases []colocatedTypeDef
var distincts []colocatedTypeDef
var enums []colocatedEnum
var structs []colocatedStruct
for _, typ := range reachable {
    switch typ.Form.(type) {
    case resolution.AliasForm: ...
    case resolution.DistinctForm: ...
    case resolution.EnumForm: ...
    case resolution.StructForm: ...
    }
}
```

The v1Template stays the same. Only the data fed into it changes.

### Step 8: Update `generateMigrateFile` for version chain discovery

Instead of receiving `versionMigrations` from the caller, the function scans
`{goPath}/migrations/v*/` to discover ALL versions. It constructs the full chain:

```
v1 → v2: TypedMigration[v1.T, v2.T](v2.Auto, v2.Post)
v2 → v3: TypedMigration[v2.T, v3.T](v3.Auto, v3.Post)
...
CodecTransition[v1.Key, v1.T](v1.Codec)  // always last, always v1
```

This makes migrate.gen.go generation idempotent — it always produces the correct chain
based on what version directories exist.

### Step 9: Simplify `oracle sync`

**File**: `oracle/cmd/sync.go`

Remove:
- `loadSnapshotResolutions()` call
- Snapshot creation logic
- `generateOpts` passing

`oracle sync` just: analyze schemas → run plugins → sync files. The go/migrate plugin
handles everything internally.

### Step 10: Simplify `oracle migrate generate`

**File**: `oracle/cmd/migrate.go`

Same simplification. Becomes: analyze schemas → run go/migrate plugin → sync files.
No snapshot management. Kept as convenience for migration-only generation.

### Step 11: Rewrite `oracle migrate check`

**File**: `oracle/cmd/migrate.go`

New logic:
1. Analyze current schemas
2. For each migratable type, compute current fingerprints
3. Find latest `state.json` in `{goPath}/migrations/`
4. Compare fingerprints
5. If different → error: "schemas changed without migration, run `oracle sync`"
6. If no state.json exists for a migratable type → error: "no migration baseline"

### Step 12: Delete old infrastructure

- Delete `oracle/snapshot/` package (or gut it)
- Delete `analyzer.SnapshotFileLoader` and `analyzer.NewSnapshotFileLoader`
- Remove `OldResolutions` and `SnapshotVersion` from `plugin.Request`
- Remove `generateOpts` struct from `generate.go` (or simplify it)
- Remove `schemas/.snapshots/` from `.gitignore` / repo

### Step 13: Tests

**state.json round-trip**: Write state, read it back, verify identical.

**Reachability**:
- Struct references type from different namespace → included
- Struct extends type from different file → included
- Transitive: A → B → C, all included
- Primitives, arrays, maps not included (their type args are followed)

**Diff from state**:
- Field added → detected
- Field removed → detected
- Field type changed → detected
- Unchanged → not flagged
- Non-migratable type changes → propagation to dependent migratable entries

**Initial generation**:
- Produces v1.gen.go, codec.gen.go, state.json, migrate.gen.go
- v1.gen.go is self-contained (all reachable types present)

**Diff generation**:
- Produces v2 directory, updated migrate.gen.go
- Does NOT modify v1 directory
- auto.gen.go has correct field classifications
- migrate.gen.go chains v1→v2 correctly

**No-change detection**:
- Running with unchanged schemas produces no output files

**Import paths**:
- All generated import paths are correct with repoRoot

---

## Design Decisions

### Non-struct type changes (aliases, distincts, enums)

The diff compares ALL entries in state.json — structs, aliases, distincts, enums. If an
alias changes target (`Key = uuid` → `Key = string`), the alias fingerprint changes.
The changed type name feeds into `deps.AffectedEntries`, which finds migratable structs
that reference `Key` and gives them propagation migrations. The auto.gen.go shows all
fields "unchanged" but the codec is regenerated with the correct type. This is correct
because the binary representation changes even though the struct fields didn't.

### Multiple migratable types in same goPath

Single state.json per version directory, single version directory per goPath. The
state.json contains fingerprints for the union of all reachable types across all
migratable entries in that path. v1.gen.go already handles multiple entries (the
template loops over entries). migrate.gen.go generates a separate
`{Name}Migrations()` function per entry.

### migrate.go preservation

The plugin must NOT emit `migrate.go` if it already exists on disk. Before adding
`migrate.go` to the response files, check `os.Stat(filepath.Join(req.RepoRoot, path))`.
If the file exists, skip it. This prevents overwriting developer-written PostMigrate
logic on re-runs. All other files in the version directory (`.gen.go`, `state.json`) are
safe to overwrite because they're generated and deterministic.

### Version directory detection

`scanLatestVersion` reads `{repoRoot}/{goPath}/migrations/`, finds the highest `vN`
directory that contains a `state.json`. The `state.json` presence is the marker for "this
version was fully generated." Directories without it are ignored — they represent
interrupted or partial generation.

### Cross-goPath type changes

Works by design. If `XY` (output `x/go/spatial`) changes and `Schematic` (output
`core/pkg/service/schematic`) references it, then `Schematic`'s state.json includes
`XY`'s fingerprint. The diff detects the change. `XY` itself isn't migratable so nothing
happens for `x/go/spatial`. `Schematic` gets a propagation migration with a regenerated
codec that picks up the new `XY` definition.

### Enum/alias-only changes trigger propagation

If an enum value is added, the enum fingerprint in state.json changes. This triggers a
propagation migration for any migratable struct referencing that enum. The auto.gen.go
shows all fields "unchanged" and the new codec handles the updated enum. This is correct
— the binary encoding of the enum changed, so stored data needs re-encoding. Skipping
this would leave stale codecs that can't decode new enum values.
