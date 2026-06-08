# LabVIEW Spike: Connect + Write

End-to-end walkthrough for driving `synnax_clib.dll` from LabVIEW: open a client, write
float64 samples, close. The public API is a flat C surface (the 7 `synnax_*` exports),
so every Call Library Function Node (CLFN) parameter is a plain scalar, string, array,
or opaque handle — no pointer-bearing structs.

## Scope / constraints (spike)

- **64-bit only.** DLL bitness must match LabVIEW (use 64-bit LabVIEW).
- **Insecure connection** for the demo: wire `secure = 0` and `ca_cert_file = ""`. (TLS
  works via `secure = 1` + a CA file path, but the spike runs against an insecure Core.)
- **Any numeric dtype, multi-channel** per write: `synnax_writer_write` takes a
  channel-major data block plus a `data_type` name (an `x::telem::DataType`).
- **Demo writes stream-only to a virtual channel** (`mode = 3`). No index/timestamp
  column is needed and no commit is required. Observe the samples live in a Console line
  plot subscribed to the channel.
- Persisted writes (`mode = 1` or `2`) additionally require writing the channel's index
  timestamps; `synnax_writer_write` covers this via its `index_channel` + `timestamps`
  parameters (or open with `auto_index` and skip them).
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
Copy-Item $dll C:\synnax\client\labview\synnax_clib.dll
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

- **Library name/path:** `C:\synnax\client\labview\synnax_clib.dll`
- **Function name:** as below
- **Calling convention:** **C**
- **Thread:** **Run in any thread** (these calls block on the network)

`UPtr` = Numeric → _Unsigned Pointer-sized Integer_. **Opaque handles**
(`SynnaxClient*`, `SynnaxWriter*`) and the `err` pointer are all `UPtr`.

**Wizard header.** The Import Shared Library wizard's C parser rejects `synnax.h`
because it can't resolve `size_t` (skips `<stddef.h>`) or the opaque `SynnaxClient`
typedef. Point the wizard at `synnax_labview.h` instead — it's ABI-identical but spells
handles as `void*` (→ UPtr) and `size_t` as `uint64_t`. Copy it next to the DLL:

```powershell
Copy-Item C:\synnax\client\clib\synnax_labview.h C:\synnax\client\labview\synnax_labview.h
```

You can also skip the wizard entirely and configure the 7 CLFNs by hand from the tables
below — no header parsing involved.

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

| #   | Param                | LabVIEW type             | Pass                       |
| --- | -------------------- | ------------------------ | -------------------------- |
| ret | return               | Numeric, Signed 32-bit   | Value                      |
| 1   | host                 | String                   | C String Pointer           |
| 2   | port                 | Numeric, Unsigned 16-bit | Value                      |
| 3   | username             | String                   | C String Pointer           |
| 4   | password             | String                   | C String Pointer           |
| 5   | secure               | Numeric, Signed 32-bit   | Value (wire `0` insecure)  |
| 6   | ca_cert_file         | String                   | C String Pointer (wire "") |
| 7   | client_cert_file     | String                   | C String Pointer (wire "") |
| 8   | client_key_file      | String                   | C String Pointer (wire "") |
| 9   | max_retries          | Numeric, Unsigned 32-bit | Value (wire `0` = default) |
| 10  | clock_skew_threshold | Numeric, Signed 64-bit   | Value (wire `0` = 1s)      |
| 11  | out_client           | Numeric, UPtr            | **Pointer to Value**       |
| 12  | err                  | Numeric, UPtr            | Value (wire `0`)           |

Wire `host=""`, `port=0`, `username=""`, `password=""` to take dev defaults
(localhost:9090 synnax/seldon); leave `client_cert_file`/`client_key_file` `""` and
`max_retries`/`clock_skew_threshold` `0` for the insecure dev path. The client handle
comes out of param 11.

### `synnax_client_close` → void

| #   | Param  | LabVIEW type  | Pass  |
| --- | ------ | ------------- | ----- |
| 1   | client | Numeric, UPtr | Value |

### `synnax_writer_open` → I32

