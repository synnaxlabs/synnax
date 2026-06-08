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

/// @brief opens a writer over the given channel keys; param order mirrors TS/Python
/// open_writer. authorities is NULL for absolute on all, length 1 to broadcast, or
/// length channel_count for per-channel; auto_index_persist_interval ns falls back to
/// 1s when <= 0. On success writes the handle to *out_writer and returns 0.
SYNNAX_EXPORT int32_t synnax_writer_open(
    SynnaxClient *client,
    int64_t start,
    const uint32_t *channels,
    size_t channel_count,
    const uint8_t *authorities,
    size_t authority_count,
    const char *subject_name,
    uint32_t subject_group,
    int32_t mode,
    int32_t err_on_unauthorized,
    int32_t enable_auto_commit,
    int64_t auto_index_persist_interval,
    int32_t auto_index,
    SynnaxWriter **out_writer,
    SynnaxError *err
);

/// @brief writes sample_count samples to each of channel_count channels (channel-major,
/// data_type is a DataType name); also writes timestamps to index_channel when nonzero.
SYNNAX_EXPORT int32_t synnax_writer_write(
    SynnaxWriter *writer,
    uint32_t index_channel,
    const int64_t *timestamps,
    const uint32_t *channels,
    size_t channel_count,
    const void *data,
    size_t sample_count,
    const char *data_type,
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
