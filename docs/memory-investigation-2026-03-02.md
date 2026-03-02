# Memory Investigation: 24 GB RSS on Synnax Server

**Date**: 2026-03-02
**Build**: `synnax-server.exe` built 2026-02-17
**Observation**: Task manager shows 24 GB memory usage. Team goes to lunch, heap clears.
Memory returns when clients reconnect.

## Profile Summary

Two pprof profiles were collected:

| Profile | Type | Total |
|---------|------|-------|
| `heap` | `inuse_space` | 11.9 GB (but 96% is pprof self-measurement artifact) |
| `allocs` | `alloc_space` | 1.77 TB lifetime allocations |

The heap profile is mostly useless — 11.5 GB is the Fiber pprof middleware
(`fasthttp/fasthttpadaptor.(*writer).Write`) buffering the profile response itself.
Real in-use heap is ~245 MB (Pebble memtables).

The 24 GB RSS is explained by **Go holding freed heap pages** due to massive allocation
churn. The runtime requests memory from the OS but returns it slowly via
`MADV_FREE`. When allocation pressure stops (lunch break), the scavenger catches up and
RSS drops.

## Root Cause: cockroachdb/errors Formatting — 608 GB (34% of all allocations)

The single largest allocation source is `cockroachdb/errors/errbase.formatRecursive`
at **608 GB cumulative**. This is triggered by two paths:

### Path 1: `errors.Is` marker fallback — 162 GB

When cockroachdb's `errors.Is(err, ref)` can't match by pointer in the unwrap chain,
it falls back to marker-based comparison (lines 85-90 of `markers/markers.go`). This
calls `getMark()` → `safeGetErrMsg()` → `.Error()` on the error, triggering full
recursive formatting of the entire error tree.

**Callers of `errors.Is` that trigger this** (from `pprof -peek`):

| Caller | Allocs | % of errors.Is total |
|--------|--------|---------------------|
| `x/query.encode` | 107 GB | 66% |
| `cesium.(*streamWriter).maybeSendRes` | 27 GB | 17% |
| `x/control.encode` | 27 GB | 16% |
| `cesium.(*streamWriter).write` | 1.5 GB | 1% |

### Path 2: Direct `.Error()` calls on joinErrors — 139 GB

`joinError.Error()` is called from `withStack.Error()` (139 GB) when error payloads
are being constructed. This happens primarily during `errors.Encode()`.

### Nested joinErrors amplify the cost

The stack traces show `joinError.SafeFormatError` calling itself **8+ levels deep**.
`errors.Combine` / `errors.Join` don't flatten — each call wraps the previous combined
error, creating a deeply nested tree. When `.Error()` is finally called, it recursively
formats every level.

## The Specific Hot Path: Frame Writer Error Accumulation

**Profile evidence** (`pprof -peek`):

```
errors.Encode callers:
  140164.22 MB (99.99%)  api.(*FrameService).Write.func2
```

Almost ALL error encoding comes from the frame writer response path.

### How it works

1. `cesium/writer_stream.go` — `streamWriter.Flow()` (line 116-147):
   - Receives write requests in a loop
   - If `accumulatedErr` is set, **skips processing** but still sends a response
   - `maybeSendRes` sets `res.Err = w.accumulatedErr` on every response

2. `cesium/writer_stream.go` — `maybeSendRes()` (line 211-226):
   - Sets `res.Err = w.accumulatedErr` (line 221)
   - Only skips sending if `res.Err == nil && command == Write && !Sync` (line 222)
   - When `accumulatedErr != nil`, **every response is sent with the error**

3. `core/pkg/api/framer/framer.go` — Write handler (line 343-351):
   - TransformSender calls `errors.Encode(ctx, i.Err, false)` on EVERY response
   - This is where the 140 GB of encode allocations come from

4. `x/query/errors.go` — `encode()` (line 37-51):
   - Calls `errors.Is` up to 4 times sequentially per error
   - Each `errors.Is` on a joinError triggers expensive marker comparison
   - Then calls `err.Error()` for the payload data

