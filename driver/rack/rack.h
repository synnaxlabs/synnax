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

#include <algorithm>
#include <filesystem>
#include <iterator>
#include <string>
#include <vector>

#include "absl/log/log.h"

#include "client/cpp/auth/auth.h"
#include "x/cpp/args/args.h"
#include "x/cpp/log/log.h"
#include "x/cpp/thread/rt/rt.h"
#include "x/cpp/uuid/uuid.h"

#ifndef SYNNAX_NILINUXRT
#include "driver/modbus/modbus.h"
#endif

#include "driver/common/sample_clock.h"
#include "driver/ethercat/ethercat.h"
#include "driver/http/http.h"
#include "driver/opcua/opcua.h"
#include "driver/task/task.h"

namespace driver::rack {
/// @brief the persisted state file or its directory cannot be used safely, such as
/// when either is a symbolic link.
const x::errors::Error STATE_ERROR = x::errors::SY.sub("driver").sub("state");

/// @brief the directory in the state directory that holds OPC UA certificates. The
/// Linux service user owns it.
const std::string OPCUA_STATE_DIR = "opcua";

struct RemoteInfo {
    /// @brief a saved rack that still needs enrollment. It is saved again only while
    /// the Driver cannot enroll it.
    synnax::rack::Key rack_key = 0;
    x::uuid::UUID cluster_key;

    template<typename Parser>
    void override(Parser &p) {
        this->rack_key = p.field("rack_key", this->rack_key);
        auto [ck, ck_err] = x::uuid::UUID::parse(
            p.field("cluster_key", this->cluster_key.to_string())
        );
        if (ck_err)
            p.field_err("cluster_key", ck_err);
        else
            this->cluster_key = ck;
    }

