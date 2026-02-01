// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <mutex>
#include <string>
#include <unordered_map>

#include "driver/ethercat/engine/engine.h"

namespace ethercat::engine {

/// Manages a pool of EtherCAT engines.
///
/// Each engine is associated with a unique key based on interface name or backend type.
/// Engines are created lazily on first request and reused for subsequent tasks.
class Pool {
    master::Factory factory;
    mutable std::mutex mu;
    std::unordered_map<std::string, std::shared_ptr<Engine>> engines;

public:
    /// Constructs a pool with the given master factory function.
    explicit Pool(master::Factory factory);

    /// Acquires or creates an engine for the specified interface/backend.
    /// @param interface_name Network interface name (used by SOEM).
    /// @param backend Backend type: "soem", "igh", or "auto".
    /// @return Shared pointer to the engine.
    std::shared_ptr<Engine> acquire(
        const std::string &interface_name,
        const std::string &backend = "auto"
    );

    /// Checks if an interface has an active (running) engine.
    bool is_active(const std::string &interface) const;

    /// Returns cached slave information from an interface's engine.
    std::vector<SlaveInfo> get_slaves(const std::string &interface) const;
};
}