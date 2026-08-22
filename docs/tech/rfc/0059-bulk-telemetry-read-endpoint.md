# 59 Bulk telemetry read endpoint

- **Author**: Emiliano Bonilla
- **Date**: 2026-08-16
- **Related**: [RFC 0006 - Freighter](0006-freighter.md),
  [RFC 0016 - Frame flight protocol](0016-frame-flight-protocol.md),
  [RFC 0039 - Server-side metadata import/export](0039-server-side-import-export.md),
  [RFC 0056 - Client telemetry layer](0056-client-telemetry-layer.md)

## 0 Summary

Reading a bounded range of telemetry today is a lockstep conversation over a WebSocket
in which the client formats every sample by hand. This RFC adds
`POST /api/v1/frame/read`: one request, one streamed response, with the output encoding
selected by the `Accept` header. The Core drives the iterator internally and writes
either the binary frame encoding or CSV straight to the socket. The WebSocket iterator
stays for cursor-driven reads.

## 1 Motivation

### 1.0 The client formats every sample

`framer.Reader` builds CSV by walking samples one at a time: `Series.at` per column per
sample, a string per value, a `RecordEntry` object and a string array per timestamp, and
a merge sort across index groups (`client/ts/src/framer/reader.ts:97-149`).

Measured on an M3 Max under Node 24 over 2 M samples and two columns:

| Path                                | Throughput   |
| ----------------------------------- | ------------ |
| `Series.at` alone                   | 9.1 Msamp/s  |
| `Series.at` plus `csv.formatValue`  | 6.5 Msamp/s  |
| Full row assembly and `TextEncoder` | 4.5 Msamp/s  |
| Byte copy of the same data          | 3170 Msamp/s |

The Core's frame codec encodes at 1.9 GB/s for a single channel and 6.0 GB/s across
eight, measured with `core/pkg/transport/http/framer/framer_bench_test.go`. The client
is roughly 100 times slower than the server that fed it. An hour of 50 channels at 1 kHz
is 180 M samples and about 40 s of formatting before any I/O, and the measurement above
excludes the per-timestamp allocation and merge.

### 1.1 The read is a lockstep conversation

`Iterator.execute` sends one command and blocks until the acknowledgement arrives
(`client/ts/src/framer/iterator.ts:231-239`). At the default chunk of 1e5 samples per
channel, the same hour is about 1800 sequential round trips. On a LAN the stall is
invisible. At a 40 ms round trip it caps the transfer near 10 MB/s regardless of how
fast either end is, and every hop holds about 20 MB in flight.

### 1.2 How it got this way

Three sound decisions met. RFC 0016 replaced JSON frames with the binary codec, fixing
the wire format but not the shape of the exchange. The iterator was built for paging,
where ask-receive-ask is correct. CSV was written in TypeScript because that is where
the download button lives. Nobody chose to format 180 M samples in a browser.

## 2 Vocabulary

- **Bulk read**: A whole-range read of a fixed set of channels, requested once and
  consumed as a stream.
- **Cursor read**: A read driven by `seek`, `next`, and `prev`, where the caller decides
  what to fetch next from what it has already seen.
- **Record**: One length-delimited unit in a `/read` response body.
- **Terminator**: The final record of a frame-encoded body, carrying success or the
  error that ended the read.

## 3 Principles

1. **One request, one streamed response**: A bulk read is not a conversation. The Core
   drives the iterator; the client receives bytes.
2. **Encoding is negotiated, never branched**: The endpoint declares its encoders and
   `Accept` selects one. No format-specific code enters the handler.
3. **The Core formats telemetry**: Per-sample work written in the Core is written once.
   Written in the clients it is written four times and drifts.
4. **Everything that can fail, fails before the first byte**: Authorization, channel
   resolution, bounds validation, and iterator open all complete while the response is
   still an ordinary error response.
5. **Bulk and cursor reads are different jobs**: `/read` serves the first. The WebSocket
   iterator keeps serving the second.

## 4 Design

### 4.0 The endpoint

`POST /api/v1/frame/read`, registered beside the existing frame routes in
`core/pkg/transport/http/http.go`:

```go
FrameRead: http.NewUnaryServer[framer.ReadRequest, framer.ReadResponse](
    router,
    "/api/v1/frame/read",
    http.WithResponseEncoders(framer.FrameEncoder, framer.CSVEncoder),
    http.WithStreamingResponse(),
),
```

`ReadRequest` carries `keys`, `bounds`, and `downsample_factor`, matching the fields
`IteratorRequest` already accepts (`core/pkg/service/framer/iterator/service.go:33-56`).

