# Synnax LabVIEW Client: Research Report

**Status:** Research spike. No implementation code yet.
**Target:** Windows 64-bit (first).
**Scope:** Basic streaming (read live) and writing only. No channel creation, ranges,
tasks, or string/JSON series in the MVP.
**Date:** 2026-06-05.

## Summary

Synnax ships client libraries for Python, TypeScript, and C++, but not for LabVIEW. This
report answers the two questions blocking the start of a LabVIEW client and lays out a
phased, verifiable build roadmap.

**Question 1 (installation / licensing): Can the free trial build PPLs?**
Yes, during the trial window. The LabVIEW free trial is full LabVIEW Professional (minus
Nigel AI), good for 7 days and extendable to 45. Professional includes the Application
Builder, which is what builds Packed Project Libraries (`.lvlibp`). Production
building/distribution needs a paid Professional license.

**Question 2 (build strategy): How do we wrap the C++ client?**
Keep the existing C++ client and put a thin `extern "C"` C-ABI shim in front of it,
compiled with the client and all heavy dependencies into one self-contained x64 DLL.
LabVIEW cannot call a C++ DLL directly, but it can call a flat C DLL through its Call
Library Function Node. The shim hides all C++ behind opaque handles. LabVIEW then wraps
each C function in a VI, and those VIs are packaged into the PPL.

The single biggest unknown is whether protobuf, gRPC, and OpenSSL link cleanly into one
Windows DLL. A Phase 0 hello-world DLL and a Phase 1 connect milestone are designed to
prove the toolchain before any real client work.

---

## Background

LabVIEW is the dominant environment for a large segment of test and measurement and
hardware teams, so a first-class LabVIEW client meaningfully widens Synnax's reach. The
near-term goal is intentionally small: connect to a Synnax Core, write telemetry, and
stream live telemetry back. Everything else (channel management, ranges, control tasks)
is deferred.

LabVIEW programs are graphical VIs. To call native code, LabVIEW provides the **Call
Library Function Node (CLFN)**, which loads a DLL and calls an exported C function. The
distribution format for compiled LabVIEW code is the **Packed Project Library (PPL,
`.lvlibp`)**, which is conceptually a DLL for LabVIEW VIs.

---

## Question 1: Installation and licensing

### Can the free trial build PPLs?

Yes, for the duration of the trial.

- The LabVIEW **free trial provides full LabVIEW Professional functionality** (the only
  exclusion noted is Nigel AI). It runs for **7 days and can be extended up to 45 days**
  after first launch.
- **PPLs are built by the LabVIEW Application Builder**, via a project Build
  Specification of type "Packed Library." The **Professional edition includes the
  Application Builder**, so the trial includes it. You can therefore build PPLs during
  the trial.
- A PPL (`.lvlibp`) is a compiled, non-editable LabVIEW library.

### Caveats worth internalizing

1. **Production needs a paid license.** Permanent building and distribution of PPLs
   requires a paid LabVIEW Professional license (or the Full Development System plus the
   Application Builder add-on). The trial covers the spike, not shipping.
2. **A PPL is not the DLL.** This is the most common point of confusion. The PPL
   packages the **LabVIEW VIs** (the LabVIEW-side wrappers we write). Our native C ABI
   DLL is a **separate artifact** that ships alongside the PPL. The VIs in the PPL call
   into that DLL through the CLFN. So:
   - PPL (`.lvlibp`) = the distributable LabVIEW glue.
   - DLL (`synnax_clib.dll`) = the native engine the glue calls.
3. **Confirm exact terms with NI.** Licensing details change across LabVIEW versions;
   verify with NI before relying on the trial for anything past prototyping.

### Sources

