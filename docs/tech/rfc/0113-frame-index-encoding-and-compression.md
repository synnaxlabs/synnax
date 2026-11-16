# 113 Frame index encoding and compression

- **Author**: Patrick Dotson
- **Date**: 2026-11-16

## 0 Summary

Frames cross the wire uncompressed. Synthetic telemetry shows that general compression
gains little on live streams, while byte shuffling plus zstd gains 1.4 to 2 times. Large
historical reads gain more. This RFC adds three things in order: a Core metric that
measures real frame sizes, opt-in zstd for `/frame/read` bodies, and a constant-step
index encoding in the frame codec. An opt-in compressed stream codec waits for real
data.

## 1 Motivation

- **No stream compression**: The WebSocket server accepts no permessage-deflate
  (`freighter/go/http/stream_server.go:155`). The Driver uses gRPC with no compression
  (`core/pkg/server/grpc.go:67`).
- **Frame reads skip gzip on purpose**: Only CSV reads are gzipped
  (`core/pkg/transport/http/http.go:195-199`).
- **Indexes ship as raw timestamps**: The codec sends 8 bytes for each index sample
  (`core/pkg/distribution/framer/codec/codec.go`), even when a hardware-timed task
  spaces them evenly.
- **Slow links pay for it**: A Core reached over WAN, cellular, or satellite links
  carries 100 `float32` channels at 10 kHz as about 32 Mbit/s.

## 2 Measurements

The production Go codec encoded synthetic frames: one index channel and N data channels.
The analog channels are a sine through a 16-bit ADC over ±10 V with one LSB of noise.
Every tenth channel is a `uint8` valve state. "Scaled" applies a calibration
(`1.0371 * v + 0.0137`), which breaks the exact low mantissa bits of a quantized value.
Live frames batch at 25 Hz, the stream rate. Each ratio is raw bytes over encoded bytes.

- **a**: Index as start and step.
- **b**: Deflate level 1 on each frame, as permessage-deflate with no context takeover.
- **c**: zstd level 1 (`klauspost/compress`) on each frame.
- **d**: Byte shuffle by sample width, then zstd level 1.

| Traffic                                     | Frame  | Index share | a     | b     | c     | d     | a+d   |
| ------------------------------------------- | ------ | ----------- | ----- | ----- | ----- | ----- | ----- |
| Stream, 10 ch `float32`, 1 kHz              | 1.8 kB | 17.5%       | 1.20x | 1.28x | 1.34x | 1.60x | 1.73x |
| Stream, 100 ch `float32`, 1 kHz             | 15 kB  | 2.1%        | 1.02x | 1.30x | 1.31x | 1.74x | 1.76x |
| Stream, 100 ch `float32`, 1 kHz, scaled     | 15 kB  | 2.1%        | 1.02x | 1.11x | 1.12x | 1.42x | 1.43x |
| Stream, 100 ch `float32`, 10 kHz            | 151 kB | 2.1%        | 1.02x | 1.29x | 1.31x | 2.06x | 2.08x |
| Stream, 20 ch `float64`, 1 kHz, scaled      | 6.2 kB | 5.2%        | 1.05x | 1.06x | 1.04x | 1.21x | 1.23x |
| Stream, 10 ch `float32`, 100 Hz             | 213 B  | 15.0%       | 1.08x | 1.12x | 0.95x | 0.99x | 1.02x |
| Read, 100 ch `float32`, 1 kHz, 10 s         | 3.8 MB | 2.1%        | 1.02x | 1.15x | 1.41x | 2.65x | 2.65x |
| Read, 10 ch `float64`, 10 kHz, 10 s, scaled | 8.1 MB | 9.9%        | 1.11x | 1.27x | 1.33x | 1.52x | 1.52x |

