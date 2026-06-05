// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "client/clib/synnax.h"

#include <cstring>

namespace {
constexpr char VERSION[] = "synnax-clib 0.1.0";
}

int32_t synnax_version(char *buf, const size_t buf_size) {
    const size_t len = sizeof(VERSION) - 1;
    if (buf != nullptr && buf_size > 0) {
        const size_t n = len < buf_size - 1 ? len : buf_size - 1;
        std::memcpy(buf, VERSION, n);
        buf[n] = '\0';
    }
    return static_cast<int32_t>(len);
}
