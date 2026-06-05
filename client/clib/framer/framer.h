// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Writer functions for the Synnax C-ABI library (synnax_clib). Mirrors the framer
// package of the TypeScript, Python, and C++ clients. Part of the public surface,
// pulled in via the synnax.h umbrella.

#pragma once

#include <stddef.h>
#include <stdint.h>

#include "client/clib/export.h"
#include "client/clib/types.h"

#ifdef __cplusplus
extern "C" {
#endif

/// @brief writer persistence/streaming mode. Values match the WriterMode enum used
/// by the other clients; DEFAULT (0) resolves to persist+stream.
typedef enum {
    SYNNAX_WRITER_MODE_DEFAULT = 0, // (persist + stream)
    SYNNAX_WRITER_MODE_PERSIST_STREAM = 1,
    SYNNAX_WRITER_MODE_PERSIST = 2,
    SYNNAX_WRITER_MODE_STREAM = 3,
} SynnaxWriterMode;

/// @brief opens a writer over the given channels. start is the domain start in ns
/// since epoch; mode is a SynnaxWriterMode value (0 = default = persist+stream);
/// enable_auto_commit nonzero makes writes immediately durable. On success writes
/// the handle to *out_writer and returns 0.
SYNNAX_EXPORT int32_t synnax_writer_open(
    SynnaxClient *client,
    int64_t start,
    const uint32_t *channels,
    size_t channel_count,
    int32_t mode,
    int32_t enable_auto_commit,
    SynnaxWriter **out_writer,
    SynnaxError *err
);

/// @brief writes sample_count float64 samples to a single channel. (float64-only
/// for now; other dtypes and multi-channel frames come later.)
SYNNAX_EXPORT int32_t synnax_writer_write(
    SynnaxWriter *writer,
    uint32_t channel,
    const double *data,
    size_t sample_count,
    SynnaxError *err
);

/// @brief commits pending writes, writing the committed end timestamp (ns) to
/// out_end_ts when non-NULL. Unnecessary under enable_auto_commit.
SYNNAX_EXPORT int32_t
synnax_writer_commit(SynnaxWriter *writer, int64_t *out_end_ts, SynnaxError *err);

/// @brief flushes, closes, and frees the writer in one call, returning any error
/// accumulated during writing. Safe on NULL. The handle is dead afterward.
SYNNAX_EXPORT int32_t synnax_writer_close(SynnaxWriter *writer, SynnaxError *err);

#ifdef __cplusplus
}
#endif