The handler enforces `access.ActionRetrieve` over `framer.OntologyIDs(keys)`, exactly as
`openIterator` does (`core/pkg/api/framer/framer.go:194`), resolves the channel records,
opens a service-layer iterator, and returns:

```go
type ReadResponse struct {
    Iterator *framer.Iterator
    Channels []channel.Channel
}
```

`ReadResponse` is never serialized by a general codec. It is the handle the two
registered encoders drive, and each encoder closes the iterator in a defer. Opening the
iterator in the handler keeps Principle 4: a bad key or a denied subject is a 400 with a
normal error body.

Calculated channels and downsampling come from the service-layer iterator, so `/read`
inherits both without knowing they exist.

### 4.1 Streaming responses in the unary HTTP server

`unaryServer.fiberHandler` encodes the response into a byte slice and writes it
(`freighter/go/http/unary_server.go:196-203`). `Encoder` already declares
`EncodeStream(context.Context, io.Writer, any) error` (`x/go/encoding/encoding.go:35`)
and nothing calls it. Fiber v3.4.0 provides `Ctx.SendStreamWriter(func(*bufio.Writer))`,
which is fasthttp chunked output.

`http.WithStreamingResponse()` is a new `UnaryServerOption` that switches a route from
`Encode` to `EncodeStream` driven by `SendStreamWriter`. It is opt-in because the two
differ in a way every existing route would feel: a buffered encode that fails still
produces a 400, and a streamed encode that fails has already committed a 200.

fasthttp runs the stream writer after the handler returns, so the closure captures the
server context rather than the `fiber.Ctx`, which fiber recycles. Middleware finishes
before the body is written, which means an instrumentation span closes at handler exit
rather than at last byte.

`WriteBufferSize` stays at fiber's 4096 default today (`core/pkg/server/http.go:84`).
`/read` writes far larger runs than that, so the server config sets it explicitly.

### 4.2 The frame encoding

Content type `application/vnd.synnax.frame`, the type the WebSocket codec already uses.

The body is a sequence of records:

```
[uint8 kind][uint32 length][bytes]
```

`kind` 0 is a frame encoded by the existing codec. `kind` 1 is the terminator, whose
bytes are a JSON `errors.Payload`, empty on success. Five bytes per record is negligible
against a chunk of telemetry.

No preamble and no handshake. The server encodes with `codec.NewStatic(keys, dataTypes)`
and the client decodes with `new Codec(keys, dataTypes)` over the keys it asked for. The
index channels the Core pulls in to timestamp a CSV never reach this encoding. Both
sides sort keys ascending before use (`codec.go:303`, `codec.ts:118`) and both start at
sequence number 1, so the states match by construction.

### 4.3 The CSV encoding

Content type `text/csv`, living beside the WebSocket frame codec at
`core/pkg/transport/http/framer/`.

Channels group by their index channel. Each group contributes its index column first,
then its data columns, each headed by the channel name. Rows merge across groups by
timestamp, with empty cells where a group has no sample at that timestamp, and a row is
written as soon as no later frame can extend it. This is the algorithm in
`client/ts/src/framer/reader.ts:77-198`, moved to the Core.

Values are appended with `strconv.Append*` into a reused buffer. There is no string per
sample and no record object per timestamp, which is the difference between the two
throughput columns in §1.0.

### 4.4 Compression

Compression follows the encoder, not the request. The route names the encoders whose
output is worth gzipping, and only those are compressed, and only when `Accept-Encoding`
asks. CSV is named; the frame encoding is not.

Measured on 10M float32 samples: gzip shrinks CSV by 70 to 81%, and shrinks a frame body
by 7.6% once the samples carry real noise instead of a ramp. The frame case costs 157 ms
of CPU across both ends to save 3 MB, so it only pays below about 150 Mbit/s.

The caller cannot make this choice for itself. `Accept-Encoding` is a forbidden header
name in the Fetch spec, so a browser always asks for gzip and the Console cannot decline
it. Deciding at the encoder is what keeps a frame read uncompressed there.

`Content-Encoding` is set before the first byte and the gzip writer wraps the
`bufio.Writer` the stream writer receives, so compression costs no buffering. It runs at
the fastest level, sharing a core with the encoder.

### 4.5 Failure after the first byte

The status line is committed before any body byte, so a read that dies partway cannot
become a 400. Under Principle 4 the failures that survive to that point are disk I/O and
a node dropping mid-read.

The frame encoding ends with a terminator record naming the error. A body that ends
without one is a transport failure, and clients distinguish the two. The terminator
carries what the encoder sees, not `Iterator.Error`: the iterator sets that error on
ordinary exhaustion whenever the read bounds reach past the last sample
(`cesium/internal/unary/iterator.go:211`), so it cannot tell a failed read from a
complete one.

