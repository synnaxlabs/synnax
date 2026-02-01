// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>

#include "x/cpp/xerrors/errors.h"

#include "driver/ethercat/engine/engine.h"

namespace ethercat::engine {

/// Manages a pool of EtherCAT engines.
///
/// Each engine is associated with a unique key from master::Info. Engines are created
/// lazily on first request and reused for subsequent tasks. The Pool owns a Manager
/// that discovers available masters and creates them.
class Pool {
    std::unique_ptr<master::Manager> manager;
    mutable std::mutex mu;
    std::unordered_map<std::string, std::shared_ptr<Engine>> engines;

public:
    /// Constructs a pool with the given manager.
    explicit Pool(std::unique_ptr<master::Manager> manager);

    /// Returns all available EtherCAT masters discovered by the manager.
    [[nodiscard]] std::vector<master::Info> enumerate() const;

    /// Acquires or creates an engine for the specified master key.
    /// @param key The master key (e.g., "igh:0" or "eth0").
    /// @return Pair of shared pointer to the engine and error.
    std::pair<std::shared_ptr<Engine>, xerrors::Error> acquire(const std::string &key);

    /// Checks if a master has an active (running) engine.
    /// @param key The master key.
    [[nodiscard]] bool is_active(const std::string &key) const;

    /// Returns cached slave information from a master's engine.
    /// @param key The master key.
    [[nodiscard]] std::vector<SlaveInfo> get_slaves(const std::string &key) const;
};

}
