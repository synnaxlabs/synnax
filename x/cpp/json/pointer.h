// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

namespace x::json {
/// @brief Pointer is a parsed JSON Pointer (RFC 6901): the unescaped reference tokens
/// that lead from the root of a document to a value. No tokens point at the whole
/// document.
struct Pointer {
    /// @brief tokens are the unescaped reference tokens, from the root.
    std::vector<std::string> tokens;

    /// @brief parses the text form of a JSON Pointer.
    /// @returns VALIDATION when text is not empty and does not start with "/", or when
    /// a "~" is not followed by "0" or "1".
    static std::pair<Pointer, errors::Error> parse(std::string_view text);

    /// @brief returns the text form of the pointer.
    [[nodiscard]] std::string to_string() const;

    bool operator==(const Pointer &) const = default;
};
}
