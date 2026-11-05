# 98 Video

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-02
- **Related**:
  [RFC 0035 - Cesium variable-length storage](0035-cesium-variable-length-storage.md),
  [RFC 0032 - Telemetry bypass](0032-telemetry-bypass.md),
  [RFC 0064 - MQTT integration](0064-mqtt-integration.md),
  [RFC 0083 - Telemetry layers](0083-telemetry-layers.md)

## 0 Summary

Synnax stores and streams video as channels. A new `video` data type holds one encoded
frame per sample. IP cameras (RTSP) and encoders such as OBS (WHIP) feed video channels
through tasks that run in the Core, and the clients can write frames directly. Cesium
stores video in a mode of the unary DB that indexes keyframes. Each streamer gets its
own queue, so a slow viewer loses frames instead of blocking writers or other viewers.
The Console shows a video channel live, or plays back a range, in a video view that
decodes with WebCodecs. Retention, set on an index channel, deletes old data from every
channel on that index. A video channel exports to MP4.

## 1 Vocabulary

- **Frame**: One encoded picture, stored as one `video` sample.
- **Keyframe**: A frame that decodes without earlier frames. For H.264, an IDR access
  unit with its SPS and PPS.
- **GOP**: A keyframe and the frames after it, up to the next keyframe.
- **Viewer**: One streamer that a client opens.
- **Gap notice**: A field in a streamer response that names the channels whose frames
  the Core dropped for that viewer.

## 2 Principles

1. **Video is a channel**: Ranges, permissions, streaming, deletes, and export work on
   video because they work on channels. There is no second store.
2. **Writers never wait for viewers**: A slow viewer loses its own frames and is told
   so. It never delays writers or other viewers (RFC 0012 §2.3, RFC 0032 §2.0).
3. **Generic layers stay generic**: The distribution layer knows nothing about codecs
   (RFC 0003 §3.0). Video behavior is a policy that the service layer injects, or a
   Cesium storage mode that the data type selects (RFC 0035 §3.3).
4. **Ingest never decodes**: The Core moves encoded frames. Only the viewer decodes.
5. **The `Feed` owns time in the Console**: Video reads go through the `Feed`, like all
   other telemetry reads (RFC 0083 §2).

## 3 Design

### 3.0 The `video` data type

`video` is a variable-density data type in Go, TypeScript, Python, and C++. Cesium and
the frame codecs branch on `IsVariable()` (`x/go/telem/versions/v0/data_type.go:32`), so
storage and the wire need no new paths. Each sample is a 2-byte header and one H.264
access unit in Annex B format:

| Byte | Field       | Values                                      |
| ---- | ----------- | ------------------------------------------- |
| 0    | Flags       | Bit 0 is set on a keyframe. Bits 1-7 are 0  |
| 1    | Codec       | 1 is H.264                                  |
| 2-   | Access unit | Annex B. A keyframe carries its SPS and PPS |

One sample decodes to exactly one picture, and decode order equals display order, so a
stream has no B-frames. This is the key chunk that WebCodecs needs (AVC codec
registration) and the frame that Foxglove's `CompressedVideo` defines. The viewer reads
its codec string from the SPS of the first keyframe.

Each language's `Series` gets frame constructors and a frame accessor. Today `Series.at`
parses `bytes` samples as JSON (`x/ts/src/telem/series.ts:851-856`), and Python cannot
build a `bytes` series from a list (`x/py/x/telem/series.py:62`). Two message limits
rise with the type: the Python client refuses WebSocket messages over 5 MB
(`client/py/synnax/transport.py:65`), and peer taps between Cores use the 4 MiB gRPC
default (`core/pkg/server/grpc.go:42`). Both rise to 64 MB, which holds a frame with
several keyframes.

### 3.1 Storage

Video channels are unary channels. `newOffsetResolver` already picks a mode by data type
(`cesium/internal/unary/resolver.go:160`), and `video` adds a third mode. Its table
holds each frame's offset and keyframe flag. The writer fills the table as it writes. On
a cold read, the mode rebuilds a domain's table from each frame's 4-byte length and
2-byte header only. The variable mode reads every byte of the domain today
(`resolver.go:83`), up to 1 GB.

