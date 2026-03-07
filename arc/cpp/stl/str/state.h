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

namespace arc::stl::str {

/// Handle store for strings created during WASM execution.
/// Manages both transient handles (cleared each cycle) and config handles
/// (stable for the State lifetime).
class State {
    static constexpr uint32_t CONFIG_HANDLE_BASE = 1 << 24;

    std::unordered_map<uint32_t, std::string> handles;
    uint32_t counter = 1;
    std::unordered_map<uint32_t, std::string> config_handles;
    uint32_t config_counter = CONFIG_HANDLE_BASE;

public:
    /// Creates a transient string handle from a C++ string.
    uint32_t create(const std::string &s) {
        const uint32_t handle = this->counter++;
        this->handles[handle] = s;
        return handle;
    }

    /// Creates a stable config string handle that persists across clear() calls.
    /// Use for config param strings whose handles are baked into node args.
    uint32_t create_config(const std::string &s) {
        const uint32_t handle = this->config_counter++;
        this->config_handles[handle] = s;
        return handle;
    }

    /// Creates a string handle from raw memory pointer and length.
    uint32_t from_memory(const uint8_t *data, uint32_t len) {
        const std::string s(reinterpret_cast<const char *>(data), len);
        const uint32_t handle = this->counter++;
        this->handles[handle] = s;
        return handle;
    }

    /// Gets the string value for a handle. Checks transient first, then config.
    /// Returns empty string if not found.
    std::string get(uint32_t handle) const {
        const auto it = this->handles.find(handle);
        if (it != this->handles.end()) return it->second;
        const auto cit = this->config_handles.find(handle);
        if (cit != this->config_handles.end()) return cit->second;
        return "";
    }

    /// Checks if a string handle exists (transient or config).
    bool exists(uint32_t handle) const {
        return this->handles.contains(handle) || this->config_handles.contains(handle);
    }

    /// Clears transient handles. Config handles are preserved.
    void clear() {
        this->handles.clear();
        this->counter = 1;
    }

    /// Clears all handles including config handles.
    void reset() {
        this->clear();
        this->config_handles.clear();
        this->config_counter = CONFIG_HANDLE_BASE;
    }
};

}
