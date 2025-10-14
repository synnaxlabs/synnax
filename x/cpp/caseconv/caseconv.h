// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <string>

namespace caseconv {
/// @brief converts a snake_case string to SCREAMING_SNAKE_CASE
/// @param input The snake_case string to convert
/// @return The input string converted to SCREAMING_SNAKE_CASE
/// @example caseconv::snake_to_scream("hello_world") returns "HELLO_WORLD"
inline std::string snake_to_scream(const std::string &input) {
    std::string result = input;
    std::transform(result.begin(), result.end(), result.begin(), [](unsigned char c) {
        return static_cast<char>(std::toupper(c));
    });
    return result;
}

/// @brief converts a snake_case string to kebab-case
/// @param input The snake_case string to convert
/// @return The input string converted to kebab-case
/// @example caseconv::snake_to_kebab("hello_world") returns "hello-world"
inline std::string snake_to_kebab(const std::string &input) {
    std::string result = input;
    std::replace(result.begin(), result.end(), '_', '-');
    return result;
}
}