A new unary iterator command, `SeekKeyframe`, puts the view at the last keyframe at or
before a time. It reads only the table, and fails on a channel that is not `video`. The
command goes through the distribution and service iterators, the API, and the TypeScript
and Python iterators, next to `seekLE` (`client/ts/src/framer/iterator.ts:182`). A seek
over channels on several Cores fails on a Core that holds none of the video channels.

Three fixes apply to all variable-density channels:

- A delete invalidates only the domains it changed. Garbage collection invalidates
  nothing, because it copies each domain whole. Today each one clears the whole cache
  (`cesium/internal/unary/delete.go:39-42,102`), and garbage collection runs every 30 s.
- One cache serves all channels of a Cesium DB and evicts under a 128 MB budget. Today
  each channel has its own cache, and it never evicts.
- Reads chunk variable data by bytes. Today they chunk by sample count
  (`defaultChunkSize`, `cesium/relay.go:47`), so a span of video comes back whole.

The table is not saved to disk. RFC 0035 §4.0 and §4.1 rejected an offset file and a
trailer, and the header-only scan makes a cold rebuild cheap.

### 3.2 Streaming

Today a stalled viewer delays each frame for all viewers by up to 20 ms
(`x/go/confluence/source.go:55-80`). If it stays stalled, the backlog reaches Cesium and
blocks writers (`cesium/options.go:109-111`). Drops go only to the log.

- **Viewer queue**: A new `confluence` segment queues without blocking and drops by a
  policy that its owner injects. It is the last stage of the service streamer
  (`core/pkg/service/framer/streamer/streamer.go:158-191`). The API streamer's buffer of
  200 responses (`core/pkg/api/framer/framer.go:428-432`) shrinks to 1, so this queue is
  the only place a viewer's backlog grows. The queue holds 16 MB. The relay's peer
  server (`core/pkg/distribution/framer/relay/server.go:36`) uses the same segment, so a
  slow link between Cores never stalls the remote relay.
- **Drop policy**: The default drops the oldest responses. For a video channel, the
  service layer drops all of that channel's queued frames and passes none until the next
  keyframe. The same rule applies after a gap notice from upstream.
- **Gap notice**: A streamer response gains `dropped`, the keys of channels that lost
  frames before it. The TypeScript, Python, and C++ streamers expose it. The `Feed`
  reads the lost span of a persisted channel again from history. A client that keeps
  state from a virtual channel, such as control state (RFC 0032 §0.3.2), retrieves it
  again. The Driver task manager does this for control state and its rack name. Task
  events carry no state to retrieve, so the Driver logs their loss.
- **Keyframe cache**: The service layer follows every video channel, the way calculation
  groups follow their inputs (`core/pkg/service/framer/calculation/group.go:94-97`), and
  keeps the frames since the last keyframe. A new viewer gets those frames first, then
  live frames. A GOP longer than the bound empties the cache until the next keyframe, as
  SRS bounds its GOP cache. Without the cache, a new viewer waits up to one keyframe
  interval: about 8 s with the OBS x264 defaults.

### 3.3 Ingest

Every source produces frames with a keyframe flag and an arrival time. A new general
package, `x/go/video`, assembles access units, checks them, and times them. Each
protocol is an adapter that feeds it. Parsing uses `bluenviron/mediacommon`.

- A stream with B-frames fails its task with an error that names the encoder setting.
- A stream with no IDR for 10 s, such as periodic intra refresh, gets a warning status,
  because no viewer can start on it.
- Parameter sets from the session description go in front of each IDR that lacks them.

Both protocols run as tasks on the Core's rack, like MQTT
(`core/pkg/service/driver/factory.go:24`, `core/pkg/service/mqtt/factory.go:146`). The
libraries use the MIT license and need no cgo, which the Core's Docker build forbids
(`core/Dockerfile:87-90`).

| Task type   | Package                 | Library                | Connects    |
| ----------- | ----------------------- | ---------------------- | ----------- |
| `rtsp_read` | `core/pkg/service/rtsp` | `bluenviron/gortsplib` | Core, out   |
| `whip_read` | `core/pkg/service/whip` | `pion/webrtc`          | Encoder, in |

