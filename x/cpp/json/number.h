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
#include <string>
#include <string_view>
#include <utility>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace x::json {
/// @brief Number is a JSON number held without losing precision. An integer that an
/// int64 or a uint64 holds stays exact, and any other number is the nearest double.
class Number {
    enum class Kind : std::uint8_t { Int, Uint, Float };

    Kind kind = Kind::Int;
    std::int64_t i = 0;
    std::uint64_t u = 0;
    double f = 0;
    /// @brief fraction is true when the text of a float had a fraction that f may round
    /// away.
    bool fraction = false;
    /// @brief text is the JSON text a float was parsed from, so errors show what f
    /// rounds.
    std::string text;

    [[nodiscard]] bool integral() const;
    [[nodiscard]] bool within(std::int64_t lo, std::uint64_t hi) const;

public:
    /// @brief creates the number 0.
    Number() = default;
    /// @brief creates a number equal to v, which errors print without the text of a
    /// JSON number.
    explicit Number(double v);
    /// @brief creates a number equal to v.
    explicit Number(std::int64_t v);
    /// @brief creates a number equal to v.
    explicit Number(std::uint64_t v);

    /// @brief parses the text of a JSON number.
    /// @returns VALIDATION when text is not a decimal number, or is past the range of a
    /// double.
    static std::pair<Number, errors::Error> parse(std::string_view text);

    /// @brief returns the number when it is an integer that an int64 holds.
    [[nodiscard]] std::optional<std::int64_t> to_int64() const;
    /// @brief returns the number when it is an integer that a uint64 holds.
    [[nodiscard]] std::optional<std::uint64_t> to_uint64() const;
    /// @brief returns the double nearest to the number.
    [[nodiscard]] double to_float64() const;

    /// @brief checks that the number converts into a sample of dt without wrapping,
    /// truncating, or overflowing to infinity.
    /// @returns VALIDATION when it does not. A data type that is not an integer or
    /// float32 takes any number.
    [[nodiscard]] errors::Error check_fits(const telem::DataType &dt) const;

    /// @brief returns the JSON text a float was parsed from, or else the number in the
    /// form the Core prints it. A number that is not finite prints as NaN, Infinity, or
    /// -Infinity.
    [[nodiscard]] std::string to_string() const;
};
}