CSV has no in-band slot that does not corrupt the file for every other tool, so a failed
CSV read ends as a short body. Clients surface an incomplete response as an error.

HTTP trailers are not used. `fetch` exposes no way to read them, so the Console, the
main consumer, would never see one.

### 4.6 Client surfaces

- **TypeScript**: `client.read(tr, channels)` posts to `/read` through
  `FileTransport.download` (`freighter/ts/src/http.ts:203`) and decodes records
  incrementally with the static `Codec`. The `FileEncoding` union
  (`freighter/ts/src/file.ts:26`) opens to carry the two content types.
- **Console**: `useDownload` (`console/src/platform/csv/useDownload.ts:49`) requests
  `text/csv` and pipes the response into `Runtime.downloadStream` unchanged.
- **Python**: `client.read` posts to `/read`. Freighter gains a download-to-memory
  variant beside its download-to-file (`freighter/py/freighter/http.py:138`).
- **C++**: unchanged. It speaks gRPC, and the endpoint is not exposed there
  (`core/pkg/transport/grpc/grpc.go:80` binds it as a noop). Nothing in the Driver reads
  history in bulk, so giving the client a second transport would buy a caller that does
  not exist. A C++ consumer would first settle whether the read belongs on gRPC as a
  server-streaming method.

Deleted: `client/ts/src/framer/reader.ts` in full, including `Reader`, `RecordEntry`,
and `mergeSortedRecords`; the `responseType: "csv"` overload on `client.read`; and
`x/ts/src/csv`, whose only consumer is that file.

## 5 Implementation phases

**Phase 1: The endpoint.** `WithStreamingResponse` in freighter, both encoders, the
handler, access control, compression, and the server config change. Additive: no
existing route changes behavior, and the phase is provable with Go specs alone.

**Phase 2: The clients and the kill list.** `client.read` in TypeScript and Python onto
`/read`, the Console export cutover, and the deletions above. The behavior change and
the removals land together so bisection points at one commit.

## 6 Resolved decisions

**6.0 The WebSocket iterator stays.** A single read path was considered and rejected.
`/read` cannot express `seekLast` followed by `prev`, which `readLatest` uses
(`client/ts/src/framer/client.ts:227-239`), and cursor-driven scrubbing wants a stateful
cursor rather than repeated bounded reads. The trade is real: two read paths mean two
places where alignment and gap handling can diverge, which is the class of defect RFC
0056 was written to close.

**6.1 The encoder seam, not a download handler.** A new freighter server kind whose
handler receives an `io.Writer` was considered. It is more honest about a streaming
producer and would let the CSV writer take channel metadata as an argument rather than
smuggling it through the response value. It was rejected because negotiation, the
encoder registry, and the client side already exist and are in production for the zip
export (`core/pkg/transport/http/http.go:296`), and rebuilding them per handler to avoid
one awkward field is a poor trade. `ReadResponse` being a handle rather than data is the
cost paid.

**6.2 Two encodings.** The binary frame codec serves clients; CSV serves export and is
the format whose cost motivated this RFC.

**6.3 An in-band terminator, not trailers and not truncation alone.** Trailers are
unreadable from `fetch`. Truncation alone would work for both formats and was rejected
for the frame encoding because it discards the reason a read failed. CSV takes
truncation because no alternative leaves a valid file.

**6.4 HTTP/1.1 chunked.** The HTTP branch matches `cmux.HTTP1Fast()`
(`core/pkg/server/http.go:44`) and runs on fasthttp, which serves no HTTP/2. For a
single bulk response HTTP/2 adds framing and a flow-control window to tune and buys
nothing, since its wins are in multiplexing small requests. gRPC is already the
node-to-node transport and no browser speaks it. A WebSocket keeps a protocol state
machine and cannot be piped to a file handle.

**6.5 Compression at the endpoint, not at a proxy.** Leaving it to a reverse proxy is
simpler and was rejected because the benefit would exist only for deployments that have
one.

## 7 What this RFC does not cover

- Telemetry import. Writing carries control authority, a start timestamp, commit
  boundaries, and partial-upload recovery, none of which have an analogue on the read
  side. Browsers also refuse to stream a request body over HTTP/1.1, so import cannot
  reuse this transport shape. It gets its own design.
- The iterator's own throughput. `/read` drives the same iterator and inherits its
  behavior unchanged.
- Node-to-node transport, which stays gRPC.

## 8 Open questions

1. Samples per emitted record, which trades chunk overhead against server memory.
2. The explicit `WriteBufferSize` value.
3. Whether CSV headers use channel names or the requesting range's aliases.
