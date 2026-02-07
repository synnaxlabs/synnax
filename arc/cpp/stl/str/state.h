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
#include <string>
#include <unordered_map>

namespace arc::runtime::stl::str {

/// Transient handle store for strings created during a single execution cycle.
/// Handles are uint32_t keys that WASM code uses to reference string objects.
class State {
    std::unordered_map<uint32_t, std::string> handles;
    uint32_t counter = 1;

public:
    /// Creates a string handle from a C++ string.
    uint32_t create(const std::string &s) {
        const uint32_t handle = this->counter++;
        this->handles[handle] = s;
        return handle;
    }

    /// Creates a string handle from raw memory pointer and length.
    uint32_t from_memory(const uint8_t *data, uint32_t len) {
        const std::string s(reinterpret_cast<const char *>(data), len);
        const uint32_t handle = this->counter++;
        this->handles[handle] = s;
        return handle;
    }

    /// Gets the string value for a handle. Returns empty string if not found.
    std::string get(uint32_t handle) const {
        const auto it = this->handles.find(handle);
        if (it == this->handles.end()) return "";
        return it->second;
    }

    /// Checks if a string handle exists.
    bool exists(uint32_t handle) const { return this->handles.contains(handle); }

    /// Clears all transient handles. Called at end of each execution cycle.
    void clear() {
        this->handles.clear();
        this->counter = 1;
    }
};

}