**WHIP**: Signaling is `POST`, `PATCH`, and `DELETE` on `/api/v1/whip/{task}`. The
Freighter router binds only GET and POST (`freighter/go/http/router.go:99-111`), so
these are raw Fiber routes placed before the embedded Console, which binds raw routes
already (`core/pkg/console/console.go:80`). Each route runs a Freighter server, so the
API layer gives it the recovery, instrumentation, and license middleware. The encoder
sends an API key secret as its bearer token (`schemas/synnax/apikey.oracle`), so the
token middleware does not apply. The API service (`core/pkg/api/whip`) checks the key
and its subject's permission to write the task's channel. All WHIP sessions send media
to one UDP port through Pion's `SetICEUDPMux`, set by a new Core flag. The Core has no
UDP socket today.

**Timestamps**: A persisted video channel has its own index. A frame's time is its
arrival time at the Core, smoothed by the RTP timestamp spacing, plus a fixed `offset`
from the task config that removes the encode and network delay. With
`time_source: "capture"`, a frame's time comes from RTCP sender reports instead, for
cameras that NTP or PTP keeps in sync, and the task drops frames until the first report.
Each session starts a new clock and drops the frames before its first keyframe, which
would decode against the last session's pictures. Cesium rejects a time that does not
increase, and the task then fails. A virtual channel cannot have an index
(`cesium/internal/channel/channel.go:119-121`), so virtual frames carry no time.

**Channels**: The Console creates a task's video channel as virtual when data saving is
off, and as a persisted channel with its own index otherwise. Today
`Task.configureReadChannels` makes every variable-type channel virtual
(`console/src/platform/task/channels.ts:149`). Client code writes frames with the normal
writer and the constructors from §3.0.

### 3.4 Retention

An index channel gains `retention`, a time span. Zero keeps data forever. It applies to
the index and to every data channel on it, because Cesium deletes an index range only
after its data channels (`cesium/delete.go:232-260`). The field is a new version of the
stored channel (`schemas/synnax/channel.oracle:96`), and the Console channel form sets
it. A job on `signal.GoTick` (`x/go/signal/go.go:62`) runs on each Core for the index
channels it leases, and deletes data older than the retention span.

Today a running writer claims from its domain start to the end of time
(`cesium/internal/unary/writer.go:133-137`), and only `EndDomain` moves the claim
(`:392`). With this RFC, a file switch also moves the claim to the last committed
sample, 1 ns before the new domain's start. A writer that stamps its commits from a
start time stamps from that sample after each commit, so a delete cannot remove it.
Retention deletes up to its cutoff or the oldest claim on the index, whichever is
earlier. A channel that writes slowly switches files rarely, so retention frees its
space only at its next switch.

The Core sets `retention` only when it creates a channel, because its create path
updates only calculated channels. It rejects a negative retention, and a retention on a
data or virtual channel. The job lives in `core/pkg/service/framer/retention` and runs
once a minute.

### 3.5 Console video view

A video view is a Core resource, like a log: `schemas/synnax/video.oracle`,
`core/pkg/service/video`, `client/ts/src/video`, `pluto/src/video`, and
`console/src/feature/video`, registered at the same Console sites as the log. It holds
one channel and one range reference, live or fixed, like a line plot
(`schemas/synnax/lineplot.oracle:287`).

The view decodes in the Aether worker with a WebCodecs `VideoDecoder`, configured with
no description so that it reads Annex B. It draws each frame into its region with the
9-argument `drawImage` (`pluto/src/vis/draw2d/canvas.ts:238`), with image smoothing on
(`pluto/src/vis/render/context.ts:40` turns it off). It keeps the newest picture until
the next one, so a resize draws it again.

Frames come from the `Feed` (`client/ts/src/framer/feed.ts:217`) through a new `frames`
subscription: every sample of a channel in a window, from the keyframe at or before the
window start. Video raw blocks use a shorter tile than the 65.5 s numeric tile
(`client/ts/src/framer/cache/block.ts:18`), so one block of a 10 Mbit/s stream cannot
fill most of the 128 MB budget (`cache/store.ts:59`).

