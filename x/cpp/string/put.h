// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

namespace x::strings {

/// @brief writes v as a zero-padded decimal of the given width into the buffer at p.
/// @param p pointer to at least width writable chars.
/// @param v non-negative integer value to write.
/// @param width number of digits to write (zero-padded on the left).
inline void writeNumber(char *p, int v, int width) noexcept {
    for (int i = width - 1; i >= 0; --i) {
        p[i] = char('0' + (v % 10));
        v /= 10;
    }
}

}
