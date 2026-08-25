# 60 Core doctor command

- **Author**: Emiliano Bonilla
- **Date**: 2026-08-25
- **Related**:
  [RFC 0024 - Cesium domain index data structure change](0024-cesium-v2.md),
  [RFC 0035 - Cesium variable-length storage](0035-cesium-variable-length-storage.md)

## 0 Summary

`synnax doctor` is a new subcommand of the Core binary that inspects a data directory
and reports on the health of stored data. It produces two kinds of output in one run:
statistics (per-channel data quantities, domain distributions, garbage and tombstone
totals, keyspace breakdowns) and findings (violations of storage invariants, referential
orphans in the KV layer, and aberrations such as overlapping or tiny domains, near-zero
gaps, and implausible timestamps).

The command is offline and strictly read-only. It never opens storage through the normal
engine paths, which mutate on open, and it never repairs anything. With the Core stopped
it inspects everything; with the Core running it still delivers the Cesium report by
decoding files directly and states that KV checks were skipped.

## 1 Vocabulary

- **Domain**: A contiguous time-bounded region of samples in a channel's `domain.DB`.
  Domains are sorted and must not overlap (RFC 0024).
- **Pointer**: The on-disk record of a domain: time range, file key, byte offset, and
  byte size. 26 bytes in `index.domain` (`cesium/internal/domain/pointer.go`).
- **Garbage**: Bytes in a data file that no pointer references. Cesium has no explicit
  tombstone records; garbage is the implicit remainder `fileSize - Σ pointer.size` per
  file (`cesium/internal/domain/delete.go`).
- **Check**: A named inspection with a stable identifier, such as
  `cesium.domain-overlap`. A check emits zero or more findings.
- **Finding**: One observed problem: check name, severity, subject (channel, entry,
  file), message, and a remediation hint.
- **Severity**: `error` (an invariant is broken), `warning` (almost always a bug or a
  degradation, but the system tolerates it), `info` (notable, not wrong).
- **Deep check**: A check that reads sample bytes, so its cost scales with stored data
  size rather than with domain or entry counts.

## 2 Motivation

There is no way to see what a Synnax data directory actually contains. The only
stored-data metric the Core exposes is a size gauge, and it is wrong in a specific way:
`cesium.DB.Metrics().DiskSize` sums live bytes only (`cesium/db.go:130`), so it
under-reports real disk usage by exactly the garbage total. Nothing reports domain
counts, domain distributions, gap structure, or how far behind garbage collection is.

Open-time validation is thin. `domain.Open` decodes `index.domain` and sums sizes; it
does not verify sortedness, non-overlap, a non-zero file key, that file keys are within
the allocated counter, or that a pointer's `offset + size` fits inside its data file
(`cesium/internal/domain/db.go:119`). A corrupted index is accepted silently and
surfaces later as wrong reads.

The KV layer accumulates tolerated inconsistencies by design. Ontology queries silently
drop resources whose backing entity is gone
(`core/pkg/service/ontology/retrieve.go:374`), several delete paths leave satellite
entries behind (range aliases, range KV pairs, task configs, user credentials), and a
lost counter key silently resets to zero and re-issues keys (`x/go/kv/counter.go:37`).
None of this is visible until it causes a defect.

Finally, field incidents need a tool. When a deployment shows wrong data, for example
elapsed times written as timestamps that land near the Unix epoch, diagnosis today means
manual spelunking through binary files. The workflow this command serves is: support
says "run `synnax doctor` and send the output", and the output is complete enough to
localize the problem in one round trip.

## 3 Principles

1. **Strictly read-only**: The doctor never writes, and never opens storage through a
   path that writes. `cesium.Open` rewrites `meta.json` through migrations and starts
   the GC ticker (`cesium/open.go:80`); `storage.OpenLayer` ratchets Pebble's format
   version (`core/pkg/storage/layer.go:357`); `aspen.Open` can bootstrap cluster state.
   All three are forbidden. Repair is out of scope entirely.
2. **A sick database must not need a healthy Core**: The command works on the files,
   offline. It degrades gracefully when the Core is running instead of failing.
3. **Format knowledge stays in the owning package**: Cesium file decoding lives in the
   `cesium` module, gorp key decoding in `x/go/gorp`, Pebble access in
   `x/go/kv/pebblekv`. The doctor composes; it does not re-implement formats.