- **Live**: The view reads the `latest` subscription and decodes the frames after its
  cursor. After a gap notice, it waits for the keyframe that the Core sends next.
- **Playback**: The view plays, pauses, scrubs, and changes speed over the range, from
  committed frames only. A seek starts the decoder at the keyframe before the cursor and
  decodes forward to it. The view draws on each animation frame while it plays, and a
  cursor before the first frame shows no picture.

WebCodecs needs a secure context. The desktop Console has one. The web Console served
over plain HTTP from another host does not, and the view says so.

### 3.6 MP4 export

`/api/v1/frame/read` gains a `video/mp4` encoder next to CSV and TDMS
(`core/pkg/transport/http/http.go:178-190`). It takes one video channel, starts at the
first keyframe in the range, and streams fragmented MP4 from `mediacommon` with no
re-encoding. The muxer lives in `x/go/video`. A read with a reduction, or a range with
no keyframe, fails before the first byte. The TypeScript and Python clients download it,
and the video view toolbar downloads the view's range.

## 4 What this RFC does not cover

- Video inside schematics, and a playback clock shared by several visualizations.
- RTMP and SRT sources.
- MP4 import, because Synnax has no telemetry import path (RFC 0059 §7).
- Encoding raw frames from USB or GenICam cameras, transcoding, and audio.
- Codecs other than H.264.

## 5 Implementation phases

Virtual video ships first. It needs the type, the streaming changes, the tasks, and the
live view, but no storage work. The Console hides video behind `FLAGS.video` until
Phase 10.

| Phase | Scope                                                  | Touches                                                                                                                                                                                   | Depends on |
| ----- | ------------------------------------------------------ | ----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Add the `video` data type and frame accessors          | `x/go/telem`, `x/ts/src/telem`, `x/py/x/telem`, `x/cpp/telem`, `client/py/synnax/transport.py`, `core/pkg/server`                                                                         | -          |
| 2     | Queue each streamer without blocking the relay         | `x/go/confluence`, `core/pkg/service/framer/streamer`, `core/pkg/api/framer`, `core/pkg/distribution/framer/relay`                                                                        | -          |
| 3     | Send gap notices to streamers                          | `core/pkg/service/framer/streamer`, `core/pkg/transport/http/framer`, `client/ts/src/framer`, `client/py/synnax/framer`, `client/cpp/framer`, `driver/task`                               | 2          |
| 4     | Drop video to the next keyframe and cache the last GOP | `core/pkg/service/framer/streamer`, `core/pkg/service/framer`                                                                                                                             | 1, 3       |
| 5     | Add video ingest and the RTSP read task                | `x/go/video`, `core/pkg/service/rtsp`, `schemas/synnax/rtsp.oracle`, `core/pkg/service/layer.go`                                                                                          | 1          |
| 6     | Add the WHIP read task                                 | `core/pkg/service/whip`, `core/pkg/api/whip`, `core/pkg/transport/http/whip`, `schemas/synnax/whip.oracle`, `core/cmd/start`                                                              | 5          |
| 7     | Configure RTSP and WHIP tasks in the Console           | `console/src/feature/rtsp`, `console/src/feature/whip`, `console/src/flags.ts`                                                                                                            | 5, 6       |
| 8     | Add the video view resource                            | `schemas/synnax/video.oracle`, `core/pkg/service/video`, `core/pkg/api/video`, `client/ts/src/video`                                                                                      | -          |
| 9     | Decode video from the Feed in Pluto                    | `client/ts/src/framer/cache`, `pluto/src/video`                                                                                                                                           | 1, 4       |
| 10    | Show live video in a Console video view                | `console/src/feature/video`, `console/src/session/video`, `console/src/flags.ts`                                                                                                          | 7, 8, 9    |
| 11    | Scope offset cache invalidation and evict by size      | `cesium/internal/unary`, `cesium`                                                                                                                                                         | -          |
| 12    | Index keyframes and seek to them in Cesium             | `cesium/internal/unary`, `cesium`                                                                                                                                                         | 1, 11      |
| 13    | Seek to keyframes through the API and clients          | `core/pkg/distribution/framer/iterator`, `core/pkg/service/framer/iterator`, `core/pkg/api/framer`, `client/ts/src/framer`, `client/py/synnax/framer`, `client/cpp/framer`, `driver/task` | 12         |
| 14    | Store and timestamp video from RTSP and WHIP tasks     | `x/go/video`, `core/pkg/service/rtsp`, `core/pkg/service/whip`, `console/src/platform/task`                                                                                               | 7          |
| 15    | Play back video ranges in the video view               | `client/ts/src/framer/cache`, `pluto/src/video`, `console/src/feature/video`                                                                                                              | 10, 13, 14 |
| 16    | Move a running writer's claim at each file switch      | `cesium/internal/unary`, `cesium/internal/domain`                                                                                                                                         | -          |
| 17    | Delete old data by index channel retention             | `schemas/synnax/channel.oracle`, `core/pkg/service/channel`, `core/pkg/service/framer`, `client/ts/src/channel`, `client/py/synnax/channel`, `console/src/platform/channel`               | 16         |
| 18    | Export video channels to MP4                           | `core/pkg/transport/http/framer`, `client/ts/src/framer`, `client/py/synnax/framer`, `console/src/feature/video`                                                                          | 12, 15     |

