# 35 - Cesium Variable-Length Storage

**Feature Name**: Variable-Length Per-Channel Storage in Cesium <br /> **Status**: In
Progress <br /> **Start Date**: 2026-04-13 <br /> **Authors**: Emiliano Bonilla <br />

**Related:** [RFC 0008 - Cesium Columnar Storage](./0008-221012-cesium-columnar.md)

# 0 - Summary

Cesium's `unary` package gains support for persisting variable-density data types
(`StringT`, `JSONT`, `BytesT`). Previously these types could only exist as virtual
(non-persisted) channels. Fixed-density and variable-length channels share the same
package, the same `domain.DB` backend, and the same writer/iterator/delete paths. The
only difference is how sample indices translate to byte offsets: fixed-density channels
compute the offset arithmetically from a constant density, variable-length channels look
it up in an in-memory offset table built from length prefixes.

The on-disk format is uint32-length-prefixed samples stored in a single `domain.DB`.
There is no offset table, trailer, or auxiliary file on disk. An in-memory offset cache
provides O(1) sample-to-byte-offset translation, populated for free during writes and
rebuilt by scanning length prefixes on cold reads after process restart.

The `telem.Series` encoding for variable types was changed from newline-delimited to
uint32-length-prefixed across all four languages (Go, Python, TypeScript, C++). The
previous encoding was broken for samples containing literal newlines.

# 1 - Vocabulary

- **Fixed-density channel** - A channel whose samples all have the same byte size (e.g.,
  `Float64T` at 8 bytes per sample).
- **Variable-length channel** - A channel whose samples have varying byte sizes
  (`StringT`, `JSONT`, `BytesT`).
- **Virtual channel** - A channel that does not persist data to disk. Stored in
  `cesium/internal/virtual/`.
- **Length prefix** - A 4-byte little-endian uint32 preceding each sample in a
  variable-length series, encoding the sample's byte length.
- **Offset table** - An in-memory `[]uint32` mapping sample index to byte offset within
  a single domain.
- **Offset cache** - A per-channel `map[uint32]*offsetTable` keyed by domain index.
- **Domain** - A contiguous time-bounded region of data within a `domain.DB`. The
  fundamental unit of storage in Cesium.

# 2 - Motivation

## 2.0 - Variable-Length Channels Cannot Be Persisted

Cesium rejects variable-density data types for persisted channels. The validation gate
in `channel.go` returns "persisted channels cannot have variable density data types."
This forces all string, JSON, and bytes channels to be virtual, meaning their data is
transient and lost when the server restarts.

Use cases that require persisted variable-length data include event logs, configuration
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

Cesium has two per-channel database variants:

| Variant   | Package    | Persists | Data Types    | Index Required |
| --------- | ---------- | -------- | ------------- | -------------- |
| `unary`   | `unary/`   | Yes      | Any non-event | Yes            |
| `virtual` | `virtual/` | No       | Any           | No             |

Variant selection happens in `cesium/open.go`. Each channel opens as either virtual or
unary based on its `Virtual` flag. Within the unary package, the channel's data type
determines whether the writer and iterator treat it as fixed-density or variable-length.

## 3.1 - Series Encoding

All variable-length data uses uint32-length-prefixed encoding:

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

Variable-length channels store data in a single `domain.DB`. Each domain is a contiguous
region of uint32-length-prefixed samples. There is no trailer, offset table, or metadata
on disk. The domain layer sees an opaque byte blob.

## 3.3 - Offset Resolver

`offsetResolver` is the single type that translates sample indices to byte offsets. It
has two modes, selected at construction time from the channel's data type:

```go
type offsetResolver struct {
    density telem.Density
    cache   *offsetCache // nil for fixed-density channels
}

func newOffsetResolver(dt telem.DataType) *offsetResolver {
    if dt.IsVariable() {
        return &offsetResolver{cache: newOffsetCache()}
    }
    return &offsetResolver{density: dt.Density()}
}
```

Fixed-density channels carry a non-zero density and a nil cache. Their `byteOffset` is
`density.Size(sampleIdx)`. Variable-length channels carry a non-nil cache and rely on
per-domain offset tables.

The cache is a `map[uint32]*offsetTable` keyed by domain index. Each `offsetTable` holds
the `[]uint32` offsets for one domain along with the domain size the table was built
from. The cache hit is gated on the domain size matching the table's recorded size,
which catches the case where the writer appends more data to the same domain index
between commits.

**During writes.** A per-writer `offsetTracker` records each sample's byte offset as
data is appended. On close, the tracker's results are flushed into the cache. Zero extra
I/O.

