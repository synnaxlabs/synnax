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
struct Config {
    synnax::RackKey rack_key;
    std::string cluster_key;
    synnax::Config connection;
    breaker::Config breaker_config;
    std::vector<std::string> integrations;
    bool debug;

    [[nodiscard]] bool integration_enabled(const std::string& integration) const;

    static std::pair<Config, xerrors::Error> load(int argc, char** argv);
};

struct PersistedState {
    synnax::RackKey rack_key;
    synnax::Config connection;
    std::string cluster_key;
};

xerrors::Error save_remote_info(const synnax::RackKey& rack_key, const std::string& cluster_key);

xerrors::Error save_conn_params(const synnax::Config& cfg);

xerrors::Error clear_persisted_state();
}