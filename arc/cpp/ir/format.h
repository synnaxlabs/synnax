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
#include <map>
#include <sstream>
#include <string>
#include <vector>

namespace arc::ir {
/// @brief Returns the tree prefix for a tree item.
/// @param last If true, returns "└── ", otherwise "├── ".
inline std::string tree_prefix(bool last) {
    return last ? "└── " : "├── ";
}

/// @brief Returns the indent for children of a tree item.
/// @param last If true, returns "    ", otherwise "│   ".
inline std::string tree_indent(bool last) {
    return last ? "    " : "│   ";
}

/// @brief Formats parameters as "name (type), name (type), ..."
template<typename Params>
std::string format_params(const Params &params) {
    if (params.empty()) return "(none)";
    std::ostringstream ss;
    bool first = true;
    for (const auto &p: params) {
        if (!first) ss << ", ";
        first = false;
        ss << p.name << " (" << p.type.to_string() << ")";
        if (!p.value.is_null()) ss << " = " << p.value.dump();
    }
    return ss.str();
}

/// @brief Formats channels as "read [id: name, ...], write [id: name, ...]"
template<typename Channels>
std::string format_channels(const Channels &ch) {
    if (ch.read.empty() && ch.write.empty()) return "(none)";

    std::ostringstream ss;
    bool has_read = !ch.read.empty();
    bool has_write = !ch.write.empty();

    if (has_read) {
        ss << "read [";
        // Sort keys for consistent output
        std::vector<std::pair<uint32_t, std::string>> read_pairs(
            ch.read.begin(),
            ch.read.end()
        );
        std::sort(read_pairs.begin(), read_pairs.end());
        bool first = true;
        for (const auto &[id, name]: read_pairs) {
            if (!first) ss << ", ";
            first = false;
            ss << id << ": " << name;
        }
        ss << "]";
    }

    if (has_write) {
        if (has_read) ss << ", ";
        ss << "write [";
        // Sort keys for consistent output
        std::vector<std::pair<uint32_t, std::string>> write_pairs(
            ch.write.begin(),
            ch.write.end()
        );
        std::sort(write_pairs.begin(), write_pairs.end());
        bool first = true;
        for (const auto &[id, name]: write_pairs) {
            if (!first) ss << ", ";
            first = false;
            ss << id << ": " << name;
        }
        ss << "]";
    }

    return ss.str();
}
}
