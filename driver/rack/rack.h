// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <winsock2.h>
#include <windows.h>
#endif

#ifdef _WIN32
#include "driver/labjack/labjack.h"
#endif

/// internal
#include "driver/ni/ni.h"
#include "driver/sequence/sequence.h"
#include "driver/heartbeat/heartbeat.h"
#include "driver/opc/opc.h"
#include "driver/task/task.h"

/// external
#include "nlohmann/json.hpp"

using json = nlohmann::json;

namespace rack {
struct RemoteInfo {
    synnax::RackKey rack_key = 0;
    std::string cluster_key = "";

    void override(config::Parser &p) {
        this->rack_key = p.optional("rack_key", this->rack_key);
        this->cluster_key = p.optional("cluster_key", this->cluster_key);
    }

    json to_json() const {
        return {
            {"rack_key", this->rack_key},
            {"cluster_key", this->cluster_key},
        };
    }
};

inline std::vector<std::string> default_integrations() {
#ifdef _WIN32
    return {
        heartbeat::INTEGRATION_NAME,
        opc::INTEGRATION_NAME,
        ni::INTEGRATION_NAME,
        labjack::INTEGRATION_NAME,
        sequence::INTEGRATION_NAME
    };
#else
    return {
        opc::INTEGRATION_NAME,
        ni::INTEGRATION_NAME,
        sequence::INTEGRATION_NAME,
        heartbeat::INTEGRATION_NAME
    };
#endif
}

/// @brief the configuration information necessary for running the driver. The driver
/// gets this configuration information from 3 places, in increasing order of priority.
///
/// 1. Reasonable defaults.
/// 2. Persisted state. The driver maintains a persisted state file (used by both the
/// 'login' command and the task manager ot save rack information). Cached rack, cluster,
/// and connection information will be kept in this file.
/// 3. Configuration file. The driver can be provided with a configuration file using
/// the --config flag followed by a path to a configuration file. This file can override
/// the values in the persisted state file.
struct Config {
    /// @brief this is the rack that the driver will attach to on the server. If not
    /// provided, the driver will automatically create a new rack and persist it in state.
    synnax::Rack rack;
    /// @brief important info used to determine the identity of the driver when
    /// connecting to a cluster. This is cached on the local file system to compare
    /// and contrast.
    RemoteInfo remote;
    /// @brief connection parameters to the Synnax cluster.
    synnax::Config connection;
    /// @brief the list of integrations enabled for the driver.
    std::vector<std::string> integrations;

    [[nodiscard]] std::unique_ptr<task::Factory> new_factory() const;

    [[nodiscard]] std::shared_ptr<synnax::Synnax> new_client() const {
        return std::make_shared<synnax::Synnax>(this->connection);
    }

    [[nodiscard]] bool integration_enabled(const std::string &integration) const;

    static std::pair<Config, xerrors::Error> load(
        int argc,
        char **argv,
        breaker::Breaker &breaker
    ) {
        rack::Config cfg{
            .connection = {
                .host = "localhost",
                .port = 9090,
                .username = "synnax",
                .password = "seldon",
            },
            .integrations = default_integrations(),
        };
        if (const auto err = cfg.load_persisted_state(argc, argv)) return {cfg, err};
        if (const auto err = cfg.load_config_file(argc, argv)) return {cfg, err};
        if (const auto err = cfg.load_remote(breaker)) return {cfg, err};
        const auto err = cfg.save_remote_info(argc, argv, cfg.remote);
        return {cfg, err};
    }

    static xerrors::Error save_conn_params(int argc, char **argv, const synnax::Config &conn_params);

    static xerrors::Error save_remote_info(int argc, char **argv, const RemoteInfo &remote_info);

    static xerrors::Error clear_persisted_state(int argc, char **argv);

    /// @brief loads the configuration from the provided command line arguments.
    /// Looks for a "--config" flag followed by a configuration file path.
    [[nodiscard]] xerrors::Error load_persisted_state(int argc, char **argv);

    [[nodiscard]] xerrors::Error load_config_file(int argc, char **argv);

    [[nodiscard]] xerrors::Error load_remote(breaker::Breaker &breaker);
};

/// @brief clears the persisted state file, removing all cached information.
xerrors::Error clear_persisted_state();

/// @brief rack is the entry point for driver operation. It is responsible for
/// communicating its identity to the Synnax cluster and managing the lifecycle
/// of tasks that are assigned to it.
class Rack {
    std::thread run_thread;
    std::unique_ptr<task::Manager> task_manager;
    breaker::Breaker breaker = breaker::Breaker({
        .name = "driver",
        .base_interval = telem::TimeSpan::seconds(1),
        .max_retries = 100,
        .scale = 1.05,
    });
    xerrors::Error run_err = xerrors::NIL;

    /// @brief returns true if the error cannot be recovered from and the rack
    /// should stop operations and shut down.
    bool should_exit(const xerrors::Error &err);

    /// @brief starts the main loop for the rack.
    void run(int argc, char **argv);
public:
    /// @brief starts the rack.
    void start(int argc, char **argv);

    /// @brief stops the rack.
    xerrors::Error stop();
};
}