4. **Fail soft, per subject**: A corrupt channel or undecodable entry produces findings
   and the run continues. Exit code 1 is reserved for the tool itself failing, not for
   bad data.
5. **Warnings never break scripts**: Only error-level findings affect the exit code.

## 4 Design

### 4.0 Command surface

```
synnax doctor [flags]
```

- `--data` / `-d`: The data directory, reusing `start.FlagData` and its default.
- `--channels`: Restrict Cesium inspection to a comma-separated list of channel keys or
  names.
- `--skip-deep`: Skip checks that read sample bytes.
- `--skip-kv` / `--skip-ts`: Skip a whole side.
- `--json`: Emit one machine-readable JSON document on stdout instead of text.
- `--verbose` / `-v`: Include info-level findings and per-channel stat tables that the
  default output summarizes.

The default run performs the full inspection, deep checks included. Flags exclude work;
nothing is opt-in. Progress goes to stderr (current phase, channels completed / total,
bytes scanned during deep checks) so stdout stays clean for piping.

Exit codes: `0` clean or warnings only, `1` the tool failed to run, `2` at least one
error-level finding. This is the `pg_amcheck` convention. Warnings deliberately do not
affect the exit code: `brew doctor` exits non-zero on warnings and that behavior is its
most complained-about defect, because it fails CI on harmless findings.

### 4.1 Access model

The two engines admit different read-only strategies:

- **Cesium**: All files are decoded directly with read-only opens. Cesium takes no file
  lock of its own, so this works while a Core is running, with known races: the index
  file is truncated and rewritten in place during persistence
  (`cesium/internal/domain/index_persist.go:48`), and GC renames data files
  (`cesium/internal/domain/delete.go:411`). The inspector tolerates both: a
  non-multiple-of-26 index tail is reported and the decodable prefix is used, and a file
  that vanishes mid-scan is retried once and then reported. Against a running Core the
  output is a consistent-enough snapshot, not a transaction.
- **Pebble**: Opened with Pebble's upstream read-only mode through a new
  `pebblekv.OpenReadOnly`, using the same `FormatMajorVersion` as production
  (`core/pkg/storage/layer.go:376`) and skipping `pebblekv.Migrate`. Pebble acquires its
  directory lock even read-only, so this requires the Core stopped. When the lock is
  held, the doctor prints a plain notice, skips every KV and cross-layer check, and
  continues with Cesium. While the doctor holds the KV open, a Core cannot start against
  the same directory; the run should therefore be short-lived and the notice in the docs
  says so.

The doctor does not touch the storage-layer `LOCK` file and never joins or bootstraps a
cluster. Node identity and membership come from decoding the persisted cluster state at
the `aspen.cluster` key (`aspen/internal/cluster/config.go:102`).

### 4.2 Cesium inspection: the `cesium/inspect` package

