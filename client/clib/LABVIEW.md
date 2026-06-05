# LabVIEW Spike: Connect + Write

End-to-end walkthrough for driving `synnax_clib.dll` from LabVIEW: open a client, write
float64 samples, close. The public API is a flat C surface (the 7 `synnax_*` exports),
so every Call Library Function Node (CLFN) parameter is a plain scalar, string, array,
or opaque handle — no pointer-bearing structs.

## Scope / constraints (spike)

- **64-bit only.** DLL bitness must match LabVIEW (use 64-bit LabVIEW).
- **Insecure connection** for the demo: wire `secure = 0` and `ca_cert_file = ""`. (TLS
  works via `secure = 1` + a CA file path, but the spike runs against an insecure Core.)
- **float64 data only**, single channel per write (`synnax_writer_write`).
- **Demo writes stream-only to a virtual channel** (`mode = 3`). No index/timestamp
  column is needed and no commit is required. Observe the samples live in a Console line
  plot subscribed to the channel.
- Persisted writes (`mode = 1` or `2`) additionally require writing the channel's index
  timestamps, which the float64 `write` does not cover yet — that's a later wrapper.
- **Errors:** pass `0` (NULL) for every `err` parameter for now; check only the `int32`
  return (0 = ok). `SynnaxError` parsing is a follow-up.

## 1. Build & export the DLL (Windows)

From `C:\synnax`, in a "Developer PowerShell for VS" (so `dumpbin` is on PATH):

```powershell
bazel --output_user_root=C:/tmp build //client/clib:synnax_clib --config=windows
```

Locate and verify the DLL:

```powershell
$bin = bazel --output_user_root=C:/tmp info --config=windows bazel-bin
$dll = Join-Path $bin "client\clib\synnax_clib.dll"
dumpbin /exports $dll | Select-String synnax_         # expect 7 synnax_* symbols
dumpbin /headers $dll | Select-String machine         # expect: 8664 (x64)
```

Copy it somewhere stable for LabVIEW to load:

```powershell
Copy-Item $dll C:\synnax\labview\synnax_clib.dll
```

The 7 exports: `synnax_client_open`, `synnax_client_close`, `synnax_client_version`,
`synnax_writer_open`, `synnax_writer_write`, `synnax_writer_commit`,
`synnax_writer_close`.

## 2. Pre-create the test channel (once)

In the Console (or Python), create a **virtual float64** channel and note its numeric
**key** (a `uint32`) — that's what LabVIEW passes, not the name.

- Console: Channels → New → name `lv_test`, data type `float64`, toggle **Virtual** on.
- Get the key from the channel's details, or in Python:

```python
import synnax as sy
c = sy.Synnax()  # localhost:9090 synnax/seldon
print(c.channels.retrieve("lv_test").key)
```

Keep a Console **line plot** open on `lv_test` to watch samples arrive.

## 3. LabVIEW CLFN configuration

One VI per function. For each Call Library Function Node:

- **Library name/path:** `C:\synnax\labview\synnax_clib.dll`
- **Function name:** as below
- **Calling convention:** **C**
- **Thread:** **Run in any thread** (these calls block on the network)

`UPtr` = Numeric → _Unsigned Pointer-sized Integer_. **Opaque handles**
(`SynnaxClient*`, `SynnaxWriter*`) and the `err` pointer are all `UPtr`.

**Wizard gotchas** (the Import Shared Library wizard guesses these wrong):

- It maps opaque handles as empty **Clusters** — change each to **UPtr**, so the handle
  from `*_open` wires straight into the next call.
- It maps `SynnaxError *err` as a 3-field cluster — that corrupts memory; for now set it
  to **UPtr** and wire **`0`** (NULL).
- "Apply to All Matching Parameters" may not appear — fix each handle one at a time.

### `synnax_client_version` → String (smoke test, no client needed)

| #   | Param  | LabVIEW type | Pass                 |
| --- | ------ | ------------ | -------------------- |
| ret | return | String       | **C String Pointer** |

Call it first to confirm the DLL loads and is the right bitness — it returns the version
string (e.g. `0.50.0`) with no inputs.

### `synnax_client_open` → I32

| #   | Param        | LabVIEW type             | Pass                       |
| --- | ------------ | ------------------------ | -------------------------- |
| ret | return       | Numeric, Signed 32-bit   | Value                      |
| 1   | host         | String                   | C String Pointer           |
| 2   | port         | Numeric, Unsigned 16-bit | Value                      |
| 3   | username     | String                   | C String Pointer           |
| 4   | password     | String                   | C String Pointer           |
| 5   | secure       | Numeric, Signed 32-bit   | Value (wire `0` insecure)  |
| 6   | ca_cert_file | String                   | C String Pointer (wire "") |
| 7   | out_client   | Numeric, UPtr            | **Pointer to Value**       |
| 8   | err          | Numeric, UPtr            | Value (wire `0`)           |

