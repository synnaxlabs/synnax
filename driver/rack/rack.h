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

/// module
#include "x/cpp/xlog/xlog.h"
#include "x/cpp/xargs/xargs.h"

/// external
#include "nlohmann/json.hpp"

/// internal
#include "driver/ni/ni.h"
#include "driver/sequence/sequence.h"
#include "driver/heartbeat/heartbeat.h"
#include "driver/opc/opc.h"
#include "driver/task/task.h"


using json = nlohmann::json;

namespace rack {
struct RemoteInfo {
    synnax::RackKey rack_key = 0;
    std::string cluster_key = "";

    void override(xjson::Parser &p) {
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

    /// @brief returns a new task factory to use for creating tasks in the task manager.
    [[nodiscard]] std::unique_ptr<task::Factory> new_factory() const;

    /// @brief returns a new Synnax client using the stored connection parameters.
    [[nodiscard]] std::shared_ptr<synnax::Synnax> new_client() const {
        return std::make_shared<synnax::Synnax>(this->connection);
    }

    /// @brief returns true if the integration with the given name is enabled.
    [[nodiscard]] bool integration_enabled(const std::string &i) const;

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << "[driver] configuration:\n"
                << "  " << xlog::SHALE << "cluster address" << xlog::RESET << ": " <<
                cfg.connection.host << ":" << cfg.connection
                .port << "\n"
                << "  " << xlog::SHALE << "username" << xlog::RESET << ": " << cfg.
                connection.username << "\n"
                << "  " << xlog::SHALE << "rack" << xlog::RESET << ": " << cfg.rack.name
                << " (" << cfg.rack.key << ")\n"
                << "  " << xlog::SHALE << "cluster key" << xlog::RESET << ": " << cfg.
                remote.cluster_key << "\n"
                << "  " << xlog::SHALE << "enabled integrations" << xlog::RESET << ": ";
        for (size_t i = 0; i < cfg.integrations.size(); ++i) {
            os << cfg.integrations[i];
            if (i < cfg.integrations.size() - 1) os << ", ";
        }
        os << "\n";
        return os;
    }

    static std::pair<Config, xerrors::Error> load(
        xargs::Parser &parser,
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
        if (const auto err = cfg.load_persisted_state(parser)) return {cfg, err};
        if (const auto err = cfg.load_config_file(parser)) return {cfg, err};
        if (const auto err = cfg.load_remote(breaker)) return {cfg, err};
        const auto err = cfg.save_remote_info(parser, cfg.remote);
        return {cfg, err};
    }

    /// @brief permanently saves connection parameters to the persisted state file.
    static xerrors::Error save_conn_params(
        xargs::Parser &args,
        const synnax::Config &conn_params
    );

    /// @brief permanently saves the remote info to the persisted state file.
    static xerrors::Error save_remote_info(
        xargs::Parser &args,
        const RemoteInfo &remote_info
    );

    static xerrors::Error clear_persisted_state(xargs::Parser &args);

    /// @brief loads the configuration from the provided command line arguments.
    /// Looks for a "--config" flag followed by a configuration file path.
    [[nodiscard]] xerrors::Error load_persisted_state(xargs::Parser &args);

    [[nodiscard]] xerrors::Error load_config_file(xargs::Parser &args);

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
        .max_retries = 200,
        .scale = 1.1,
        .max_interval = telem::TimeSpan::minutes(1)
    });
    xerrors::Error run_err = xerrors::NIL;

    /// @brief returns true if the error cannot be recovered from and the rack
    /// should stop operations and shut down.
    bool should_exit(const xerrors::Error &err);

    /// @brief starts the main loop for the rack.
    void run(xargs::Parser &args);

public:
    /// @brief starts the rack.
    void start(xargs::Parser &args);

    /// @brief stops the rack.
    xerrors::Error stop();
};
}
