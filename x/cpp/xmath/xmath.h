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

namespace xmath {

/// @brief computes floor division, rounding toward negative infinity instead of toward
/// zero.
/// @param a the dividend.
/// @param b the divisor.
/// @returns the quotient rounded toward negative infinity.
[[nodiscard]] inline constexpr int64_t floor_div(const int64_t a, const int64_t b) {
    return a / b - (a % b != 0 && (a ^ b) < 0);
}

}
