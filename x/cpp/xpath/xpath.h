// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <filesystem>
#include <string>
#include <vector>

namespace xpath {
/// @brief joins path segments with a separator, skipping empty segments.
inline std::string
join(const std::string &sep, const std::vector<std::string> &segments) {
    std::string result;
    for (const auto &seg: segments) {
        if (seg.empty()) continue;
        if (!result.empty()) result += sep;
        result += seg;
    }
    return result;
}
/// @param returns the current working directory as a string.
inline std::string cwd() {
    return std::filesystem::current_path().string();
}

/// @brief resolves the relative path into an absolute path using the current
/// working directory.
/// @param path the relative path to resolve.
inline std::string resolve_relative(const std::string &path) {
    const std::filesystem::path base = std::filesystem::current_path();
    const std::filesystem::path relative(path);
    return (base / relative).lexically_normal().string();
}
}
