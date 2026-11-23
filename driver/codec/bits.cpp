// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <cstdint>
#include <string>
#include <utility>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/codec/bits.h"
#include "driver/codec/errors.h"

namespace driver::codec {
std::pair<BitRange, x::errors::Error>
BitRange::create(const synnax::icd::BinaryBaseField &field) {
    const auto length = field.bit_length;
    if (length < 1 || length > 64)
        return {
            BitRange{},
            x::errors::Error(
                CONFIG_ERROR,
                "bit length must be from 1 to 64, got " + std::to_string(length)
            ),
        };
    const bool big = field.byte_order == synnax::icd::BYTE_ORDER_BIG_ENDIAN;
    if (!big && field.byte_order != synnax::icd::BYTE_ORDER_LITTLE_ENDIAN)
        return {
            BitRange{},
            x::errors::Error(CONFIG_ERROR, "unknown byte order " + field.byte_order),
        };
    // Number the bits of either order as one run: a little-endian position counts up
    // from the least significant bit of byte 0, and a big-endian position counts up
    // from the most significant bit of byte 0.
    const std::size_t start = field.start_bit;
    const std::size_t first = big ? (start / 8) * 8 + 7 - start % 8 : start;
    const std::size_t last = first + length - 1;
    BitRange r;
    r.length = length;
    r.end_byte = last / 8 + 1;
    for (std::size_t b = first / 8; b <= last / 8; b++) {
        const auto lo = std::max(first, b * 8);
        const auto hi = std::min(last, b * 8 + 7);
        auto &s = r.segments[r.count++];
        s.byte = b;
        s.mask = static_cast<std::uint8_t>((1U << (hi - lo + 1)) - 1);
        if (big) {
            s.shift = static_cast<std::uint8_t>(7 - (hi - b * 8));
            s.dest = static_cast<std::uint8_t>(last - hi);
        } else {
            s.shift = static_cast<std::uint8_t>(lo - b * 8);
            s.dest = static_cast<std::uint8_t>(lo - first);
        }
    }
    return {r, x::errors::NIL};
}
}
