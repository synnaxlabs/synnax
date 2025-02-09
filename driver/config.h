// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <fstream>
#include <filesystem>

/// external
#include "nlohmann/json.hpp"
#include "glog/logging.h"

/// internal
#include "driver/opc/opc.h"
#include "driver/ni/ni.h"
#include "client/cpp/synnax.h"
#include "driver/breaker/breaker.h"
#include "sequence/sequence.h"

#ifdef _WIN32
#include "driver/labjack/labjack.h"
#endif

using json = nlohmann::json;

namespace configd {

/// @brief logging related configuration parameters;
struct LoggingConfig {
    /// @brief whether to enable extra debug logging.
    bool debug;
    /// @brief the path to store logs in.
    std::string log_path;
};

/// @brief configuration for running the driver.
struct Config {
    /// @brief the key of the rack to identify the driver. If not provided, a new rack
    /// will be created and stored in the persisted state. If provided and a rack exists
    /// in the persisted state, this will override the persisted rack key.
    synnax::RackKey rack_key;
    /// @brief the connection parameters for the Synnax cluster.
    synnax::Config client_config;
    /// @brief breaker configuration for retrying failed requests.
    breaker::Config breaker_config;
    /// @brief the list of integrations to enable. All integrations are enabled by
    /// default.
    std::vector<std::string> integrations;
    /// @brief returns true if the given integration is enabled.
    [[nodiscard]] bool integration_enabled(const std::string &integration) const;
};

/// @brief state that the driver persists between multiple runs.
struct PersistedState {
    /// @brief the key identifying the rack within the Synnax cluster.
    synnax::RackKey rack_key;
    /// @brief connection parameters for the Synnax cluster.
    synnax::Config connection;
};

std::string get_persisted_state_path();
synnax::Config parse_synnax_config(config::Parser &conn);
std::pair<PersistedState, freighter::Error> load_persisted_state();
freighter::Error save_persisted_state(const PersistedState &state);
std::pair<configd::Config, freighter::Error> parse(const json &content);
json read(const std::string &path);

extern const std::vector<std::string> DEFAULT_INTEGRATIONS;
}  // namespace configd