A new public package in the `cesium` module. It scans the Cesium root, and for each
channel directory decodes `meta.json` leniently (raw JSON first, so a legacy or future
version is reported rather than migrated or rejected), reads `counter.domain`, decodes
`index.domain` through a read-only decode function exported from
`cesium/internal/domain` (the pointer codec's single home), and stats every data file.
It returns a per-channel report plus findings; it opens nothing read-write.

**Per-channel statistics**:

- Domain count, live bytes, on-disk bytes, garbage bytes and ratio, file count, and the
  number of files past the GC threshold (recomputing the exact production test,
  `tombstoneSize >= GCThreshold * FileSize`, with the configured or default values).
- Sample count: `size / density` summed over pointers for fixed-density channels; a
  length-prefix walk during deep checks for variable-length channels (bytes only when
  deep checks are skipped).
- Time span, domain duration and size distributions (min / p50 / p99 / max), counts of
  tiny domains and of near-zero gaps between domains.

The DB-level roll-up aggregates these and lists the top offenders by garbage, by domain
count, and by tiny-domain count, so the interesting channels surface without reading
per-channel tables.

**Structural checks** (per channel, metadata cost):

| Check                   | Severity | Condition                                       |
| ----------------------- | -------- | ----------------------------------------------- |
| `cesium.index-decode`   | error    | `index.domain` size is not a multiple of 26     |
| `cesium.domain-order`   | error    | pointers are not sorted by start                |
| `cesium.domain-overlap` | error    | a pointer's range overlaps its neighbor         |
| `cesium.domain-bounds`  | error    | a pointer has `start > end`, or `end` exclusive |
|                         |          | semantics are violated                          |
| `cesium.file-key`       | error    | a pointer has `fileKey == 0` or a key above     |
|                         |          | `counter.domain`                                |
| `cesium.file-bounds`    | error    | `offset + size` exceeds the data file's size    |
| `cesium.missing-file`   | error    | a pointer references a file that does not exist |
| `cesium.density-align`  | error    | fixed-density pointer `size % density != 0`     |
| `cesium.meta`           | error    | `meta.json` missing, undecodable, or failing    |
|                         |          | `Channel.Validate`                              |
| `cesium.orphan-file`    | warning  | a data file no pointer references, or an        |
|                         |          | unparseable name in the Cesium root             |
| `cesium.artifacts`      | warning  | leftover `_gc` / `_temp` files or               |
|                         |          | `<key>-DELETE-<rand>` directories from an       |
|                         |          | interrupted GC or delete                        |
| `cesium.ignored`        | warning  | a channel the v2 migration marks ignored: it    |
|                         |          | still consumes disk but is invisible            |
| `cesium.index-ref`      | error    | a data channel whose index channel directory    |
|                         |          | is absent or not an index                       |
| `cesium.garbage`        | warning  | garbage ratio above threshold despite eligible  |
|                         |          | files: GC is not keeping up                     |
| `cesium.tiny-domain`    | warning  | domains below the size / duration floor         |
| `cesium.micro-gap`      | warning  | inter-domain gaps inside the near-zero band:    |
|                         |          | writers are fragmenting what should be one      |
|                         |          | continuous domain                               |
| `cesium.time-bounds`    | warning  | pointer bounds far in the past or future, with  |
|                         |          | a distinct message for the near-epoch band      |
|                         |          | (the elapsed-time-as-timestamp signature)       |

`cesium.time-bounds` applies to every persisted channel, because pointer bounds are
timestamps regardless of the channel's data type. Continuity semantics follow the
production rule: touching bounds (`end == next.start`) are one continuous region
(`cesium/internal/index/domain.go:491`); any positive gap splits it, and gaps inside the
near-zero band are the aberration this check exists to catch.

**Deep checks** (read sample bytes):

| Check                  | Severity | Condition                                       |
| ---------------------- | -------- | ----------------------------------------------- |
| `cesium.index-content` | error    | an index channel's stored timestamps are not    |
|                        |          | strictly increasing within a domain, the first  |
|                        |          | sample differs from the domain start, or a      |
|                        |          | sample falls outside the pointer bounds         |
| `cesium.varlen-walk`   | error    | a variable-length domain's length prefixes do   |
|                        |          | not walk exactly to the domain's end, or a      |
|                        |          | prefix exceeds the remaining bytes (extends the |
|                        |          | existing truncation detection in                |
|                        |          | `cesium/internal/unary/resolver.go:98`)         |

`cesium.index-content` is the check that catches bound-versus-content disagreement,
which today is silent corruption: nothing in the engine ever re-verifies that an index's
samples agree with the pointer bounds derived from them.

### 4.3 KV inspection

**Structural pass** (no service knowledge). The doctor walks the entire keyspace once
and buckets every key: `gorp.<TypeName><key>` entries by longest-match against the known
type names, `gorp.migration.<TypeName>` state, and the known non-gorp keys
(`aspen.cluster`, `--dig/` digests, `ver`, the per-node channel and rack counters,
`sy_task_legacy_key/` staging entries). Gorp keys carry no delimiter between the type
prefix and the encoded key (`x/go/gorp/entries.go:204`), so bucketing is longest-match
and ambiguous matches are themselves reported. Per bucket it reports entry count and
byte totals; per gorp type it attempts a decode of every value with the production codec
chain (orc with msgpack fallback, `core/pkg/distribution/layer.go:151`). Decode
iteration uses `gorp.WrapReader` with explicit per-entry error checking; `OpenNexter` is
unsuitable because it drops undecodable entries silently (`x/go/gorp/reader.go:128`).

| Check                | Severity | Condition                                       |
| -------------------- | -------- | ----------------------------------------------- |
| `kv.decode`          | error    | an entry fails to decode under both codecs      |
| `kv.unknown-prefix`  | warning  | keys outside every known bucket, including      |
|                      |          | pre-v0.54 legacy prefixes that `normalize_keys` |
|                      |          | never moved (`x/go/gorp/table.go:286`)          |
| `kv.migration-state` | warning  | a table's applied-migration set is behind the   |
|                      |          | chain the binary ships                          |
| `kv.counter`         | error    | a key counter is missing or below the maximum   |
|                      |          | issued key it guards: the next allocation would |
|                      |          | re-issue an existing key                        |
| `kv.staging`         | warning  | leftover `sy_task_legacy_key/` staging entries  |
|                      |          | from a partially-run migration                  |

**Referential pass** (service types, no services constructed). The doctor imports the
entry and key types that the service layer owns and cross-references tables. There is no
runtime registry of gorp types to reuse; the wiring is an explicit table in the doctor,
following the codebase's explicit-wiring-site rule, seeded from the hand list in
`core/pkg/service/normalize_keys_test.go:44` and the task-config registry pattern
(`core/pkg/service/layer.go:594`).

| Check                      | Severity | Condition                                    |
| -------------------------- | -------- | -------------------------------------------- |
| `ontology.rel-key`         | error    | a relationship key fails to parse            |
| `ontology.rel-endpoint`    | warning  | a relationship references a resource that no |
|                            |          | longer exists                                |
| `ontology.resource-type`   | error    | a resource's type is not a known ontology    |
|                            |          | type: traversal panics on it today           |
|                            |          | (`core/pkg/service/ontology/service.go:55`)  |
| `ontology.resource-orphan` | warning  | a resource whose backing entity is gone:     |
|                            |          | queries silently drop it today, or whose     |
|                            |          | type no table backs                          |
| `ref.alias`                | warning  | a range alias whose range or channel is gone |
| `ref.range-kv`             | warning  | a range KV pair whose range is gone          |
| `ref.task-config`          | warning  | a task config entry with no task, or a task  |
|                            |          | with no stored configuration                 |
| `ref.credentials`          | warning  | credentials whose user no longer exists      |
| `ref.rack`                 | warning  | a task or device referencing a deleted rack  |
| `ref.channel-index`        | error    | a channel whose index channel entry is gone  |
| `ref.policy-object`        | info     | a policy object referencing a deleted        |
|                            |          | resource                                     |
| `ref.panel-tab`            | info     | a panel tab referencing a deleted entity     |

The Aspen data is summarized, not judged: node membership and heartbeats from
`aspen.cluster`, plus the digest count against the live key count (digests survive key
deletion by design, so the delta is reported as info).

### 4.4 Cross-layer checks

With both sides open, the doctor reconciles the channel table against the Cesium tree.
The host node's key comes from the persisted cluster state, because a node's Cesium
store only holds channels leased to that node.

| Check                | Severity | Condition                                         |
| -------------------- | -------- | ------------------------------------------------- |
| `cross.channel-dir`  | warning  | a non-virtual channel leased to this node with no |
|                      |          | Cesium directory, or a Cesium directory with no   |
|                      |          | channel entry (leaked storage)                    |
| `cross.channel-meta` | error    | disagreement between the KV channel entry and     |
|                      |          | `meta.json` on data type, index association, or   |
|                      |          | the virtual flag                                  |

Both directions of `cross.channel-dir` stay at warning severity: free channels and
multi-node lease movement make hard claims unsafe from one node's directory.

### 4.5 Output

The text report has four sections: a header (directory, node identity, engine versions,
whether KV was reachable), a summary (totals, garbage, top offenders), the findings
grouped by severity with one line per finding (`check-name  subject  message  hint`),
and a closing verdict line. `--verbose` adds the per-channel stat table and info
findings. `--json` emits the same content as one document: run metadata, the stats
objects, and a flat findings array covering both stores, so support tooling and tests
consume it without scraping text. The per-channel findings stay nested under the
time-series report as well, where they carry their channel.

### 4.6 Testing

`cesium/inspect` is tested against the golden fixture DBs (`cesium/internal/testdata/`)
plus a new corrupted-fixture corpus: each structural check gets a fixture that a small
test helper derives by bit-editing a valid DB (truncated index tail, swapped pointers,
overlapping bounds, a deleted data file, an oversized pointer). The doctor engine is
tested by building real gorp stores with production codecs and deliberately orphaned
entries, per the substitute-with-real- things rule. The command itself is exercised the
way `core/cmd/cmd_test.go` drives commands, asserting on `--json` output and exit codes.

## 5 Implementation phases

- **Phase 1: `cesium/inspect`.** The read-only decode surface exported from
  `cesium/internal/domain`, the inspect package with all Cesium stats and checks (deep
  included), and the fixture corpus. Lands green and reviewable inside the `cesium`
  module with no Core changes.
- **Phase 2: the command.** `pebblekv.OpenReadOnly`, gorp prefix helpers, the
  `core/pkg/doctor` engine (KV structural, referential, and cross-layer passes, text and
  JSON rendering, progress), the `core/cmd/doctor` shell, and the CLI reference page
  (`docs/site/src/pages/reference/core/cli-reference.mdx`).

The boundary buys reviewable units in two different Go modules and isolates format-
decoding risk from command plumbing. No further split is warranted.

**Compatibility**: The change is purely additive; no wire format, persisted shape, or
migration changes. The doctor reads every format `main` has released, including
pre-current `meta.json` versions (decoded leniently, never migrated) and msgpack-era KV
values (via the production fallback codec).

## 6 Resolved decisions

- **Offline, not in-server**: An API-served checker was rejected because the primary
  workflow is diagnosing a deployment that may not have a healthy Core, and every
  comparable storage-engine tool (`influxd inspect`, `cockroach debug`, `promtool tsdb`)
  settled offline for the same reason. The trade is real: the Console cannot surface
  doctor results without future server-side work, and KV checks require the Core
  stopped.
- **One command, not a report / verify / dump family**: InfluxDB's split taxonomy was
  rejected because our workflow is a single support round trip; splitting doubles it.
  The trade: a stats-only consumer pays for checks it did not ask for, mitigated by the
  skip flags.
- **The name `doctor`**: `inspect` reads as a neutral stats browser and fits an umbrella
  of subcommands we chose not to build; `check` excludes the stats half; `debug` implies
  developer internals. `doctor` imports an expectation from `brew`/`flutter` of
  environment checking rather than data checking; the fit to "run this and send me the
  output" outweighs it.
- **Strictly read-only, no `--fix`**: Every surveyed database tool separates reporting
  from repair, and a tool run on already-suspect data must be incapable of making it
  worse. The trade: trivially fixable findings (leftover `_gc` files, orphaned aliases)
  still require manual action or a future repair command.
- **Deep checks on by default**: The surveyed tools default shallow; this command
  defaults to the full inspection with opt-out flags, because the default run should be
  the complete evidence in one round trip. The trade: a first run on a large store is
  slow, mitigated by progress reporting and `--skip-deep`.
- **Warnings do not affect the exit code**: `brew doctor`'s warnings-exit-1 behavior is
  the documented anti-pattern; `pg_amcheck`'s 0/1/2 split is adopted verbatim.
- **A hand-wired type table instead of a new runtime registry**: A registry of all gorp
  types would serve exactly one consumer today. The explicit table in `core/pkg/doctor`
  follows the module-level wiring-site rule; if a second consumer appears, extraction
  becomes justified.

## 7 What this RFC does not cover

- Repair commands of any kind, and any `--fix` behavior.
- Server-side or Console-surfaced diagnostics.
- Multi-node aggregation: the doctor inspects one node's data directory; a cluster is
  diagnosed by running it per node.
- Driver, Console, or client-side stored state.

## 8 Open questions

1. **Thresholds**: The tiny-domain floor (bytes and duration), the near-zero gap band,
   the far-past cutoff, the far-future slack against wall clock, and the near-epoch band
   for the elapsed-as-timestamp signature. All are check parameters with defaults to be
   tuned against real field data dirs.
2. **Exclusion flag set**: Whether `--skip-deep` / `--skip-kv` / `--skip-ts` is the
   right granularity, or per-check suppression (`--skip cesium.micro-gap`) is worth
   adding in v1.
3. **Progress cadence**: Per-channel lines versus a throttled single-line status on
   stderr.
4. **Finding type placement**: Whether `cesium/inspect` and `core/pkg/doctor` share a
   finding type from `x`, or the doctor adapts Cesium-native report types. Decided at
   implementation time by whichever keeps `cesium/inspect`'s surface smaller.
