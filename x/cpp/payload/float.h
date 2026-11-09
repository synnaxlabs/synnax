// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/json/json.h"

namespace x::payload {

/// @brief Float is a 64-bit float. It is a JSON number when finite, and the text
/// "NaN", "Infinity", or "-Infinity" when not.
struct Float {
    double value = 0;

    Float() = default;

    // NOLINTNEXTLINE(google-explicit-constructor)
    Float(const double value): value(value) {}

    /// @brief parses a float from a JSON number or a non-finite text.
    static Float parse(x::json::Parser parser);

    /// @brief serializes the float as a JSON number, or as its text when it is not
    /// finite.
    [[nodiscard]] x::json::json to_json() const;
};

/// @brief lets nlohmann::json assign a Float as its JSON form.
inline void to_json(x::json::json &j, const Float &f) {
    j = f.to_json();
}

}
