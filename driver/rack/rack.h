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

#include "driver/ni/ni.h"
#include "driver/sequence/sequence.h"

/// internal
#include "driver/ni/ni.h"
#include "driver/opc/opc.h"
#include "driver/sequence/sequence.h"
#include "driver/task/task.h"

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
        opc::INTEGRATION_NAME,
        ni::INTEGRATION_NAME,
        labjack::INTEGRATION_NAME,
        sequence::INTEGRATION_NAME
    };
#else
    return {
        opc::INTEGRATION_NAME,
        ni::INTEGRATION_NAME,
        sequence::INTEGRATION_NAME
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
        if (const auto err = cfg.load_persisted_state()) return {cfg, err};
        if (const auto err = cfg.load_config_file(argc, argv)) return {cfg, err};
        if (const auto err = cfg.load_remote(breaker)) return {cfg, err};
        const auto err = cfg.save_remote_info(cfg.remote);
        return {cfg, err};
    }

    static xerrors::Error save_conn_params(const synnax::Config &conn_params);

    static xerrors::Error save_remote_info(const RemoteInfo &remote_info);

    static xerrors::Error clear_persisted_state();

    /// @brief loads the configuration from the provided command line arguments.
    /// Looks for a "--config" flag followed by a configuration file path.
    [[nodiscard]] xerrors::Error load_persisted_state();

    [[nodiscard]] xerrors::Error load_config_file(int argc, char **argv);

    [[nodiscard]] xerrors::Error load_remote(breaker::Breaker &breaker);
};

/// @brief clears the persisted state file, removing all cached information.
xerrors::Error clear_persisted_state();


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

    bool should_exit(const xerrors::Error &err) {
        this->run_err = err;
        if (err) {
            if (err.matches(freighter::UNREACHABLE) && breaker.wait(err))
                return false;
            return true;
        }
        return false;
    }

    void run(const int argc, char **argv) {
        while (this->breaker.running()) {
            auto [cfg, err] = Config::load(argc, argv, this->breaker);
            if (err) {
                if (this->should_exit(err)) return;
                continue;
            }
            this->task_manager = std::make_unique<task::Manager>(
                cfg.rack,
                cfg.new_client(),
                cfg.new_factory()
            );
            err = this->task_manager->run();
            if (err && this->should_exit(err)) return;
        }
    }

public:
    void start(int argc, char **argv) {
        this->breaker.start();
        this->run_thread = std::thread([this, argv, argc] {
            this->run(argc, argv);
        });
    }

    xerrors::Error stop() {
        if (!this->breaker.running()) return xerrors::NIL;
        breaker.stop();
        if (task_manager != nullptr) task_manager->stop();
        this->run_thread.join();
        return xerrors::NIL;
    }
};
}
