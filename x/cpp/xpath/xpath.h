// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <filesystem>

namespace xpath {
inline std::string cwd() {
    return std::filesystem::current_path().string();
}

inline std::string resolve_relative(const std::string &path) {
    const std::filesystem::path base = std::filesystem::current_path();
    const std::filesystem::path relative(path);
    return (base / relative).lexically_normal().string();
}
}