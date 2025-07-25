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
#include <windows.h>
#include <winsock2.h>
#endif


/// module
#include "x/cpp/xargs/xargs.h"
#include "x/cpp/xlog/xlog.h"

/// external
#include "nlohmann/json.hpp"

/// internal
#include "driver/labjack/labjack.h"
#include "driver/ni/ni.h"
#include "driver/opc/opc.h"
#include "driver/rack/state/state.h"
#include "driver/sequence/sequence.h"
#include "driver/task/common/sample_clock.h"
#include "driver/task/task.h"

using json = nlohmann::json;

namespace rack {
struct RemoteInfo {
    synnax::RackKey rack_key = 0;
    std::string cluster_key;

    template<typename Parser>
    void override(Parser &p) {
        this->rack_key = p.optional("rack_key", this->rack_key);
        this->cluster_key = p.optional("cluster_key", this->cluster_key);
    }

    [[nodiscard]] json to_json() const {
        return {
            {"rack_key", this->rack_key},
            {"cluster_key", this->cluster_key},
        };
    }
};

inline std::vector<std::string> default_integrations() {
    return {
        opc::INTEGRATION_NAME,
        ni::INTEGRATION_NAME,
        sequence::INTEGRATION_NAME,
        labjack::INTEGRATION_NAME,
    };
}

/// @brief the configuration information necessary for running the driver. The
/// driver gets this configuration information from 3 places, in increasing order of
/// priority.
///
/// 1. Reasonable defaults.
/// 2. Persisted state. The driver maintains a persisted state file (used by both
/// the 'login' command and the task manager ot save rack information). Cached rack,
/// cluster, and connection information will be kept in this file.
/// 3. Configuration file. The driver can be provided with a configuration file
/// using the --config flag followed by a path to a configuration file. This file
/// can override the values in the persisted state file.
struct Config {
    /// @brief this is the rack that the driver will attach to on the server. If not
    /// provided, the driver will automatically create a new rack and persist it in
    /// state.
    synnax::Rack rack;
    /// @brief important info used to determine the identity of the driver when
    /// connecting to a cluster. This is cached on the local file system to compare
    /// and contrast.
    RemoteInfo remote_info;
    /// @brief timing options for tasks in the driver.
    common::TimingConfig timing;
    /// @brief connection parameters to the Synnax cluster.
    synnax::Config connection;
    /// @brief the list of integrations enabled for the driver.
    std::vector<std::string> integrations;

    /// @brief returns a new task factory to use for creating tasks in the task
    /// manager.
    [[nodiscard]] std::unique_ptr<task::Factory> new_factory() const;

    /// @brief returns a new Synnax client using the stored connection parameters.
    [[nodiscard]] std::shared_ptr<synnax::Synnax> new_client() const {
        return std::make_shared<synnax::Synnax>(this->connection);
    }

    /// @brief returns true if the integration with the given name is enabled.
    [[nodiscard]] bool integration_enabled(const std::string &i) const;

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << "configuration:\n"
           << cfg.connection << cfg.timing << "\n"
           << "  " << xlog::SHALE() << "enabled integrations" << xlog::RESET() << ": ";
        for (size_t i = 0; i < cfg.integrations.size(); ++i) {
            os << cfg.integrations[i];
            if (i < cfg.integrations.size() - 1) os << ", ";
        }
        os << "\n";
        return os;
    }

    static std::pair<Config, xerrors::Error>
    load(xargs::Parser &parser, breaker::Breaker &breaker) {
        rack::Config cfg{
            .connection =
                {
                    .host = "localhost",
                    .port = 9090,
                    .username = "synnax",
                    .password = "seldon",
                },
            .integrations = default_integrations(),
        };
        VLOG(1) << "loading configuration from persisted state";
        if (const auto err = cfg.load_persisted_state(parser)) return {cfg, err};
        VLOG(1) << "loading configuration from config file";
        if (const auto err = cfg.load_config_file(parser, breaker)) return {cfg, err};
        VLOG(1) << "loading configuration from environment";
        if (const auto err = cfg.load_env()) return {cfg, err};
        VLOG(1) << "loading configuration from command line";
        if (const auto err = cfg.load_args(parser)) return {cfg, err};
        if (breaker.retry_count() == 0) LOG(INFO) << cfg;
        if (const auto err = cfg.load_remote(breaker)) return {cfg, err};
        LOG(INFO) << xlog::BLUE() << "successfully reached cluster at "
                  << cfg.connection.address() << ". Continuing with driver startup"
                  << xlog::RESET();
        LOG(INFO) << "remote info" << "\n"
                  << xlog::SHALE() << "  rack: " << xlog::RESET() << cfg.rack.name
                  << " (" << cfg.remote_info.rack_key << ")\n"
                  << xlog::SHALE() << "  cluster: " << xlog::RESET()
                  << cfg.remote_info.cluster_key;
        VLOG(1) << "saving remote info";
        const auto err = Config::save_remote_info(parser, cfg.remote_info);
        VLOG(1) << "saved remote info";
        return {cfg, err};
    }

    void override_integrations(
        const std::vector<std::string> &enable,
        const std::vector<std::string> &disable
    ) {
        std::set i_set(this->integrations.begin(), this->integrations.end());
        for (const auto &integration: disable)
            i_set.erase(integration);
        for (const auto &integration: enable)
            i_set.insert(integration);
        this->integrations = std::vector(i_set.begin(), i_set.end());
    }

    /// @brief permanently saves connection parameters to the persisted state file.
    static xerrors::Error
    save_conn_params(xargs::Parser &args, const synnax::Config &conn_params);

    /// @brief permanently saves the remote info to the persisted state file.
    static xerrors::Error
    save_remote_info(xargs::Parser &args, const RemoteInfo &remote_info);

    static xerrors::Error clear_persisted_state(xargs::Parser &args);

    /// @brief loads the configuration from the provided command line arguments.
    /// Looks for a "--config" flag followed by a configuration file path.
    [[nodiscard]] xerrors::Error load_persisted_state(xargs::Parser &args);

    [[nodiscard]] xerrors::Error
    load_config_file(xargs::Parser &args, breaker::Breaker &breaker);

    [[nodiscard]] xerrors::Error load_env();

    [[nodiscard]] xerrors::Error load_args(xargs::Parser &args);

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
        .base_interval = telem::SECOND,
        .max_retries = 200,
        .scale = 1.1,
        .max_interval = telem::MINUTE,
    });
    xerrors::Error run_err = xerrors::NIL;

    /// @brief returns true if the error cannot be recovered from and the rack
    /// should stop operations and shut down.
    bool
    should_exit(const xerrors::Error &err, const std::function<void()> &on_shutdown);

    /// @brief starts the main loop for the rack.
    void run(xargs::Parser &args, const std::function<void()> &on_shutdown);

public:
    /// @brief starts the rack.
    /// @param args Parser containing command line arguments
    /// @param on_shutdown Optional callback that will be called if the rack shuts
    /// down prematurely
    void start(xargs::Parser &args, std::function<void()> on_shutdown = nullptr);

    /// @brief stops the rack.
    xerrors::Error stop();
};
}