Every phase is compatible with shipped releases. Phase 3 adds a response field, Phase 13
adds an iterator command, and Phase 17 adds a channel version with a migration.

## 6 Resolved decisions

1. **Channels, not a media store**: MediaMTX and Frigate keep MP4 segments with their
   own index, which would need their own streaming, permissions, deletes, and sync with
   telemetry. Foxglove, Rerun, and ROS 2 store one frame per record and seek by
   keyframe, as this RFC does. The trade is real: a segment file opens in any video
   tool.
2. **A unary mode, not a third DB type**: A sibling of unary and virtual repeats the
   writer, iterator, control, and delete code, which RFC 0035 §4.5 rejected for
   variable-length data, and touches about 80 dispatch sites in seven Cesium files. The
   trade is real: a separate DB could own its file layout and save its table to disk.
3. **A data type, not a channel field**: Cesium sees the data type, so it selects the
   storage mode with no new plumbing. A `bytes` channel with a field needs a channel
   migration, and `Series.at` would still parse its frames as JSON.
4. **The queue ends the service streamer, not the relay**: The relay has no channel
   types (RFC 0003 §3.0), so it cannot apply the video policy. A queue in the relay
   alone also leaves 200 responses of backlog after it, about 7 s of video at 30 fps.
5. **Drop and notify, not disconnect**: A closed stream reconnects and falls behind
   again. A gap notice keeps the stream and lets the `Feed` repair stored channels.
6. **Core tasks, not Driver tasks**: The mature RTSP and WebRTC libraries are pure Go.
   In the Driver they would need FFmpeg or live555 in Bazel. The trade is real (RFC 0064
   §4.0): the Core must reach the camera, and a Driver alone cannot run these tasks.
7. **RTSP and WHIP, not RTMP and SRT**: Both carry RTCP sender reports for capture time.
   RTMP and SRT carry only stream-relative time, OBS sends B-frames over RTMP by
   default, and each needs another port.
8. **Arrival time by default**: Capture time is correct only when the camera clock is in
   sync. Arrival time always increases, and its delay is near constant, so the offset
   removes it. Foxglove plays by log time, and ROS stamps acquisition time.
9. **No B-frames**: Foxglove and Rerun reject them too. Reordering adds a decode time to
   every frame and delays playback.
10. **Retention on the index channel**: Cesium deletes an index range only after its
    data channels, so the index is the unit. Kafka sets retention per topic, and
    TimescaleDB per table.
11. **WebCodecs, not a `<video>` element**: Media Source Extensions need fMP4 and draw
    outside the Aether canvas, and WebRTC playback covers only live video.
12. **Through the `Feed`, not around it**: RFC 0083 §2 keeps one join of history and
    live data, and one cache.

## 7 Open questions

- The bound of the keyframe cache, in frames and bytes.
- The name and default of the WHIP UDP port flag.
- The tile length for video raw blocks, and whether video gets its own `Store` budget.
