// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <string>
#include <string_view>

namespace x::json {
/// @brief LiteralScan is the result of reading a JSON string literal.
struct LiteralScan {
    /// @brief end is the offset past the closing quote, or the offset of the first byte
    /// that is not valid.
    std::size_t end = 0;
    /// @brief problem says why the literal is not valid, or is empty when it is.
    std::string_view problem;
};

/// @brief reads the JSON string literal at the start of text, which starts with its
/// opening quote. When dst is not null, it appends the decoded string to dst.
LiteralScan scan_literal(std::string_view text, std::string *dst);
}
