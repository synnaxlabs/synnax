// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Channel functions for the Synnax C-ABI library (synnax_clib). Mirrors the channel
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

/// @brief resolves name_count '\n'-delimited names to caller-allocated out_keys,
/// out_index_keys, and '\n'-joined out_dtypes; NULL skips the last two. Misses return
/// NOT_FOUND but still fill every slot, aligned to the names: key 0 and an empty dtype.
SYNNAX_EXPORT int32_t synnax_channel_retrieve_keys(
    SynnaxClient *client,
    const char *names,
    size_t name_count,
    uint32_t *out_keys,
    uint32_t *out_index_keys,
    char *out_dtypes,
    size_t out_dtypes_size,
    SynnaxError *err
);

/// @brief creates count channels, writing their keys to out_keys. names and data_types
/// are '\n'-delimited (count each); is_index/index/is_virtual are arrays of count.
SYNNAX_EXPORT int32_t synnax_channel_create(
    SynnaxClient *client,
    const char *names,
    const char *data_types,
    const uint8_t *is_index,
    const uint32_t *index,
    const uint8_t *is_virtual,
    size_t count,
    uint32_t *out_keys,
    SynnaxError *err
);

#ifdef __cplusplus
}
#endif
