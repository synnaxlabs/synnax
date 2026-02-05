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

/// @brief manages a pool of EtherCAT engines keyed by master identifier.
class Pool {
    std::unique_ptr<master::Manager> manager;
    mutable std::mutex mu;
    std::unordered_map<std::string, std::shared_ptr<Engine>> engines;

public:
    /// @brief constructs a pool with the given manager.
    explicit Pool(std::unique_ptr<master::Manager> manager);

    /// @brief returns all available EtherCAT masters discovered by the manager.
    [[nodiscard]] std::vector<master::Info> enumerate() const;

    /// @brief acquires or creates an engine for the specified master key.
    std::pair<std::shared_ptr<Engine>, xerrors::Error> acquire(const std::string &key);

    /// @brief checks if a master has an active (running) engine.
    [[nodiscard]] bool is_active(const std::string &key) const;

    /// @brief returns cached slave information from a master's engine.
    [[nodiscard]] std::vector<slave::DiscoveryResult>
    get_slaves(const std::string &key) const;

    /// @brief discovers slaves on a master, handling engine lifecycle internally.
    /// If engine is running, returns cached slaves. If not, initializes first.
    /// @param key The master key (e.g., "igh:0" or "eth0").
    /// @return Pair of slave list and error.
    [[nodiscard]] std::pair<std::vector<slave::DiscoveryResult>, xerrors::Error>
    discover_slaves(const std::string &key);

private:
    /// @brief acquires or creates an engine without locking. Caller must hold mu.
    std::pair<std::shared_ptr<Engine>, xerrors::Error>
    acquire_unlocked(const std::string &key);
};

}
