// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <type_traits>

namespace xmath {

/// Floor division: rounds toward negative infinity instead of toward zero.
template<typename T>
constexpr T floor_div(const T a, const T b) {
    static_assert(std::is_integral_v<T> && std::is_signed_v<T>,
                  "floor_div requires a signed integer type");
    return a / b - (a % b != 0 && (a ^ b) < 0);
}

}
