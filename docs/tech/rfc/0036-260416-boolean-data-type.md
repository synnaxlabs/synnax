# 36 - Boolean Data Type

**Feature Name**: Boolean Data Type <br /> **Status**: Draft <br /> **Start Date**:
2026-04-16 <br /> **Authors**: Emiliano Bonilla <br />

**Related:** [RFC 0007 - Data Types](./0007-220823-data-type.md),
[RFC 0035 - Cesium Variable-Length Storage](./0035-260413-cesium-variable-length-storage.md)

# 0 - Summary

Synnax gains a first-class boolean data type, `BoolT`. A boolean channel has three
distinct representations that do not agree:

- **In memory**: byte-packed. One sample per byte, canonical values `0x00` (false) and
  `0x01` (true). The `telem.Series` density invariant is preserved; every existing
  iterator, writer, Cesium reader, and client `TypedArray` view treats a bool sample
  identically to a uint8 sample.
- **On the wire**: bit-packed. The Freighter frame codec packs eight samples into one
  byte on send and unpacks back to byte-packed `Series` on receive. Digital traffic
  drops by a factor of eight before any further compression.
- **On disk**: byte-packed today, identical to the in-memory form. A future Cesium
  storage codec can compress boolean channels independently (bit-packing, run-length
  encoding, or both) without affecting in-memory or wire representations.

Write paths normalize any nonzero source byte to `0x01` at the client and server ingest
boundaries. Reads always return canonical bytes. The type exists to carry digital-signal
intent across the stack so that Console, Arc, schematics, and driver integrations can
specialize on it, and to unlock bit-level wire efficiency for the substantial class of
workloads where digital I/O traffic is significant.

# 1 - Vocabulary

- **Boolean channel** - A channel whose data type is `BoolT`.
- **Canonical representation** - The rule that a well-formed bool sample in memory or on
  disk is exactly `0x00` or `0x01`. No other byte value is valid.
- **Normalization** - The write-side transformation that maps any nonzero source byte to
  `0x01` and any zero byte to `0x00` before the sample enters storage.
- **Density** - Bytes per sample in memory and on disk. For `BoolT`, density is 1,
  identical to `Uint8T`. Wire density is one eighth of memory density.
- **Wire codec** - The per-type encode and decode stage inside Freighter that translates
  between a `Series`'s in-memory byte sequence and its on-the-wire byte sequence.
  Fixed-density types other than `BoolT` encode as raw bytes; variable-length types
  (`StringT`, `JSONT`, `BytesT`) use the length-prefixed codec defined in RFC 0035;
  `BoolT` uses the bit-packed codec defined in §3.2.
- **Bit-packed wire format** - The Freighter encoding for `BoolT` series: `ceil(N / 8)`
  bytes of packed data with LSB-first bit order within each byte, accompanied by the
  `sample_count` already present in the frame header.

# 2 - Motivation

## 2.0 - No First-Class Boolean Type

Digital I/O sources produce `{0, 1}` signals: Modbus coils, LabJack DIO, NI digital
lines, OPC UA boolean nodes, EtherCAT `EC_BOOLEAN` objects. Each of these surfaces its
values as `Uint8T` today, because that is the narrowest fixed-width numeric type
available. The type system therefore cannot distinguish "this channel is a digital
signal" from "this is an arbitrary 8-bit integer."

The affordance gap this creates is consistent across the stack:

- Console value displays render `1` and `0` instead of `ON` and `OFF`, and cannot
  auto-wire toggle or indicator affordances.
- Schematic control chips cannot specialize their symbol or interaction model on the
  source channel's type.
- Arc programs cannot distinguish boolean logic from integer arithmetic at the type
  level. Comparison and logical operators land in the same `u8` namespace as addition.
- Driver integrations re-derive the boolean constraint per protocol. Modbus knows its
  coils are `{0, 1}`; LabJack knows its DIO is `{0, 1}`; neither communicates that fact
  upward through the channel's type.

Each consumer works around the gap locally. A shared type removes the workaround.

## 2.1 - Digital Traffic Is Significant At The Wire Layer

Industrial deployments routinely create hundreds or thousands of digital channels, and
the driver layer surfaces each bit as its own channel. A Modbus `readCoils(addr, 16)`
call reads sixteen bits in a single protocol request and fans them out to sixteen
separate `Uint8T` channels in Synnax. A deployment with a thousand such channels
sampling at one kilohertz produces one megabyte per second of digital traffic in the
current byte-packed wire format. That figure is a rounding error next to a handful of
`Float64T` channels at ten kilohertz, but in deployments where digital I/O dominates
(interlock matrices, distributed status reporting, valve and relay control at scale) it
is the dominant traffic class and bit packing on the wire reduces it by a factor of
eight.

