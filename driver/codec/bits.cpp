// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>

#include "client/cpp/library/types.gen.h"

#include "driver/codec/bits.h"
#include "driver/codec/errors.h"

namespace driver::codec {
std::pair<BitRange, x::errors::Error> BitRange::compile(
    const std::uint16_t start_bit,
    const std::uint8_t bit_length,
    const std::string &byte_order
) {
    if (bit_length < 1 || bit_length > 64)
        return {
            {},
            x::errors::Error(
                LAYOUT_ERROR,
                "bit length must be from 1 to 64, got " + std::to_string(bit_length)
            ),
        };
    const bool big = byte_order == synnax::library::BYTE_ORDER_BIG_ENDIAN;
    if (!big && byte_order != synnax::library::BYTE_ORDER_LITTLE_ENDIAN)
        return {
            {},
            x::errors::Error(LAYOUT_ERROR, "unknown byte order: " + byte_order),
        };
    // Both orders map to a linear run of bit positions. Little-endian positions count
    // from the least significant bit of byte 0, big-endian ones from the most
    // significant bit of byte 0.
    const std::size_t first = big ? (start_bit / 8) * 8 + 7 - start_bit % 8 : start_bit;
    const std::size_t last = first + bit_length - 1;
    BitRange r;
    r.length_ = bit_length;
    r.end_ = last / 8 + 1;
    for (std::size_t b = first / 8; b <= last / 8; b++) {
        const auto lo = std::max(first, b * 8);
        const auto hi = std::min(last, b * 8 + 7);
        const auto width = hi - lo + 1;
        auto &s = r.segments[r.count++];
        s.byte = b;
        s.mask = static_cast<std::uint8_t>((1u << width) - 1);
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