| #   | Param                       | LabVIEW type                | Pass                             |
| --- | --------------------------- | --------------------------- | -------------------------------- |
| ret | return                      | Numeric, Signed 32-bit      | Value                            |
| 1   | client                      | Numeric, UPtr               | Value                            |
| 2   | start                       | Numeric, Signed 64-bit      | Value (wire `0` for stream-only) |
| 3   | channels                    | Array, Unsigned 32-bit, 1-D | **Array Data Pointer**           |
| 4   | channel_count               | Numeric, UPtr               | Value (wire array size)          |
| 5   | authorities                 | Array, Unsigned 8-bit, 1-D  | **Array Data Pointer** (empty)   |
| 6   | authority_count             | Numeric, UPtr               | Value (wire `0` = absolute)      |
| 7   | subject_name                | String                      | C String Pointer (wire "")       |
| 8   | subject_group               | Numeric, Unsigned 32-bit    | Value (wire `0`)                 |
| 9   | mode                        | Numeric, Signed 32-bit      | Value (wire `3` = stream)        |
| 10  | err_on_unauthorized         | Numeric, Signed 32-bit      | Value (wire `0`)                 |
| 11  | enable_auto_commit          | Numeric, Signed 32-bit      | Value (wire `0`)                 |
| 12  | auto_index_persist_interval | Numeric, Signed 64-bit      | Value (wire `0` = 1s default)    |
| 13  | auto_index                  | Numeric, Signed 32-bit      | Value (wire `0`)                 |
| 14  | out_writer                  | Numeric, UPtr               | **Pointer to Value**             |
| 15  | err                         | Numeric, UPtr               | Value (wire `0`)                 |

Build a U32 array with the single `lv_test` key for param 3; wire its size to param 4.
For `mode`, see the **Writer mode Ring** below. For a simple uncontended writer, default
the control/index params: empty `authorities` array + `authority_count = 0` (absolute on
all channels), `subject_name = ""`, `subject_group = 0`, `err_on_unauthorized = 0`,
`auto_index_persist_interval = 0` (⇒ 1s), `auto_index = 0`.

### `synnax_writer_write` → I32 (multi-channel, any dtype)

One export backs a **polymorphic** `Synnax Write.vi`; each concrete instance (RT-safe,
no malleable VIs) fixes the `data` array element type and hardcodes `data_type`.

| #   | Param         | LabVIEW type                | Pass                                       |
| --- | ------------- | --------------------------- | ------------------------------------------ |
| ret | return        | Numeric, Signed 32-bit      | Value                                      |
| 1   | writer        | Numeric, UPtr               | Value                                      |
| 2   | index_channel | Numeric, Unsigned 32-bit    | Value (`0` = auto-index)                   |
| 3   | timestamps    | Array, Signed 64-bit, 1-D   | **Array Data Pointer** (empty if no index) |
| 4   | channels      | Array, Unsigned 32-bit, 1-D | **Array Data Pointer**                     |
| 5   | channel_count | Numeric, UPtr               | Value (# channels)                         |
| 6   | data          | Array, &lt;dtype&gt;, 1/2-D | **Array Data Pointer**                     |
| 7   | sample_count  | Numeric, UPtr               | Value (samples per channel)                |
| 8   | data_type     | String                      | C String Pointer (`"float64"`, …)          |
| 9   | err           | Numeric, UPtr               | Value (wire `0`)                           |

- **`data` is channel-major.** A 2-D array with rows = channels, cols = samples passes
  correctly via the 2-D Array Data Pointer: `sample_count` = cols, `channel_count` =
  rows.
- **`data_type`** is an `x::telem::DataType` name (`"float64"`, `"float32"`, `"int64"`,
  `"int32"`, `"uint8"`, `"timestamp"`, …) — hardcode it per polymorphic instance.
- **Index:** non-auto-indexed writer → wire `index_channel` + an I64 `timestamps` array
  (`sample_count` entries, ns since epoch). Auto-indexed writer → `index_channel = 0`,
  empty `timestamps`.
- **Waveform instances** compute `timestamps[i] = t0_ns + i*dT_ns` in the VI and pass
  them like any explicit-timestamp call — the DLL never sees a waveform.

Shapes vary by the arrays: 1 ch (`channels = [k]`, count 1) vs N ch; 1 samp
(`sample_count = 1`) vs S samp; 2-D `data` for N×S.

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