This RFC takes the bandwidth reduction seriously enough to pay for it with a codec
branch, but not seriously enough to let it reshape the in-memory `Series` invariant. See
§3.0 for the separation.

## 2.2 - Ecosystem Precedent

Every mainstream time-series system separates in-memory representation from wire and
disk representation for booleans:

- InfluxDB TSM: byte-packed in memory, bit-packed with run-length encoding on disk.
- Prometheus: byte-packed in memory, gorilla-compressed on disk.
- Parquet: byte-packed after decode, RLE plus bit-packing in the file format.
- EtherCAT: byte-packed in host memory, bit-packed in PDO wire frames.

The client ecosystems Synnax integrates with also default to byte-packed in memory.
NumPy `np.bool_` is one byte per element. Go `bool` is one byte. C++ `bool` is
effectively one byte in every mainstream ABI. JavaScript has no native boolean typed
array; `Uint8Array` with `0`/`1` is the idiomatic interchange.

Apache Arrow is the notable exception and represents booleans as packed bits throughout,
including in memory. Arrow paid for that choice with a purpose-built boolean builder,
bit-offset-aware compute kernels, and a special case in every pipeline stage that
indexes by element. Synnax has no such kernel library and should not build one for a
single type.

# 3 - Design

## 3.0 - Representation Axes

The design separates three representations that conventional discussions conflate:

| Axis      | Representation                       | Density |
| --------- | ------------------------------------ | ------- |
| In memory | Byte-packed, canonical `{0x00,0x01}` | 1 byte  |
| On wire   | Bit-packed, LSB-first                | 1 bit   |
| On disk   | Byte-packed, identical to memory     | 1 byte  |

Each axis is negotiated independently. Treating them as a single choice is the mistake
that forces either Arrow-style bit-packed ergonomics or naive byte-packed bandwidth
waste. Synnax chooses byte-packed in memory (for compute ergonomics and invariant
preservation) and bit-packed on the wire (for bandwidth). Disk stays byte-packed for
now, with a future Cesium storage codec expected to compress aggressively using the same
codec abstraction the wire layer establishes here.

## 3.1 - In Memory: Byte-Packed

Each `BoolT` sample occupies one byte in `Series.Data`. The sample value is exactly
`0x00` or `0x01`. Density is 1. The `Series` invariant (sample `i` lives at byte offset
`i * density`) holds. Every existing iterator, writer, Cesium domain reader,
Distribution aligner, and client-side `TypedArray` view treats a bool sample identically
to a uint8 sample. No iterator signature, no writer contract, and no offset arithmetic
changes.

Client-language surfaces expose language-native booleans (`bool` in Go, `boolean` in
TypeScript, `np.bool_` in Python, `bool` in C++). The conversion between the language
value and the canonical byte happens at the edge of the client library, not inside
`Series` operations.

## 3.2 - On The Wire: Bit-Packed Codec

The Freighter frame codec encodes `BoolT` series using a bit-packed format. A series of
`N` samples transmits as `ceil(N / 8)` bytes. Sample `k` occupies bit `k mod 8` of byte
`k / 8`, LSB-first within each byte. The `sample_count` field already present in
Freighter's per-series frame header tells the decoder how many bits to recover from the
trailing partial byte; no new header field is required.

Encoder (Go sketch):

```go
func encodeBool(src []byte) []byte {
    dst := make([]byte, (len(src)+7)/8)
    for i, b := range src {
        if b != 0 {
            dst[i/8] |= 1 << (i % 8)
        }
    }
    return dst
}
```

Decoder (Go sketch):

```go
func decodeBool(src []byte, n int) []byte {
    dst := make([]byte, n)
    for i := 0; i < n; i++ {
        dst[i] = (src[i/8] >> (i % 8)) & 1
    }
    return dst
}
```

The decoder produces canonical bytes (`0x00` or `0x01`) unconditionally. The in-memory
`Series` never observes non-canonical bytes through this path.

