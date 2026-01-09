// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once
#include "driver/arc/arc.h"

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <windows.h>
#include <winsock2.h>
#endif

#include "nlohmann/json.hpp"

#include "x/cpp/args/args.h"
#include "x/cpp/log/log.h"

#include "driver/labjack/labjack.h"
#ifndef SYNNAX_NILINUXRT
#include "driver/modbus/modbus.h"
#endif
#include "driver/ni/ni.h"
#include "driver/opc/opc.h"
#include "driver/rack/status/status.h"
#include "driver/sequence/sequence.h"
#include "driver/task/common/sample_clock.h"
#include "driver/task/task.h"

using json = nlohmann::json;

namespace driver::rack {
struct RemoteInfo {
    synnax::RackKey rack_key = 0;
    std::string cluster_key;

    template<typename Parser>
    void override(Parser &p) {
        this->rack_key = p.field("rack_key", this->rack_key);
        this->cluster_key = p.field("cluster_key", this->cluster_key);
    }

    [[nodiscard]] json to_json() const {
        return {
            {"rack_key", this->rack_key},
            {"cluster_key", this->cluster_key},
        };
    }
};

inline std::vector<std::string> default_integrations() {
    std::vector<std::string> integrations = {
        driver::opc::INTEGRATION_NAME,
        driver::ni::INTEGRATION_NAME,
        driver::sequence::INTEGRATION_NAME,
        driver::labjack::INTEGRATION_NAME,
        driver::arc::INTEGRATION_NAME,
    };
#ifndef SYNNAX_NILINUXRT
    integrations.push_back(driver::modbus::INTEGRATION_NAME);
#endif
    return integrations;
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
    driver::task::common::TimingConfig timing;
    /// @brief configuration for the task manager.
    task::ManagerConfig manager;
    /// @brief connection parameters to the Synnax cluster.
    synnax::Config connection;
    /// @brief the list of integrations enabled for the driver.
    std::vector<std::string> integrations;

    /// @brief returns a new task factory to use for creating tasks in the task
    /// manager.
    [[nodiscard]] std::unique_ptr<driver::task::Factory> new_factory() const;

    /// @brief returns a new Synnax client using the stored connection parameters.
    [[nodiscard]] std::shared_ptr<synnax::Synnax> new_client() const {
        return std::make_shared<synnax::Synnax>(this->connection);
    }

    /// @brief returns true if the integration with the given name is enabled.
    [[nodiscard]] bool integration_enabled(const std::string &i) const;

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << "configuration:\n"
           << cfg.connection << cfg.timing << "\n"
           << cfg.manager << "\n"
           << "  " << xlog::SHALE() << "enabled integrations" << xlog::RESET() << ": ";
        for (size_t i = 0; i < cfg.integrations.size(); ++i) {
            os << cfg.integrations[i];
            if (i < cfg.integrations.size() - 1) os << ", ";
        }
        os << "\n";
        return os;
    }

    static std::pair<Config, x::errors::Error>
    load(x::args::Parser &parser, x::breaker::Breaker &breaker) {
        driver::rack::Config cfg{
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
        LOG(INFO) << x::log::BLUE() << "successfully reached cluster at "
                  << cfg.connection.address() << ". Continuing with driver startup"
                  << x::log::RESET();
        LOG(INFO) << "remote info" << "\n"
                  << x::log::SHALE() << "  rack: " << x::log::RESET() << cfg.rack.name
                  << " (" << cfg.remote_info.rack_key << ")\n"
                  << x::log::SHALE() << "  cluster: " << x::log::RESET()
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
    static x::errors::Error
    save_conn_params(x::args::Parser &args, const synnax::Config &conn_params);

    /// @brief permanently saves the remote info to the persisted state file.
    static x::errors::Error
    save_remote_info(x::args::Parser &args, const RemoteInfo &remote_info);

    static x::errors::Error clear_persisted_state(x::args::Parser &args);

    /// @brief loads the configuration from the provided command line arguments.
    /// Looks for a "--config" flag followed by a configuration file path.
    [[nodiscard]] x::errors::Error load_persisted_state(x::args::Parser &args);

    [[nodiscard]] x::errors::Error
    load_config_file(x::args::Parser &args, x::breaker::Breaker &breaker);

    [[nodiscard]] x::errors::Error load_env();

    [[nodiscard]] x::errors::Error load_args(x::args::Parser &args);

    [[nodiscard]] x::errors::Error load_remote(x::breaker::Breaker &breaker);
};

/// @brief clears the persisted state file, removing all cached information.
x::errors::Error clear_persisted_state();

/// @brief rack is the entry point for driver operation. It is responsible for
/// communicating its identity to the Synnax cluster and managing the lifecycle
/// of tasks that are assigned to it.
class Rack {
    std::thread run_thread;
    std::unique_ptr<driver::task::Manager> task_manager;
    x::breaker::Breaker breaker = x::breaker::Breaker({
        .name = "driver",
        .base_interval = x::telem::SECOND,
        .max_retries = 200,
        .scale = 1.1f,
        .max_interval = x::telem::MINUTE,
    });
    x::errors::Error run_err = x::errors::NIL;

    /// @brief returns true if the error cannot be recovered from and the rack
    /// should stop operations and shut down.
    bool
    should_exit(const x::errors::Error &err, const std::function<void()> &on_shutdown);

    /// @brief starts the main loop for the rack.
    void run(x::args::Parser &args, const std::function<void()> &on_shutdown);

public:
    /// @brief destructor ensures thread is properly joined
    ~Rack();

    /// @brief starts the rack.
    /// @param args Parser containing command line arguments
    /// @param on_shutdown Optional callback that will be called if the rack shuts
    /// down prematurely
    void start(x::args::Parser &args, std::function<void()> on_shutdown = nullptr);

    /// @brief stops the rack.
    x::errors::Error stop();
};
}
