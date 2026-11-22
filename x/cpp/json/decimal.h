// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <optional>
#include <string_view>

#include "x/cpp/errors/errors.h"

namespace x::json {
/// @brief Decimal is the magnitude of a decimal number times a power of ten, truncated
/// toward zero.
struct Decimal {
    std::uint64_t magnitude = 0;
    bool negative = false;
    /// @brief fraction is true when the truncation dropped a digit other than zero.
    bool fraction = false;
    /// @brief overflow is true when the magnitude does not fit a uint64.
    bool overflow = false;
};

/// @brief returns the decimal number text times 10^scale, or nothing when text is not a
/// decimal number.
std::optional<Decimal> parse_decimal(std::string_view text, int scale);

/// @brief returns the VALIDATION error for text that is not a number.
errors::Error invalid_number(std::string_view text);
}
