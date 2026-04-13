# 35 - Cesium Variable-Length Storage

**Feature Name**: Variable-Length Per-Channel Storage in Cesium <br /> **Status**: In
Progress <br /> **Start Date**: 2026-04-13 <br /> **Authors**: Emiliano Bonilla <br />

**Related:** [RFC 0008 - Cesium Columnar Storage](./0008-221012-cesium-columnar.md)

# 0 - Summary

Cesium gains a third per-channel database variant, `fixed` (renamed from `unary`), for
persisting variable-density data types (`StringT`, `JSONT`, `BytesT`). Previously these
types could only exist as virtual (non-persisted) channels.

The on-disk format is uint32-length-prefixed samples stored in a single `domain.DB`.
There is no offset table, trailer, or auxiliary file on disk. An in-memory offset cache
provides O(1) sample-to-byte-offset translation, built for free during writes and
rebuilt by scanning length prefixes on cold reads after process restart.

The `variable` package is a sibling of `fixed`, sharing the same external dependencies
(`domain.DB`, `index.Domain`, `control.Controller`) but implementing its own writer,
iterator, and delete logic. Variable channels sit inside the same `idxWriter` group as
fixed channels that share an index, participating in the same commit and control
transfer lifecycle.

The `telem.Series` encoding for variable types was changed from newline-delimited to
uint32-length-prefixed across all four languages (Go, Python, TypeScript, C++). The
previous encoding was broken for samples containing literal newlines.

# 1 - Vocabulary

- **Fixed-density channel** - A channel whose samples all have the same byte size (e.g.,
  Float64T at 8 bytes per sample). Stored in `cesium/internal/fixed/`.
- **Variable-density channel** - A channel whose samples have varying byte sizes
  (StringT, JSONT, BytesT). Stored in `cesium/internal/variable/`.
- **Virtual channel** - A channel that does not persist data to disk. Stored in
  `cesium/internal/virtual/`.
- **Length prefix** - A 4-byte little-endian uint32 preceding each sample in a
  variable-density series, encoding the sample's byte length.
- **Offset cache** - An in-memory `[]uint32` array per domain mapping sample index to
  byte offset within the domain.
- **Domain** - A contiguous time-bounded region of data within a `domain.DB`. The
  fundamental unit of storage in Cesium.

# 2 - Motivation

## 2.0 - Variable-Density Channels Cannot Be Persisted

Cesium rejects variable-density data types for persisted channels. The validation gate
in `channel.go` returns "persisted channels cannot have variable density data types."
This forces all string, JSON, and bytes channels to be virtual, meaning their data is
transient and lost when the server restarts.

Use cases that require persisted variable-density data include event logs, configuration
snapshots, audit trails, annotations, and any metadata that naturally varies in size.
Today these must be stored outside Synnax or lost on restart.

## 2.1 - Newline-Delimited Encoding Is Broken

The `telem.Series` encoding for variable types uses `\n` as a delimiter between samples.
Any sample containing a literal newline (common in JSON and multi-line strings) corrupts
`Series.Len()`, `Series.At()`, and `Series.Samples()`. This bug was masked because
variable channels were virtual and data was transient. Persisting this format would make
it a data corruption vector.

# 3 - Design

## 3.0 - Variant Taxonomy

| Variant    | Package     | Persists | Data Types         | Index Required |
| ---------- | ----------- | -------- | ------------------ | -------------- |
| `fixed`    | `fixed/`    | Yes      | Fixed-density only | Yes            |
| `variable` | `variable/` | Yes      | Variable-density   | Yes            |
| `virtual`  | `virtual/`  | No       | Any                | No             |

Variant selection happens in `cesium/open.go`: try virtual, then variable, then fixed.

## 3.1 - Series Encoding

All variable-density data uses uint32-length-prefixed encoding:

```
[uint32 len_0][sample_0 bytes][uint32 len_1][sample_1 bytes]...
```

Each sample is preceded by a 4-byte little-endian uint32 indicating the sample's byte
length. This format is binary safe (no reserved bytes), self-describing (decodable by
scanning length prefixes), and has a constant 4-byte overhead per sample. The length
prefix size is defined as a named constant in each language (`variableLengthPrefixSize`
in Go, `VARIABLE_LENGTH_PREFIX_SIZE` in Python/C++, `UINT32_SIZE` in TypeScript).

This encoding is used both on the wire (`telem.Series.Data`) and on disk. No conversion
is needed between in-memory and persisted format.

## 3.2 - On-Disk Format

Variable channels store data in a single `domain.DB`. Each domain is a contiguous region
of uint32-length-prefixed samples. There is no trailer, offset table, or metadata on
disk. The domain layer sees an opaque byte blob.

## 3.3 - In-Memory Offset Cache

The `variable.DB` maintains a per-domain offset cache: `map[uint32]*offsetTable` where
each `offsetTable` is a `[]uint32` array mapping sample index to byte offset.