- [LabVIEW Free Trial (NI)](https://www.ni.com/en/shop/labview/free-trial)
- [Select Your NI LabVIEW Edition (NI)](https://www.ni.com/en/shop/labview/select-edition)
- [LabVIEW Application Builder Module (NI)](https://www.ni.com/en-us/shop/product/labview-application-builder-module.html)
- [Using Packed Project Libraries in LabVIEW Projects (NI)](https://www.ni.com/docs/en-US/bundle/labview/page/using-packed-project-libraries-in-labview-projects.html)

---

## Question 2: Build strategy

### Why a C shim is required (not optional)

LabVIEW's CLFN can only call **flat C DLLs**. It cannot call a C++ DLL directly for two
reasons:

1. **Name mangling.** C++ compilers decorate exported symbol names; LabVIEW expects
   undecorated C names. The fix is `extern "C"` on every exported function.
2. **No STL across the boundary.** C++ standard-library types (`std::string`,
   `std::vector`, `std::pair`, `std::variant`, smart pointers) have no stable binary
   layout that a non-C++ caller can construct or read. They cannot cross a C ABI.

The Synnax C++ client's **entire public API is C++-only**. A full sweep of `client/cpp`,
`freighter/cpp`, and `x/cpp` found **no `extern "C"` and no C ABI anywhere**. Every entry
point uses classes, templates, `std::*` containers, and `std::pair<T, Error>` returns.
For example:

- `synnax::Synnax(const synnax::Config&)` where `Config` holds `std::string` fields.
- `Client::open_writer(const WriterConfig&) -> std::pair<Writer, x::errors::Error>`.
- `Streamer::read() -> std::pair<x::telem::Frame, x::errors::Error>` where `Frame` is
  move-only and holds `unique_ptr`/`shared_ptr` internals.

So we do **not** rewrite the client in C, and we do **not** modify the C++ client. We add
a new, thin C wrapper that exposes flat C functions and hides every C++ detail behind
opaque handles.

### Recommended architecture

```
LabVIEW VIs  ---->  PPL (.lvlibp)          [the LabVIEW glue you distribute]
     |
     |  Call Library Function Node (C calling convention, x64)
     v
synnax_clib.dll                            [one self-contained native DLL]
     |  extern "C" shim   (client/clib/synnax.h + synnax.cpp)
     v
Synnax C++ client  (//client/cpp:synnax, linked statically)
     v
gRPC + protobuf + OpenSSL + glog          [all linked statically INSIDE the DLL]
     v
Synnax Core  (gRPC)
```

The DLL must be **x64** to match 64-bit LabVIEW. A CLFN can only load a DLL whose bitness
matches the LabVIEW IDE; a 32-bit DLL in 64-bit LabVIEW fails with "The library selected
is not valid for the current platform."

The repo's `.bazelrc` already enforces `/MT` (static C runtime) on Windows and
`--dynamic_mode=off`. Both are exactly what a single fat DLL with no external runtime
dependencies wants.

### The C-ABI shim

Proposed new directory, a sibling of `client/cpp`, `client/py`, `client/ts`:

```
client/clib/
  BUILD.bazel      cc_library (shim) + cc_shared_library (DLL)
  synnax.h         PURE C public header that LabVIEW consumes (no C++)
  synnax.cpp       implementation; includes the real C++ client
  export.h         SYNNAX_EXPORT macro (__declspec(dllexport) / visibility)
  synnax.def       Windows export allowlist (the synnax_* names only)
  labview/         VIs + PPL (added in Phase 5)
```

The public header is **pure C** so it documents the exact ABI and can be reused by other
FFI consumers (Python ctypes, MATLAB, C#). The `.cpp` includes the real C++ client.

**Universal conventions:**

- **Opaque handles.** Forward-declared structs hide all C++ internals:
  `SynnaxClient*`, `SynnaxWriter*`, `SynnaxStreamer*`, `SynnaxFrame*`.
- **Errors.** Every fallible function returns `int32_t` (0 = ok) and fills a
  caller-allocated error struct with fixed buffers (no cross-boundary string to free):
  ```c
  typedef struct {
      int32_t code;          /* 0 == ok; nonzero == failure */
      char    type[128];     /* e.g. "sy.query.not_found" */
      char    message[512];  /* human-readable */
  } SynnaxError;
  ```
  This maps directly onto the C++ `std::pair<T, x::errors::Error>` pattern: split into an
  `int32_t` return, an out-param handle, and the error struct.
- **Exception firewall (non-negotiable).** Every `extern "C"` body is wrapped in
  `try { ... } catch (...) { fill error; return code; }`. The C++ client throws in
  several paths (for example `Frame::at` on a missing channel, and Series bounds checks).
  An exception unwinding across the C ABI is undefined behavior and would crash LabVIEW.
- **Memory ownership.** Every library-allocated handle has a matching `*_free` in the
  DLL. LabVIEW never frees library memory, because the DLL carries its own static C
  runtime.

**Minimal API surface (connect + write + stream).** Illustrative design sketch, not
final:

```c
/* ---- client lifecycle ---- */
typedef struct {
    const char* host;             /* NULL -> "localhost" */
    uint16_t    port;             /* 0    -> 9090         */
    const char* username;         /* NULL -> "synnax"     */
    const char* password;         /* NULL -> "seldon"     */
    const char* ca_cert_file;     /* NULL/"" -> insecure  */
    const char* client_cert_file;
    const char* client_key_file;
} SynnaxClientConfig;

int32_t synnax_client_open(const SynnaxClientConfig* cfg,
                           SynnaxClient** out_client, SynnaxError* out_err);
void    synnax_client_close(SynnaxClient* client);

/* ---- writer ---- */
int32_t synnax_writer_open  (SynnaxClient* c, const SynnaxWriterConfig* cfg,
                             SynnaxWriter** out_w, SynnaxError* out_err);
int32_t synnax_writer_write (SynnaxWriter* w,
                             const uint32_t* channels,
                             const int32_t*  data_types,        /* per channel */
                             const void* const* column_data,    /* one buffer per channel */
                             size_t n_channels, size_t sample_count,
                             SynnaxError* out_err);
int32_t synnax_writer_commit(SynnaxWriter* w, int64_t* out_end_ts, SynnaxError* out_err);
int32_t synnax_writer_close (SynnaxWriter* w, SynnaxError* out_err);
void    synnax_writer_free  (SynnaxWriter* w);

/* ---- streamer ---- */
int32_t synnax_streamer_open (SynnaxClient* c, const SynnaxStreamerConfig* cfg,
                              SynnaxStreamer** out_s, SynnaxError* out_err);
int32_t synnax_streamer_read (SynnaxStreamer* s,                /* BLOCKING */
                              SynnaxFrame** out_frame, SynnaxError* out_err);
int32_t synnax_streamer_close(SynnaxStreamer* s, SynnaxError* out_err);
void    synnax_streamer_free (SynnaxStreamer* s);

/* ---- frame accessors (read the owned frame, then free it) ---- */
size_t  synnax_frame_channel_count(const SynnaxFrame* f);
int32_t synnax_frame_series_info  (const SynnaxFrame* f, size_t i,
                                   int32_t* out_data_type, size_t* out_sample_count);
int32_t synnax_frame_copy_column  (const SynnaxFrame* f, size_t i,
                                   int32_t out_data_type, /* desired type; shim casts */
                                   void* out_buf, size_t buf_capacity_samples,
                                   size_t* out_copied, SynnaxError* out_err);
void    synnax_frame_free(SynnaxFrame* f);
```

**Three design points that the codebase forces:**

1. **Connect must be validated explicitly.** The `synnax::Synnax` constructor does not
   open or check the connection; gRPC connects lazily and a background checker thread
   polls. If the shim returns success right after construction, "connect" always appears
   to succeed even against a dead host. So `synnax_client_open` must call
   `client->connectivity->check()` and surface the resulting state's error before
   returning. (See `client/cpp/synnax.h` and `client/cpp/connection/checker.*`.)

2. **Writing is column-oriented and typed.** LabVIEW's strength is fixed-type arrays
   (DBL, SGL, I64). `synnax_writer_write` takes parallel typed columns (channel keys, a
   data-type code per channel, an array of buffer pointers, and a shared sample count)
   and assembles the `x::telem::Frame` internally using `Series::cast(...)`. Timestamp
   index columns are passed as int64 nanosecond buffers tagged as the timestamp type.
   (See `x/cpp/telem/series.h`, `x/cpp/telem/frame.h`.)

3. **Reading uses a two-pass, library-owned frame.** `Frame` is move-only and its typed
   `Series` sits behind a `shared_ptr<byte[]>`, so it cannot be exposed directly.
   `synnax_streamer_read` moves the frame onto the heap and returns it as an owned
   `SynnaxFrame*`. LabVIEW then calls `synnax_frame_series_info` to learn each column's
   type and length, pre-allocates a matching array, calls `synnax_frame_copy_column` to
   copy out (the shim casts, commonly to float64 for a single generic VI), and finally
   `synnax_frame_free`. This "ask the size, then ask the data" idiom is the only sane way
   to return a variable-shape, variable-typed frame to a language that needs fixed
   arrays.

**Blocking read and threading.** `Streamer::read()` blocks until data or stream failure.
The shim stays simple and synchronous. On the LabVIEW side, run `synnax_streamer_read` in
a dedicated loop and call `synnax_streamer_close` from another loop to unblock it (the
C++ `Streamer` supports a concurrent close). We deliberately avoid C-to-LabVIEW
callbacks, which are fragile; a dedicated read loop is the robust pattern.

### Bazel build (in-repo `cc_shared_library`)

Two targets in `client/clib/BUILD.bazel`:

```python
load("@rules_cc//cc:cc_library.bzl", "cc_library")
load("@rules_cc//cc:cc_shared_library.bzl", "cc_shared_library")

cc_library(
    name = "clib",
    srcs = ["synnax.cpp"],
    hdrs = ["synnax.h", "export.h"],
    defines = ["SYNNAX_BUILDING_DLL=1"],   # flips SYNNAX_EXPORT to dllexport
    deps = ["//client/cpp:synnax"],        # pulls grpc++/protobuf/openssl/glog transitively
    alwayslink = True,                      # keep exported symbols from being dropped
    visibility = ["//visibility:public"],
)

cc_shared_library(
    name = "synnax_clib",                   # -> synnax_clib.dll
    deps = [":clib"],
    win_def_file = "synnax.def",            # export allowlist
    target_compatible_with = ["@platforms//os:windows"],
    visibility = ["//visibility:public"],
)
```

Build command:

```
bazel build //client/clib:synnax_clib --config=windows -c opt
```

`-c opt` matters: a debug build of the full gRPC/protobuf/abseil tree is enormous and
slow; opt is what ships.

**Symbol export mechanism.** Use a `.def` allowlist as the primary mechanism, plus a
`__declspec(dllexport)` macro as a backup. The repo already fights gRPC symbol collisions
with `-fvisibility=hidden` (there is an explicit note in `.bazelrc` about a clash with
LabJack's bundled gRPC), so by default nothing is exported. A hand-maintained `.def`
listing exactly the `synnax_*` functions exports our C API and nothing from
gRPC/protobuf, which both avoids symbol bloat and is independent of whether `dllexport`
survives `alwayslink`. Do **not** use `WINDOWS_EXPORT_ALL_SYMBOLS`: exporting all of
gRPC/protobuf is the collision risk we are avoiding. Verify the result with
`dumpbin /exports synnax_clib.dll`.

**New ground.** There is no `cc_shared_library` anywhere in the repo today. Bazel's
`rules_cc` supports it, but this is the main thing to de-risk, which is why Phase 0
exists.

### Sources

- [Calling C/C++ DLLs from LabVIEW (NI)](https://forums.ni.com/t5/Developer-Center-Resources/Calling-C-C-DLLs-from-LabVIEW/ta-p/3522488)
- [Configuring the Call Library Function Node (NI)](https://zone.ni.com/reference/en-XX/help/371361R-01/lvexcodeconcepts/configuring_the_clf_node/)
- [Calling a 32-Bit DLL from 64-Bit LabVIEW and Vice Versa (NI)](https://knowledge.ni.com/KnowledgeArticleDetails?id=kA00Z000000PA7SSAW)

---

## Phased roadmap

Each milestone is independently verifiable. Do them in order; the early phases exist to
retire risk cheaply.

| Phase | Goal | Pass/fail check |
| --- | --- | --- |
| **0** | Hello-world DLL LabVIEW can load. `client/clib/` exports one trivial function (`synnax_version`), no Synnax deps. | `dumpbin /exports` shows the symbol; a LabVIEW CLFN loads and calls it at x64. De-risks the whole Bazel + `.def` + CLFN toolchain. |
| **1** | Link the full client and connect. Add `//client/cpp:synnax`; implement `synnax_client_open/close` with `connectivity->check()` and the exception firewall. | Good credentials return 0; a bad host or bad credentials return nonzero with a populated `SynnaxError`. Proves the big risk: static-linking gRPC/protobuf/OpenSSL into a Windows DLL. |
| **2** | Write one frame. Implement the writer functions. Channel is pre-created via Console or Python. | Write a float64 column plus its timestamp index, then confirm the samples in the Console. |
| **3** | Stream one frame. Implement the streamer and frame accessors. | Write from Python or the Phase 2 path; confirm `read` returns a frame whose `copy_column` yields the written values. |
| **4** | Robustness. Concurrent close unblocks a blocked read; firewall coverage on every function; error-code enum; null-arg guards; error-buffer truncation safety. | Stress the read/close interplay and null inputs without crashing LabVIEW. |
| **5** | Package into a PPL. One VI per C function (CLFN configured: C calling convention; arrays as "Array Data Pointer"; strings as "C String Pointer"; handles as pointer-sized integer). Compose connect/write/stream demo VIs; build the `.lvlibp`. | A clean LabVIEW machine with only the PPL and the DLL on disk can connect, write, and stream end to end with no source VIs present. |

---

## Key risks and open questions

1. **Static-linking protobuf, gRPC, and OpenSSL into one Windows DLL.** Highest-impact
   unknown. Risks include protobuf's global descriptor pool rejecting duplicate
   registration, `/MT` static-CRT consistency across every object in the DLL, and
   OpenSSL's dependence on `user32.lib`. Proven in Phase 1. Validate this before building
   out the full API.
2. **`cc_shared_library` is new to the repo.** The strict Windows warnings-as-errors
   config (`/WX`) may surface fresh warnings in the shim translation unit; expect to add
   a couple of `per_file_copt` relaxations scoped to `client/clib/`.
3. **LabVIEW threading for the blocking read.** Confirm the dedicated-loop plus
   concurrent-close pattern unblocks cleanly, and that the CLFN set to "Run in any
   thread" does not deadlock the LabVIEW root loop. This is LabVIEW-side validation.
4. **Bitness.** This plan assumes 64-bit LabVIEW. If a 32-bit LabVIEW must be supported
   later, an x86 DLL is needed and the entire static-link stack must be re-validated on
   the more memory-constrained x86 linker.
5. **Licensing.** The trial is sufficient for the spike. Confirm Professional terms with
   NI before relying on it for production PPL building or distribution.
6. **No channel creation in scope.** Every test and demo depends on channels existing
   already (created via Console or Python). Acceptable for the MVP; revisit if LabVIEW
   users need to create channels.

---

## Appendix: key C++ files referenced

- `client/cpp/synnax.h` : `Config` struct, `Synnax` constructor, the `connectivity`
  checker. Basis for the connect shim and the explicit connection check.
- `client/cpp/framer/framer.h` : `WriterConfig`, `Writer`, `StreamerConfig`, `Streamer`,
  and the framer `Client`. Basis for the write and stream shims.
- `x/cpp/telem/series.h` : `Series::cast`, `write_casted`, `values<T>`, `at`. The typed
  buffer bridge used by both write and read.
- `x/cpp/telem/frame.h` : the move-only `Frame` plus `emplace`, `at`, `size`. Drives the
  `SynnaxFrame` opaque-handle accessor design.
- `client/cpp/BUILD.bazel` and `.bazelrc` : the static `cc_library` to wrap, and the
  global static, visibility, and CRT flags the DLL build inherits.
