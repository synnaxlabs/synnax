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
#include <cstring>
#include <unordered_map>

#include "x/cpp/telem/series.h"

namespace arc::stl::series {

/// Transient handle store for series created during a single execution cycle.
/// Handles are uint32_t keys that WASM code uses to reference series objects.
class State {
    std::unordered_map<uint32_t, telem::Series> handles;
    uint32_t counter = 1;

public:
    /// Stores a series and returns its handle.
    uint32_t store(telem::Series s) {
        const uint32_t handle = this->counter++;
        this->handles.emplace(handle, std::move(s));
        return handle;
    }

    /// Gets a mutable series by handle. Returns nullptr if not found.
    telem::Series *get(uint32_t handle) {
        const auto it = this->handles.find(handle);
        if (it == this->handles.end()) return nullptr;
        return &it->second;
    }

    /// Gets a const series by handle. Returns nullptr if not found.
    const telem::Series *get(uint32_t handle) const {
        const auto it = this->handles.find(handle);
        if (it == this->handles.end()) return nullptr;
        return &it->second;
    }

    /// Clears all transient handles. Called at end of each execution cycle.
    void clear() {
        this->handles.clear();
        this->counter = 1;
    }
};

}
