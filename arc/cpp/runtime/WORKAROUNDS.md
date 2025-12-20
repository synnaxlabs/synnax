# Arc Runtime Workarounds

This document tracks hacky/workaround fixes in the Arc C++ runtime that should be
revisited for proper solutions.

---

## 1. Entry Node Bypasses Watermark System

**Location:** `arc/cpp/runtime/stage/stage.h:34-47`

**What:** Changed `StageEntry::next()` to use `is_input_source_truthy()` instead of the
normal `refresh_inputs()` + `state.input(0)` flow.

**Why it's hacky:** Entry nodes now completely bypass the watermark-based input system
that all other nodes use. This creates a special case where entry nodes behave
differently from every other node type.

**Root cause not fixed:** The watermark system tracks timestamps to avoid reprocessing
data, but when timestamps don't change between stage re-entries (because the source
reuses input timestamps), the system thinks there's no new data. The proper fix would
be to either reset watermarks on stage transition or have a different propagation model
for one-shot edges.

---

## 2. New `is_input_source_truthy()` Method

**Location:** `arc/cpp/runtime/state/state.h:149-159`

**What:** Added a method that reads directly from `input_sources[i]->data` without
going through the accumulated/aligned data buffers.

**Why it's hacky:** This method exists solely to work around the watermark issue above.
It duplicates logic and creates two different ways to read input data.

---

## 3. Smart Pointer `.get()` Null Checks

**Location:** `arc/cpp/runtime/state/state.cpp:249-252`

**What:** Changed from `src->time != nullptr` to
`const auto *time_ptr = src->time.get(); if (time_ptr != nullptr ...)`.

**Why it's hacky:** The `local_shared` smart pointer's `operator!=` comparison to
nullptr apparently doesn't work as expected. Using `.get()` explicitly is a workaround
for what should be standard smart pointer behavior.

**Root cause not fixed:** The `xmemory::local_shared` implementation may have a bug or
non-standard behavior in its null comparison operators.

---

## 4. `size() > 0` Instead of `!empty()`

**Location:** `arc/cpp/runtime/state/state.cpp:252`, `arc/cpp/runtime/state/state.h:133`

**What:** Replaced `!series.empty()` with `series.size() > 0`.

**Why it's hacky:** These should be semantically identical. The change suggests
`empty()` may not be reliable, possibly due to how it interacts with the smart pointer
dereferencing.

**Root cause not fixed:** Either the `telem::Series::empty()` method or the smart
pointer's `operator->` has an issue that causes `empty()` to return incorrect results
in some states.

---

## Summary of Underlying Issues Not Addressed

1. **Watermark system incompatible with control flow:** The timestamp-based watermark
   tracking is designed for data processing, not control flow. Entry nodes need a
   different execution model.

2. **`local_shared` smart pointer quirks:** Multiple defensive workarounds suggest the
   custom smart pointer doesn't behave like `std::shared_ptr`. The root implementation
   may need review.

3. **Series state corruption on stage re-entry:** The crash when re-entering a stage
   suggests something about the output buffer state becomes invalid. The defensive null
   checks mask the symptom but don't explain why the state becomes invalid.
