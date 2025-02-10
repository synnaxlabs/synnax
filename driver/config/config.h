// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"

namespace driver {
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
    synnax::RackKey rack_key;
    /// @brief the key of the cluster the driver is expected to connect to. If this
    /// does not match the key of the cluster, it will cause the driver to abandon
    /// it's persisted state and re-register with the server.
    std::string cluster_key;
    /// @brief connection parameters to the Synnax cluster.
    synnax::Config connection;
    /// @brief r
    breaker::Config retry_config;
    /// @brief the list of integrations enabled for the driver.
    std::vector<std::string> integrations;
    /// @brief whether to enable debug logging.
    bool debug;

    /// @brief returns true if the given integration should be enabled.
    [[nodiscard]] bool integration_enabled(const std::string &integration) const;

    /// @brief loads the configuration from the provided command line arguments.
    /// Looks for a "--config" flag followed by a configuration file path.
    static std::pair<Config, xerrors::Error> load(int argc, char **argv);
};

struct PersistedState {
    synnax::RackKey rack_key;
    synnax::Config connection;
    std::string cluster_key;
};

/// @brief saves information about the cluster and rack to the persisted state file,
/// making it available when the configuration is loaded next.
xerrors::Error save_remote_info(
    const synnax::RackKey &rack_key,
    const std::string &cluster_key
);

/// @brief saves connection parameters to the persisted state file, making them
/// available for when the configuration is loaded next.
xerrors::Error save_conn_params(const synnax::Config &cfg);

/// @brief clears the persisted state file, removing all cached information.
xerrors::Error clear_persisted_state();
}
