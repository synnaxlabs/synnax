// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// This is the pure-C public header for the Synnax C-ABI library (synnax_clib).
// It is consumed by flat-C FFI callers such as LabVIEW's Call Library Function
// Node. It must remain valid C: no C++ types, namespaces, or STL.

#pragma once

#include <stddef.h>
#include <stdint.h>

#include "client/clib/export.h"

#ifdef __cplusplus
extern "C" {
#endif

/// @brief writes the null-terminated library version into buf (at most buf_size
/// bytes) and returns the full length excluding the null; a return >= buf_size
/// means the value was truncated. buf may be NULL when buf_size is 0.
SYNNAX_EXPORT int32_t synnax_version(char *buf, size_t buf_size);

#ifdef __cplusplus
}
#endif
