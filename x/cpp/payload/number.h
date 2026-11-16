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

#include "x/cpp/json/json.h"

namespace x::payload {

/// @brief Float is a 64-bit float. It is a JSON number when finite, and the text
/// "NaN", "Infinity", or "-Infinity" when not.
struct Float {
    double value = 0;

    Float() = default;

    // NOLINTNEXTLINE(misc-explicit-constructor): generated fields default to 0.
    Float(const double value): value(value) {}

    /// @brief parses a float from a JSON number or a non-finite text.
    static Float parse(const x::json::Parser &parser);

    /// @brief serializes the float as a JSON number, or as its text when it is not
    /// finite.
    [[nodiscard]] x::json::json to_json() const;
};

/// @brief Integer is a signed 64-bit integer. It is a JSON string of decimal digits,
/// so it keeps every digit.
struct Integer {
    std::int64_t value = 0;

    Integer() = default;

    // NOLINTNEXTLINE(misc-explicit-constructor): generated fields default to 0.
    Integer(const std::int64_t value): value(value) {}

    /// @brief parses an integer from a JSON string of decimal digits or a JSON number.
    static Integer parse(const x::json::Parser &parser);

    /// @brief serializes the integer as a JSON string of decimal digits.
    [[nodiscard]] x::json::json to_json() const;
};

/// @brief Unsigned is an unsigned 64-bit integer. It is a JSON string of decimal
/// digits, so it keeps every digit.
struct Unsigned {
    std::uint64_t value = 0;

    Unsigned() = default;

    // NOLINTNEXTLINE(misc-explicit-constructor): generated fields default to 0.
    Unsigned(const std::uint64_t value): value(value) {}

    /// @brief parses an unsigned integer from a JSON string of decimal digits or a
    /// JSON number.
    static Unsigned parse(const x::json::Parser &parser);

    /// @brief serializes the unsigned integer as a JSON string of decimal digits.
    [[nodiscard]] x::json::json to_json() const;
};

/// @brief lets nlohmann::json assign a Float as its JSON form.
inline void to_json(x::json::json &j, const Float &f) {
    j = f.to_json();
}

/// @brief lets nlohmann::json assign an Integer as its JSON form.
inline void to_json(x::json::json &j, const Integer &i) {
    j = i.to_json();
}

/// @brief lets nlohmann::json assign an Unsigned as its JSON form.
inline void to_json(x::json::json &j, const Unsigned &u) {
    j = u.to_json();
}

}
