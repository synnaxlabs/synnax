// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <bit>
#include <cmath>
#include <cstddef>
#include <cstdint>
#include <limits>
#include <optional>
#include <string>
#include <type_traits>
#include <utility>
#include <variant>

#include "client/cpp/icd/types.gen.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/number.h"

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

std::pair<Encoding, x::errors::Error> encoding_of(const synnax::icd::BinaryField &f) {
    if (std::holds_alternative<synnax::icd::IntegerBinaryField>(f))
        return {Encoding::SIGNED, x::errors::NIL};
    if (!std::holds_alternative<synnax::icd::FloatBinaryField>(f))
        return {Encoding::UNSIGNED, x::errors::NIL};
    const auto n = synnax::icd::base(f).bit_length;
    if (n == 32) return {Encoding::FLOAT32, x::errors::NIL};
    if (n == 64) return {Encoding::FLOAT64, x::errors::NIL};
    return {
        Encoding::FLOAT64,
        x::errors::Error(
            CONFIG_ERROR,
            "a float field must be 32 or 64 bits, not " + std::to_string(n)
        ),
    };
}

std::pair<std::optional<std::uint64_t>, x::errors::Error>
default_raw(const synnax::icd::BinaryField &f) {
    using Result = std::pair<std::optional<std::uint64_t>, x::errors::Error>;
    const auto [encoding, err] = encoding_of(f);
    if (err) return {std::nullopt, err};
    const auto n = synnax::icd::base(f).bit_length;
    const auto out_of_range = [n](const std::string &v) -> Result {
        return {
            std::nullopt,
            x::errors::Error(
                CONFIG_ERROR,
                v + " is out of range for " + std::to_string(n) + " bits"
            ),
        };
    };
    return std::visit(
        [&]<typename F>(const F &v) -> Result {
            if (!v.default_.has_value()) return {std::nullopt, x::errors::NIL};
            if constexpr (std::is_same_v<F, synnax::icd::FloatBinaryField>) {
                const double d = v.default_->value;
                if (encoding == Encoding::FLOAT64)
                    return {std::bit_cast<std::uint64_t>(d), x::errors::NIL};
                if (std::abs(d) > std::numeric_limits<float>::max())
                    return {
                        std::nullopt,
                        x::errors::Error(
                            CONFIG_ERROR,
                            x::json::Number(d).to_string() + " overflows float32"
                        ),
                    };
                return {
                    std::bit_cast<std::uint32_t>(static_cast<float>(d)),
                    x::errors::NIL,
                };
            } else if constexpr (std::is_same_v<F, synnax::icd::IntegerBinaryField>) {
                const auto i = v.default_->value;
                const auto shift = 64 - n;
                if ((i << shift) >> shift != i) return out_of_range(std::to_string(i));
                return {static_cast<std::uint64_t>(i), x::errors::NIL};
            } else {
                const auto u = v.default_->value;
                if (n < 64 && u >> n != 0) return out_of_range(std::to_string(u));
                return {u, x::errors::NIL};
            }
        },
        f
    );
}
}