### The amplification math

- Client writes frames at high frequency (thousands/sec)
- Writer hits one error → `accumulatedErr` is set
- Every subsequent write request gets a response with the same error
- Each response → `errors.Encode` → `query.encode` → 4x `errors.Is` → marker fallback
  → recursive `joinError.SafeFormatError`
- Thousands of times per second, encoding the same error

## Other Significant Allocators

| Source | Flat Allocs | Notes |
|--------|-------------|-------|
| `bytes.growSlice` | 122 GB | Buffer growth from string/error formatting |
| `arc/runtime/wasm.(*nodeImpl).Next` | 115 GB | Arc calculator wasm execution |
| `gopsutil/net.getTCPConnections` | 64 GB | Fiber reading /proc/net/tcp |
| `framer/codec.(*Codec).DecodeStream` | 57 GB | Frame deserialization |
| `protobuf consumeBytesNoZero` | 57 GB | gRPC protobuf decode |
| `zap stacktrace.Capture` | 55 GB | Logger stack capture |
| `aspen/internal/kv.storeState.Copy` | 50 GB | Aspen KV state snapshots |
| `redact/internal/buffer.makeSlice` | 47 GB | Inside error formatting |
| `msgpack/v5.NewDecoder` | 43 GB | Msgpack decoder creation |
| `driver/internal/log.PipeToLogger` | 42 GB | Driver subprocess log piping |
| `errors/withstack.(*stack).StackTrace` | 37 GB | Stack trace capture in error wrapping |
| `cesium/internal/domain.(*pointerCodec).encode` | 36 GB | Cesium domain encoding |

## Proposed Fixes

### Fix 1: Don't re-encode the same error on every response

The writer should encode the error once when `accumulatedErr` is first set, and cache
the payload. Subsequent responses should send the cached payload instead of re-encoding.

**Files**: `cesium/writer_stream.go`, potentially `core/pkg/api/framer/framer.go`

### Fix 2: Make error encoding cheaper

The `query.encode`, `control.encode`, and `validate.encode` functions use
`cockroachdb/errors.Is` (via `x/errors.Is`) for sentinel checks. These can safely use
stdlib `errors.Is` instead — the sentinels (`ErrNotFound`, `ErrUnauthorized`, etc.) are
in the unwrap chain, so stdlib pointer comparison works.

**Files**:
- `x/go/query/errors.go` — `encode()` function
- `x/go/control/` — `encode()` function
- `x/go/validate/` — `encode()` function
- Any other registered error encoders

Note: changing `x/errors.Is` globally breaks tests that compare two independent
`errors.New("same message")` instances (cockroachdb matches by message marker, stdlib
matches by pointer). The encode functions are safe because they check against
package-level sentinel variables.

### Fix 3 (investigation needed): What error is the writer accumulating?

The writer error accumulation is the trigger. Understanding what error the writer hits
would reveal whether there's an underlying bug causing the error in the first place.

## Key Files

| File | Relevance |
|------|-----------|
| `cesium/writer_stream.go` | Writer error accumulation and response sending |
| `core/pkg/api/framer/framer.go` | API layer error encoding per response |
| `x/go/errors/encode.go` | Error registry and encode/decode |
| `x/go/errors/errors.go` | Thin wrapper around cockroachdb/errors |
| `x/go/query/errors.go` | query.encode — 66% of errors.Is cost |
| `x/go/control/` | control.encode — 16% of errors.Is cost |
| `cockroachdb/errors@v1.12.0/markers/markers.go` | The expensive Is implementation |

## How to Verify

After fixes, re-profile with:
```
go tool pprof http://<server>/debug/pprof/allocs
```

Check that:
1. `cockroachdb/errors/errbase.formatRecursive` drops significantly
2. `joinError.SafeFormatError` drops significantly
3. RSS stabilizes at a lower level under sustained write load with errors

Also check `runtime.ReadMemStats` for `HeapSys` vs `HeapInuse` vs `HeapReleased` to
confirm the RSS-vs-heap theory.