Each language's Freighter frame codec gains one branch for `BoolT`: Go at
`core/pkg/distribution/framer/codec/codec.go`, TypeScript at
`client/ts/src/framer/codec.ts`, Python at `client/py/synnax/framer/codec.py`, and C++
at `client/cpp/framer/codec.cpp`. The frame codec already dispatches on
`DataType.IsVariable()` for the length-prefixed variable-length types introduced in
RFC 0035. `BoolT` is a second dispatch arm in the same codec, distinct from both the
fixed-density raw-bytes path and the variable-length length-prefixed path.

The in-memory `Series.Data` remains byte-packed canonical bytes. Bit-packing happens
only when the frame codec serializes a `Series` onto the wire, and unpacking happens
only when the frame codec deserializes a `Series` off the wire. The `Series` type itself
does not grow a new marshaling contract.

Network impact: a deployment with 1000 digital channels sampling at 1 kHz drops from 1
MB/s to 125 KB/s of digital traffic before any further compression is applied.

## 3.3 - On Disk: Byte-Packed

Cesium persists `BoolT` samples as byte-packed bytes on disk. Density lookup returns 1,
`offsetResolver` takes the fixed-density branch, and domain layout, iterator state,
writer grouping, commit, and delete paths are all unchanged.

This choice is deliberately scoped to the initial implementation. A future storage codec
layer is expected to compress `BoolT` aggressively: bit-packing alone recovers the wire
codec's 8x density, and run-length encoding on top typically delivers compression ratios
well above that because industrial digital signals are sparse (most samples repeat). The
codec dispatch abstraction introduced at the wire layer in §3.2 is the natural substrate
for this storage codec when it is built.

## 3.4 - Canonical Representation

A well-formed `BoolT` sample in memory or on disk is exactly one of two byte values:

- `0x00` represents false.
- `0x01` represents true.

No other byte value is valid. Reads from Cesium, reads through the wire codec, and the
`Series.At(i)` accessor always return one of these two canonical values.

The bit-packed wire format does not introduce a third canonical value; it is a lossless
recoding of the two-value in-memory form. Encode sees canonical bytes and produces
packed bits; decode sees packed bits and produces canonical bytes.

## 3.5 - Write-Path Normalization

Writes accept any byte value at the client API boundary. A source byte `b` is mapped to
`b != 0 ? 0x01 : 0x00` before the sample enters the in-memory `Series`. The same
normalization runs again at server ingest as a defense-in-depth check. Once a sample is
in a `Series`, it is canonical; the wire codec therefore never has to normalize, and the
disk layer never has to normalize on read.

Normalization matches C-family boolean semantics, matches how every mainstream client
language treats "truthiness," and removes the need for callers to pre-canonicalize
values coming from arithmetic (`x > threshold`, `count & 1`, the result of a cast from a
signed integer).

## 3.6 - Cast Matrix

Casts involving `BoolT` are deterministic and require no special-case error handling:

- Any numeric type to `BoolT`: `x != 0 ? 0x01 : 0x00`.
- `BoolT` to any numeric type: direct copy of the source byte. The byte is guaranteed to
  be `0x00` or `0x01` by the canonical representation, both of which are valid values in
  every integer and floating-point target.
- `BoolT` to `BoolT`: identity.

The cast table in each language implementation reuses the existing per-pair cast
infrastructure. There is no boolean-specific code path beyond the entry in the table.

## 3.7 - Cross-Language Surface

The type is added to each language's telem module using the existing data type
scaffolding. No new generic machinery is introduced beyond the codec branch in §3.2.

**Go** (`x/go/telem/data_type.go`). A new `BoolT DataType = "bool"` constant with
density 1. The density lookup table gains one entry. The Go-level client API for a
boolean channel reads and writes `bool`; the conversion between `bool` and the canonical
byte happens at the edge of the client library. The Freighter frame codec branch lives
in `core/pkg/distribution/framer/codec/codec.go`.

**TypeScript** (`x/ts/src/telem/telem.ts`). A new `DataType.BOOLEAN` instance annotated
with density 1. The `TypedArray` backing is `Uint8Array`; the public client surface
returns and accepts `boolean[]`. Conversion happens in the series accessor. The
Freighter frame codec branch lives in `client/ts/src/framer/codec.ts`.

**Python** (`x/py/x/telem/telem.py`). A new `DataType.BOOL` entry mapped to `np.bool_`.
NumPy's `bool_` is already one byte, so the in-memory mapping is zero-overhead and
`np.ndarray[bool]` round-trips through Synnax without reshaping. The Freighter frame
codec branch lives in `client/py/synnax/framer/codec.py`.