    [[nodiscard]] x::json::json to_json() const {
        x::json::json j = {{"cluster_key", this->cluster_key.to_string()}};
        if (this->rack_key != 0) j["rack_key"] = this->rack_key;
        return j;
    }
};

/// @brief returns the integrations that this platform's build adds to the ones every
/// build has. Each platform defines it in its own source.
std::vector<std::string> platform_integrations();

/// @brief returns the integrations that this build runs. A config enables all of them
/// unless it names its own.
inline std::vector<std::string> built_integrations() {
    std::vector<std::string> integrations = {
        opcua::INTEGRATION_NAME,
        arc::INTEGRATION_NAME,
        ethercat::INTEGRATION_NAME,
        http::INTEGRATION_NAME
    };
    std::ranges::copy(platform_integrations(), std::back_inserter(integrations));
#ifndef SYNNAX_NILINUXRT
    integrations.push_back(modbus::INTEGRATION_NAME);
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
    /// @brief the rack the driver runs as. It is the subject of the driver's API key.
    synnax::rack::Rack rack;
    /// @brief important info used to determine the identity of the driver when
    /// connecting to a cluster. This is cached on the local file system to compare
    /// and contrast.
    RemoteInfo remote_info;
    /// @brief timing options for tasks in the driver.
    common::TimingConfig timing;
    /// @brief configuration for the task manager.
    task::ManagerConfig manager;
    /// @brief connection parameters to the Synnax cluster.
    synnax::Config connection;
    /// @brief the list of integrations enabled for the driver.
    std::vector<std::string> integrations;
    /// @brief the directory that holds the persisted state file. Integrations keep
    /// their own files, such as certificates, inside it.
    std::filesystem::path state_dir;

    /// @brief returns a new task factory to use for creating tasks in the task
    /// manager.
    [[nodiscard]] std::unique_ptr<task::Factory>
    new_factory(const std::shared_ptr<x::thread::rt::Manager> &rt_manager) const;

    /// @brief returns a new Synnax client using the stored connection parameters.
    [[nodiscard]] std::shared_ptr<synnax::Synnax> new_client() const {
        return std::make_shared<synnax::Synnax>(this->connection);
    }

    /// @brief returns true if the integration with the given name is enabled.
    [[nodiscard]] bool integration_enabled(const std::string &i) const;

    friend std::ostream &operator<<(std::ostream &os, const Config &cfg) {
        os << "configuration:\n"
           << cfg.connection << "\n"
           << cfg.timing << "\n"
           << cfg.manager << "\n"
           << "  " << x::log::SHALE() << "enabled integrations" << x::log::RESET()
           << ": ";
        for (size_t i = 0; i < cfg.integrations.size(); ++i) {
            os << cfg.integrations[i];
            if (i < cfg.integrations.size() - 1) os << ", ";
        }
        return os;
    }

    /// @brief builds the configuration from defaults, persisted state, the config file,
    /// the environment, and command line arguments, each overriding the last. Does not
    /// reach the Core. On error, returns the configuration built so far.
    static std::pair<Config, x::errors::Error>
    resolve(x::args::Parser &parser, x::breaker::Breaker &breaker) {
        rack::Config cfg{
            .connection =
                {
                    .host = "localhost",
                    .port = 9090,
                    .username = "",
                    .password = "",
                },
            .integrations = built_integrations(),
        };
        VLOG(1) << "loading configuration from persisted state";
        if (const auto err = cfg.load_persisted_state(parser)) return {cfg, err};
        VLOG(1) << "loading configuration from config file";
        if (const auto err = cfg.load_config_file(parser, breaker)) return {cfg, err};
        VLOG(1) << "loading configuration from environment";
        if (const auto err = cfg.load_env()) return {cfg, err};
        VLOG(1) << "loading configuration from command line";
        if (const auto err = cfg.load_args(parser)) return {cfg, err};
        cfg.resolve_credentials(breaker.retry_count() == 0);
        return {cfg, x::errors::NIL};
    }

    /// @brief makes the API key win over a username and password, warning when it
    /// drops them if warn is true. Without a key, a missing username defaults to
    /// "synnax" and a missing password to "seldon".
    void resolve_credentials(const bool warn) {
        auto &conn = this->connection;
        if (!conn.api_key.empty()) {
            if (warn && (!conn.username.empty() || !conn.password.empty()))
                LOG(WARNING) << "an API key is set, so the username and password "
                                "are ignored. Remove them from the configuration";
            conn.username.clear();
            conn.password.clear();
            return;
        }
        if (conn.username.empty()) conn.username = "synnax";
        if (conn.password.empty()) conn.password = "seldon";
    }

    static std::pair<Config, x::errors::Error>
    load(x::args::Parser &parser, x::breaker::Breaker &breaker) {
        auto [cfg, err] = resolve(parser, breaker);
        if (err) return {cfg, err};
        if (breaker.retry_count() == 0) LOG(INFO) << cfg;
        if (cfg.connection.api_key.empty()) {
            err = cfg.enroll();
            if (err && cfg.rack.key != 0 &&
                err.matches(synnax::auth::ERR_ACCESS_DENIED))
                LOG(WARNING) << "Failed to enroll Driver. " << err.data
                             << " Until then, the Driver logs in with the password.";
            else if (err)
                return {cfg, err};
            else {
                if (err = save_api_key(parser, cfg.connection); err) return {cfg, err};
                LOG(INFO) << x::log::GREEN() << "enrolled rack " << cfg.rack.name
                          << " and saved its API key" << x::log::RESET();
            }
        }
        err = cfg.load_remote();
        if (err) return {cfg, err};
        LOG(INFO) << x::log::BLUE() << "successfully reached cluster at "
                  << cfg.connection.address() << ". Continuing with driver startup"
                  << x::log::RESET();
        LOG(INFO) << "remote info" << "\n"
                  << x::log::SHALE() << "  rack: " << x::log::RESET() << cfg.rack.name
                  << " (" << cfg.rack.key << ")\n"
                  << x::log::SHALE() << "  cluster: " << x::log::RESET()
                  << cfg.remote_info.cluster_key;
        VLOG(1) << "saving remote info";
        err = Config::save_remote_info(parser, cfg.remote_info);
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

    /// @brief saves the connection and its API key to the persisted state file, without
    /// a username or password.
    static x::errors::Error
    save_api_key(x::args::Parser &args, const synnax::Config &conn);

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

    /// @brief logs in with the username and password, then creates an API key for the
    /// saved rack that needs enrollment, or for a new rack when there is none. Sets the
    /// key and the rack, and clears the username and password. Returns
    /// synnax::auth::ERR_ACCESS_DENIED when the user cannot create the key, and sets
    /// the rack only if it was saved.
    [[nodiscard]] x::errors::Error enroll();

    /// @brief logs in with the API key and loads the rack it logs in as. Without a key,
    /// it logs in with the password and loads the saved rack.
    [[nodiscard]] x::errors::Error load_remote();
};

/// @brief clears the persisted state file, removing all cached information.
x::errors::Error clear_persisted_state();

/// @brief rack is the entry point for driver operation. It is responsible for
/// communicating its identity to the Synnax cluster and managing the lifecycle
/// of tasks that are assigned to it.
class Rack {
    std::thread run_thread;
    std::unique_ptr<task::Manager> task_manager;
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
