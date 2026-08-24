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
#include <memory>
#include <string>
#include <unordered_map>

namespace arc::stl::strings {

/// Handle store for strings created during WASM execution.
/// Manages both transient handles (cleared each cycle) and literal handles
/// (stable for the State lifetime).
class State {
    static constexpr uint32_t LITERAL_HANDLE_BASE = 1 << 24;

    std::unordered_map<uint32_t, std::string> handles;
    uint32_t counter = 1;
    std::unordered_map<uint32_t, std::string> literal_handles;
    uint32_t literal_counter = LITERAL_HANDLE_BASE;

public:
    /// Creates a transient string handle. Empty input returns handle 0.
    uint32_t create(const std::string &s) {
        if (s.empty()) return 0;
        const uint32_t handle = this->counter++;
        this->handles[handle] = s;
        return handle;
    }

    /// Creates a stable literal string handle that persists across clear() calls.
    /// Use for literal param strings baked into node args.
    /// Empty input returns handle 0.
    uint32_t create_literal(const std::string &s) {
        if (s.empty()) return 0;
        const uint32_t handle = this->literal_counter++;
        this->literal_handles[handle] = s;
        return handle;
    }

    /// Creates a string handle from raw memory pointer and length. Zero-length
    /// input returns handle 0 to match create("").
    uint32_t from_memory(const uint8_t *data, uint32_t len) {
        if (len == 0) return 0;
        const std::string s(reinterpret_cast<const char *>(data), len);
        const uint32_t handle = this->counter++;
        this->handles[handle] = s;
        return handle;
    }

    /// Gets the string value for a handle. Handle 0 is the empty string (per
    /// create), returned without a lookup. Unknown handles return "".
    std::string get(uint32_t handle) const {
        if (handle == 0) return "";
        const auto it = this->handles.find(handle);
        if (it != this->handles.end()) return it->second;
        const auto cit = this->literal_handles.find(handle);
        if (cit != this->literal_handles.end()) return cit->second;
        return "";
    }

    /// Checks if a string handle exists. Handle 0 always exists as the empty
    /// string, matching the create("") -> 0 -> get(0) == "" round-trip.
    bool exists(uint32_t handle) const {
        if (handle == 0) return true;
        return this->handles.contains(handle) || this->literal_handles.contains(handle);
    }

    /// Clears transient handles. Literal handles are preserved.
    void clear() {
        this->handles.clear();
        this->counter = 1;
    }

    /// Clears all handles including literal handles.
    void reset() {
        this->clear();
        this->literal_handles.clear();
        this->literal_counter = LITERAL_HANDLE_BASE;
    }
};

/// StateConsumer lets stl::Modules opt into receiving the runtime's shared
/// string state before bind_to. Inherit alongside stl::Module.
class StateConsumer {
public:
    virtual ~StateConsumer() = default;
    virtual void set_str_state(std::shared_ptr<State>) = 0;
};

}