Wire `host=""`, `port=0`, `username=""`, `password=""` to take dev defaults
(localhost:9090 synnax/seldon). The client handle comes out of param 7.

### `synnax_client_close` → void

| #   | Param  | LabVIEW type  | Pass  |
| --- | ------ | ------------- | ----- |
| 1   | client | Numeric, UPtr | Value |

### `synnax_writer_open` → I32

| #   | Param              | LabVIEW type                | Pass                             |
| --- | ------------------ | --------------------------- | -------------------------------- |
| ret | return             | Numeric, Signed 32-bit      | Value                            |
| 1   | client             | Numeric, UPtr               | Value                            |
| 2   | start              | Numeric, Signed 64-bit      | Value (wire `0` for stream-only) |
| 3   | channels           | Array, Unsigned 32-bit, 1-D | **Array Data Pointer**           |
| 4   | channel_count      | Numeric, UPtr               | Value (wire array size)          |
| 5   | mode               | Numeric, Signed 32-bit      | Value (wire `3` = stream)        |
| 6   | enable_auto_commit | Numeric, Signed 32-bit      | Value (wire `0`)                 |
| 7   | out_writer         | Numeric, UPtr               | **Pointer to Value**             |
| 8   | err                | Numeric, UPtr               | Value (wire `0`)                 |

Build a U32 array with the single `lv_test` key for param 3; wire its size to param 4.
For `mode`, see the **Writer mode Ring** below.

### `synnax_writer_write` → I32

| #   | Param        | LabVIEW type              | Pass                      |
| --- | ------------ | ------------------------- | ------------------------- |
| ret | return       | Numeric, Signed 32-bit    | Value                     |
| 1   | writer       | Numeric, UPtr             | Value                     |
| 2   | channel      | Numeric, Unsigned 32-bit  | Value (the `lv_test` key) |
| 3   | data         | Array, 64-bit double, 1-D | **Array Data Pointer**    |
| 4   | sample_count | Numeric, UPtr             | Value (wire array size)   |
| 5   | err          | Numeric, UPtr             | Value (wire `0`)          |

### `synnax_writer_commit` → I32 (not needed for the stream-only demo)

| #   | Param      | LabVIEW type           | Pass                 |
| --- | ---------- | ---------------------- | -------------------- |
| ret | return     | Numeric, Signed 32-bit | Value                |
| 1   | writer     | Numeric, UPtr          | Value                |
| 2   | out_end_ts | Numeric, Signed 64-bit | **Pointer to Value** |
| 3   | err        | Numeric, UPtr          | Value (wire `0`)     |

Only needed for persisted writes when `enable_auto_commit = 0`. Skip it for the
stream-only demo.

### `synnax_writer_close` → I32 (closes **and** frees)

| #   | Param  | LabVIEW type           | Pass             |
| --- | ------ | ---------------------- | ---------------- |
| ret | return | Numeric, Signed 32-bit | Value            |
| 1   | writer | Numeric, UPtr          | Value            |
| 2   | err    | Numeric, UPtr          | Value (wire `0`) |

There is no separate `free` — `close` flushes, returns the accumulated error, and frees
the handle in one call.

## Writer mode Ring

Make a typedef'd **Ring** control (not an Enum — LabVIEW Enums are 0-indexed and can't
hold arbitrary values) wired to the I32 `mode` terminal, mirroring `SynnaxWriterMode`:

| Item           | Value |
| -------------- | ----- |
| Default        | 0     |
| Persist+Stream | 1     |
| Persist        | 2     |
| Stream         | 3     |

The demo uses **Stream (3)**.

## 4. Demo VI wire order

Chain the handles left to right (use the error/return I32 as your sequencing wire):

```
client_open ──► client handle ─┬─► writer_open ──► writer handle ──► write (loop) ──► writer_close
                               │
                               └────────────────────────────────────────────────► client_close
```

1. `synnax_client_open` (host="", port=0, user="", pass="", secure=0, ca="") → client
   handle.
2. `synnax_writer_open` (client, start=0, [lv_test key], count=1, mode=3, auto_commit=0)
   → writer handle.
3. `synnax_writer_write` (writer, lv_test key, DBL array, count) — call repeatedly in a
   loop; watch the Console line plot.
4. `synnax_writer_close` (closes + frees the writer).
5. `synnax_client_close`.

Check the I32 return after each call; nonzero means failure (wire it to a stop condition
/ indicator). Error-message detail comes later once we parse `SynnaxError`.

## Next steps

- Parse `SynnaxError` (cluster of `i32` + `128`-byte + `512`-byte buffers) to surface
  failure messages in LabVIEW instead of a bare return code.
- Add a persisted-write path (index timestamps + float64 column) and use `commit`.
- Package the VIs into a `.lvlibp` (PPL) per RESEARCH.md Phase 5.