**On cold read.** The cache is empty. The first read of a domain opens a reader and
calls `buildOffsetTable`, which scans length prefixes sequentially. For event-rate data
(10-100 events/sec) this takes under 10ms per domain. All subsequent reads within the
process lifetime are served from the cache.

**On delete or GC.** The resolver's `invalidate` method drops all cached tables. Next
read rebuilds them. Deletes could in principle invalidate only the affected domains, but
full invalidation is simpler and cold-read cost is low.

## 3.4 - Index Interaction

`index.Domain` resolves timestamps to sample indices. It returns `DistanceApproximation`
with `Lower`/`Upper` as `int64` sample counts. The conversion from sample index to byte
offset is the single point where the two modes diverge, and it is isolated behind
`offsetResolver.byteOffset`. The index package itself is unchanged.

## 3.5 - Writer Grouping

All non-virtual writers sit inside `idxWriter.internal`, a single
`map[ChannelKey]*unaryWriterState`. One `idxWriter` per index group. There is no split
between fixed-density and variable-length writers at this layer; the `unary.Writer`
already handles the distinction internally through its `offsetTracker`.

Validation enforces equal sample counts across all channels in the group. The density
equality check is skipped when either side is variable-length, since variable-length
series report `UnknownDensity`.

## 3.6 - Unified Package Structure

The initial design sketched `unary/` and a sibling `variable/` package. The shipped
implementation collapses both into a single `unary/` package with polymorphism pushed
down into `offsetResolver` and `offsetTracker`. The reason is that the two variants
share almost everything: the writer contract, the iterator state machine, the control
gate, the commit and close paths, the delete path. The only behaviors that actually
differ are:

| Concern                    | Fixed-density             | Variable-length            |
| -------------------------- | ------------------------- | -------------------------- |
| Sample count from bytes    | `density.SampleCount(n)`  | Offset table sample count  |
| Sample index to byte range | `density.Size(sampleIdx)` | `offsetTable.byteOffsetAt` |
| Per-write bookkeeping      | Increment counter         | Record offset per sample   |

Isolating those three behaviors inside `offsetResolver` / `offsetTracker` removes the
~130 lines of duplicated scaffolding the sibling-package design would have required, and
keeps the iterator and delete paths single-implementation.

# 4 - Alternatives Considered

## 4.0 - Parallel Offset Domain.DB

Two `domain.DB` instances per variable-length channel: one for data, one for uint32 byte
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

Use `\0` instead of `\n`. Works for `StringT` and `JSONT` (UTF-8 never contains null
bytes) but fails for `BytesT` (arbitrary binary). Would require two encodings instead of
one.

## 4.4 - Varint Length Prefix

Use variable-length integers instead of fixed uint32. Saves 2-3 bytes per sample.
Rejected because the savings are marginal for realistic sample sizes and uint32 is
simpler: fixed-width headers, predictable offsets, single-instruction decode.

## 4.5 - Sibling `variable/` Package

The sibling-package design kept fixed and variable code in separate packages to make the
density-dependent methods textually distinct. Rejected after the initial implementation
showed ~130 lines of structural duplication (`Open`, `Config`, `Close`, meta operations,
`HasDataFor`, `Size`) and parallel iterator state machines that had to evolve together
anyway. Collapsing into `unary` with a polymorphic `offsetResolver` preserved the
behavioral split at the one point that matters and removed the rest of the duplication.

# 5 - Implementation

## 5.0 - Phase 1: Series Encoding

Switched variable-length encoding from newline-delimited to uint32-length-prefixed in
`x/go/telem/`, `x/py/x/telem/`, `x/ts/src/telem/`, `x/cpp/telem/`. Updated all
signals/CDC producers, control digest encoding, and client codec layers. Added
`MarshalVariableSample()` utility.

## 5.1 - Phase 2: Sibling `variable/` Package

Added `cesium/internal/variable/` with its own DB, Writer, Iterator, Delete, and offset
cache. Wired into all cesium top-level files: `db.go`, `open.go`, `channel.go`,
`delete.go`, `control.go`, `iterator_open.go`, `iterator_stream.go`, `writer_open.go`,
`writer_stream.go`. Removed the persisted-variable validation gate.

## 5.2 - Phase 3: Package Unification

Collapsed the `variable/` package into `unary/`. The two variants now share one package,
one writer, one iterator, and one delete path. Density-dependent logic is isolated in
`offsetResolver` (per-DB) and `offsetTracker` (per-writer), both in
`cesium/internal/unary/resolver.go`. The sibling package and its parallel
implementations were removed.
