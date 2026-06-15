// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Shared C types for the Synnax C-ABI library (synnax_clib): the error struct and
// the opaque handles. Included by both synnax.h and framer/framer.h.

#pragma once

#include <stdint.h>

#ifdef __cplusplus
extern "C" {
#endif

/// @brief opaque handle to a connected client (see synnax_client_open).
typedef struct SynnaxClient SynnaxClient;

/// @brief opaque handle to an open writer (see synnax_writer_open).
typedef struct SynnaxWriter SynnaxWriter;

/// @brief result of a fallible call. code is 0 on success; nonzero on failure with
/// type (e.g. "sy.validation") and a human-readable message filled in. May be NULL
/// to ignore the details and branch on the return code alone.
typedef struct {
    int32_t code;
    char type[128];
    char message[512];
} SynnaxError;

#ifdef __cplusplus
}
#endif