| Step (100 ch, 1 kHz frames) | Throughput | Per frame |
| --------------------------- | ---------- | --------- |
| Codec encode                | 1734 MB/s  | 8.7 µs    |
| Deflate level 1             | 53 MB/s    | 288 µs    |
| zstd level 1                | 215 MB/s   | 71 µs     |
| Shuffle and zstd level 1    | 107 MB/s   | 142 µs    |
| zstd decode                 | 693 MB/s   | 22 µs     |

- **The data is synthetic**: A smooth sine compresses better than a real sensor. Treat
  the ratios as an upper bound.
- **The host was loaded**: The load average was 40 on one Apple silicon Mac, so the
  throughput numbers are low. Compare them with each other only.
- **Index encoding pays only on narrow frames**: It saves 15 to 18 percent on 10
  channels and 2 percent on 100.
- **Shuffle is the main lever**: Plain zstd or deflate gains 1.1 to 1.3 times on scaled
  data. Shuffling adds 0.3 to 0.7.
- **Small frames do not compress**: A 213 B frame grows under zstd.

## 3 Decisions

1. **Measure real frames first**: The Core reports encoded frame bytes and the index
   share per stream. Phase 8 waits for that data.
2. **No default compression on live streams**: Deflate runs at 53 MB/s and gains less
   than zstd. Most Cores run on a LAN, where compression costs more than it saves.
3. **zstd for frame reads, when asked**: The Core compresses a `/frame/read` frame body
   with zstd level 1 when the request sends `Accept-Encoding: zstd`. Chrome 123, Firefox
   126, and Safari 26.3 decode it natively. A client that does not ask gets the same
   body as today, so shipped clients do not change.
4. **A constant-step index in a new codec version**: The encoder writes an index series
   as start and step when every delta is equal, and as raw samples otherwise. It uses
   the reserved extended flags byte (`codec.go`, bit 7) and a new content type,
   `application/vnd.synnax.frame.v2`. The Core keeps the current codec for shipped
   clients, as the release model requires for stable releases.
5. **A compressed stream codec only on proof**: An opt-in content type that shuffles and
   then zstd-compresses each frame ships only when Phase 1 data from a real Core shows
   at least 1.5 times on its streams, and a user on a link under 100 Mbit/s needs it.
6. **The Driver path is separate**: The Driver talks gRPC. gRPC compression is gzip and
   deflate only, so a Driver change needs its own RFC.

## 4 Implementation phases

Each phase merges into `main` alone. A client phase follows its Core phase.

| Phase | Scope                                                 | Touches                                                    | Depends on |
| ----- | ----------------------------------------------------- | ---------------------------------------------------------- | ---------- |
| 1     | Report encoded frame sizes in the Core                | `core/pkg/api/framer`, `freighter/go/http`                 | -          |
| 2     | Serve zstd frame reads when the client accepts it     | `freighter/go/http`, `core/pkg/transport/http`             | -          |
| 3     | Accept zstd frame reads in the Python client          | `freighter/py`, `client/py`                                | 2          |
| 4     | Add a constant-step index to the frame codec          | `core/pkg/distribution/framer/codec`, `core/pkg/transport` | -          |
| 5     | Decode constant-step indexes in the TypeScript client | `client/ts/src/framer`                                     | 4          |
| 6     | Decode constant-step indexes in the Python client     | `client/py/synnax/framer`                                  | 4          |
| 7     | Decode constant-step indexes in the C++ client        | `client/cpp/framer`                                        | 4          |
| 8     | Add an opt-in shuffle and zstd stream codec           | `core/pkg/transport`, `client/ts`, `client/py`             | 1, 4       |

## 5 Open questions

- **Python on 3.13**: urllib3 decodes zstd natively on Python 3.14. On 3.13 it needs the
  `zstandard` package. Phase 3 either adds it as a dependency or asks for zstd only on
  3.14 and later.
- **Linux Console**: The WebKitGTK version that decodes zstd is not verified. The
  Console falls back to an uncompressed body when the WebView does not ask.
