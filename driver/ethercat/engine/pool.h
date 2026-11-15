// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <mutex>
#include <string>
#include <unordered_map>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"
#include "driver/ethercat/engine/engine.h"

namespace driver::ethercat::engine {
/// @brief library info for the packet capture dependency required by SOEM.
const LibraryInfo PCAP_LIBRARY_INFO = {
    "Npcap (packet capture for EtherCAT)",
    "https://npcap.com/#download"
};

/// @brief returns the saved settings of the slaves on a network, as for
/// Config::slave_configs. It gets the key of the network and the name that messages to
/// the user give it.
using SlaveConfigs = std::function<std::pair<std::vector<SavedSlave>, x::errors::Error>(
    const std::string &network,
    const std::string &name
)>;

/// @brief manages a pool of EtherCAT engines keyed by master identifier.
class Pool {
    std::unique_ptr<master::Manager> manager;
    std::shared_ptr<x::thread::rt::Manager> rt_manager;
    SlaveConfigs slave_configs;
    x::telem::TimeSpan start_timeout;
    mutable std::mutex mu;
    std::unordered_map<std::string, std::shared_ptr<Engine>> engines;

public:
    /// @brief constructs a pool with the given manager and optional RT manager. Each
    /// engine reads its slave settings from slave_configs when its bus starts, and
    /// stops retrying a failed start after start_timeout.
    Pool(
        std::unique_ptr<master::Manager> manager,
        SlaveConfigs slave_configs,
        std::shared_ptr<x::thread::rt::Manager> rt_manager = nullptr,
        x::telem::TimeSpan start_timeout = Config().start_timeout
    );

    /// @brief returns all available EtherCAT masters discovered by the manager.
    [[nodiscard]] std::vector<master::Info> enumerate() const;

    /// @brief returns why no master can be created, or NIL when masters are available.
    /// A pool built without a manager reports the missing packet capture library.
    [[nodiscard]] x::errors::Error check_available() const;

    /// @brief acquires or creates an engine for the specified master key.
    std::pair<std::shared_ptr<Engine>, x::errors::Error>
    acquire(const std::string &key);

    /// @brief returns the latest condition of each enabled slave on a master, or an
    /// empty list when its bus is not running.
    [[nodiscard]] std::vector<slave::Health> health(const std::string &key) const;

    /// @brief returns the slaves on the master with key, as Engine::discover() does.
    [[nodiscard]] std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error>
    discover(const std::string &key);

    /// @brief returns the slaves on the master with key, as Engine::rediscover() does.
    [[nodiscard]] std::pair<std::vector<slave::DiscoveryResult>, x::errors::Error>
    rediscover(const std::string &key);
};

}
