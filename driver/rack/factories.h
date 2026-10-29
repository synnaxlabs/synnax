// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <memory>
#include <string>
#include <vector>

#include "absl/log/log.h"

#include "driver/rack/rack.h"
#include "driver/task/task.h"

namespace driver::rack {
using FactoryList = std::vector<std::unique_ptr<task::Factory>>;

/// @brief adds the factory that factory_creator returns when the integration is enabled
/// and its SDK libraries load.
template<typename F>
void configure_integration(
    const Config &config,
    FactoryList &factories,
    const std::string &integration_name,
    F factory_creator
) {
    if (std::ranges::find(config.integrations, integration_name) ==
        config.integrations.end()) {
        VLOG(1) << "[" << integration_name << "] integration not enabled";
        return;
    }
    if (!config.libraries_loaded(integration_name)) {
        LOG(WARNING) << "[" << integration_name
                     << "] integration is enabled, but its libraries do not load on "
                        "this computer";
        return;
    }
    VLOG(1) << "[" << integration_name << "] integration supported";
    factories.push_back(factory_creator());
}

/// @brief adds the NI factory that this platform's build runs.
void configure_ni(const Config &config, FactoryList &factories);
}