**C++** (`x/cpp/telem/telem.h`). A new `BOOL_T` instance with density 1. At the byte
level the storage is `uint8_t`; at the value level the client surface exposes `bool`.
The Freighter frame codec branch lives in `client/cpp/framer/codec.cpp`.

# 4 - Alternatives Considered

## 4.0 - Bit-Packed In Memory

Under this alternative, `Series.Data` stores packed bits and sample `i` is extracted via
`(data[i / 8] >> (i % 8)) & 1`. Wire and disk match.

Rejected because it forces sub-byte offset arithmetic into `offsetResolver` and every
consumer that indexes into a series, introduces a partial-byte tail convention at every
write boundary, requires a bit-aware iterator contract in Cesium's `unary` package, and
produces a synthetic `Bit1Array` shim in TypeScript (which has no native primitive), an
unpacking step in Python (whose `np.bool_` is byte-wide anyway), and a hand-rolled
bitset in C++ (to avoid `std::vector<bool>`). This is the Arrow path; it is coherent,
but it requires bit-offset-aware compute kernels to exist, and Synnax has no such
library to reuse. The 8x in-memory density win does not justify the abstraction cost.

## 4.1 - Byte-Packed On The Wire

Under this alternative, the wire codec encodes `BoolT` as raw byte-packed bytes,
identical to `Uint8T`. No codec branch is added.

Rejected because the 8x wire traffic penalty is paid for every deployment whose workload
is dominated by digital I/O, which is the exact traffic class most amenable to bit
packing. The cost of the wire codec branch (four languages, O(N) encode and decode,
bounded by memcpy throughput) is small relative to the recurring bandwidth cost. The
codec abstraction itself also opens the door to per-type codecs for other types (gorilla
for `Float64T`, delta for monotonic counters, RLE for any type with repeated values),
which this RFC does not commit to building but does not foreclose on either.

## 4.2 - Strict Rejection Of Non-Canonical Writes

Under this alternative, writes of any byte other than `0x00` or `0x01` return an error.
The invariant is cleaner in the sense that every write path treats the sample as already
canonical.

Rejected because it pushes a normalization burden onto every caller that computes a
boolean value from arithmetic. The resulting ergonomics diverge from C-family
conventions, and the defensive normalization has to happen somewhere; centralizing it at
the write boundary is strictly less code than distributing it across every caller.

## 4.3 - Keep Using `Uint8T`

Under this alternative, digital channels continue to be typed as `Uint8T` and consumers
infer booleanness from context. This is the current state.

Rejected because the affordance gap in Console, Arc, schematic, and driver integrations
is permanent under this choice, and the workarounds each consumer maintains locally are
strictly more code than one shared type. Keeping `Uint8T` also forfeits the wire-layer
bit-packing opportunity described in §3.2, because the codec cannot safely assume that
an arbitrary `Uint8T` channel is digital.

# 5 - Open Questions

The following concerns are downstream of this RFC and are explicitly out of scope. Each
warrants its own decision record.

## 5.0 - Driver Migration

Existing Modbus coils, LabJack DIO, NI digital lines, and OPC UA boolean nodes create
channels typed as `Uint8T`. The migration strategy for these (silent flip on next
channel creation, per-integration opt-in flag, greenfield-only with no change to
existing channels) is a breaking-change question that this RFC does not resolve.

## 5.1 - Arc Type System

Whether Arc's type system treats `bool` as a distinct type or as an alias for `u8`
affects the signatures of comparison and logical operators, the set of valid casts, and
the shape of compiler diagnostics. The answer interacts with Arc's existing treatment of
truthiness in `if` and loop conditions.

## 5.2 - Console And Schematic Rendering

Value labels, line plot axes and gridlines, schematic value overlays, and toggle-capable
control chips all have rendering decisions to make when the source channel is boolean.
These are UI decisions that build on top of the type introduced here, not protocol
decisions.

## 5.3 - Generalized Per-Type Wire Codecs

The codec dispatch introduced in §3.2 for `BoolT` sits alongside the length-prefixed
codec introduced in RFC 0035 for variable-length types. Whether Synnax should generalize
this into a first-class per-type codec registry (admitting gorilla for `Float64T`, delta
for monotonic counters, RLE for any type, and a future Cesium disk codec layer reusing
the same registry) is a strictly larger question than `BoolT` and is out of scope here.
`BoolT` is the forcing function; the generalization is the follow-up.