The offset cache replaces the arithmetic (`sample_index * density`) that fixed-density
channels use for converting between sample indices and byte offsets.

**During writes:** The writer scans each sample's length prefix as it writes, recording
byte offsets in a growing slice. Zero extra I/O.

**On writer close:** The writer's offset table is stored in the DB's cache.

**On cold read:** The cache is empty. The first read of a domain scans length prefixes
sequentially to build the offset table. All subsequent reads use the cached table. For
event-rate data (10-100 events/sec), this scan takes under 10ms per domain.

**On delete:** Cached entries for affected domains are invalidated. Split domains
rebuild their offset tables lazily on next read.

**On GC:** The entire cache is invalidated since domain indices may shift during file
compaction.

## 3.4 - Index Interaction

`index.Domain` resolves timestamps to sample indices. It returns `DistanceApproximation`
with `Lower`/`Upper` as `int64` sample counts. The conversion from sample index to byte
offset is the single point where fixed and variable diverge:

- Fixed: `byteOffset = sampleIndex * density`
- Variable: `byteOffset = offsetCache[sampleIndex]`

The index package itself is unchanged.

## 3.5 - Writer Grouping

Variable channels sit inside `idxWriter.varInternal` alongside fixed channels in
`idxWriter.internal`. One `idxWriter` per index group. The idxWriter handles write,
validate, commit, and close for both fixed and variable writers.

Validation enforces equal sample counts across all channels in the group (both fixed and
variable). The density check is skipped for variable channels since their density is
`UnknownDensity`.

## 3.6 - Package Structure and Code Sharing

The `variable` package is a sibling of `fixed`, not a wrapper. Both compose the same
external dependencies but implement their own writer, iterator, and delete logic. This
is deliberate: the density-dependent methods are the heart of each package and work
fundamentally differently.

Methods that are density-dependent and reimplemented in `variable`:

| Method                     | Fixed                        | Variable                    |
| -------------------------- | ---------------------------- | --------------------------- |
| `Writer.len()`             | `density.SampleCount(bytes)` | In-memory offset table      |
| `Iterator.sliceDomain()`   | `density.Size(sampleIdx)`    | `offsetCache[sampleIdx]`    |
| `Iterator.autoNext/Prev()` | Density-based chunk sizing   | Offset-cache chunk sizing   |
| `calculateStartOffset()`   | `density.Size(sampleOffset)` | `offsetCache[sampleOffset]` |

Methods that are structurally duplicated (~130 lines of scaffolding): `Open`, `Config`,
`Close`, meta operations, `HasDataFor`, `Size`.

# 4 - Alternatives Considered

## 4.0 - Parallel Offset Domain.DB

Two `domain.DB` instances per variable channel: one for data, one for uint32 byte
offsets. Rejected because `domain.Writer` autonomously switches files when the file size
threshold is reached. The two DBs hit this threshold at different times, creating
misaligned domain boundaries. The coordination logic to force-sync file switches adds
significant complexity.

## 4.1 - Inline Trailer

Write an offset table as the last bytes of each domain on commit. Rejected because
`domain.Writer` is append-only and auto-commit updates the pointer in place. A trailer
written at commit N is embedded in the middle of the domain when commit N+1 appends more
data.

## 4.2 - Newline Delimiter

Keep the `\n`-delimited format. Rejected because `\n` is common in text data. Any sample
containing a literal newline corrupts Series operations.

## 4.3 - Null Byte Delimiter

Use `\0` instead of `\n`. Works for StringT and JSONT (UTF-8 never contains null bytes)
but fails for BytesT (arbitrary binary). Would require two encodings instead of one.

## 4.4 - Varint Length Prefix

Use variable-length integers instead of fixed uint32. Saves 2-3 bytes per sample.
Rejected because the savings are marginal for realistic sample sizes and uint32 is
simpler: fixed-width headers, predictable offsets, single-instruction decode.

# 5 - Implementation

## 5.0 - Phase 1: Series Encoding (Complete)

Switched variable-density encoding from newline-delimited to uint32-length-prefixed in
`x/go/telem/`, `x/py/x/telem/`, `x/ts/src/telem/`, `x/cpp/telem/`. Updated all
signals/CDC producers, control digest encoding, and client codec layers. Added
`MarshalVariableSample()` utility. PR #2217.

## 5.1 - Phase 2: Variable DB Package (Complete)

Added `cesium/internal/variable/` with DB, Writer, Iterator, Delete, and offset cache.
Wired into all cesium top-level files: `db.go`, `open.go`, `channel.go`, `delete.go`,
`control.go`, `iterator_open.go`, `iterator_stream.go`, `writer_open.go`,
`writer_stream.go`. Removed the validation gate. 48 internal tests, 26 integration tests
woven into existing test files. PR #2219.

## 5.2 - Phase 3: Rename (Complete)

Renamed `unary/` to `fixed/` across the cesium codebase. Mechanical find-replace with no
behavioral changes.
