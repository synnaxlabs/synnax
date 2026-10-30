// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <vector>

#include "absl/log/log.h"

#include "driver/rack/rack.h"
#include "driver/rack/unavailable.h"
#include "driver/task/task.h"

namespace driver::rack {
using FactoryList = std::vector<std::unique_ptr<task::Factory>>;

/// @brief adds the factory that factory_creator returns when the integration is
/// enabled, or an UnavailableFactory that names the configuration when it is not.
template<typename F>
void configure_integration(
    const Config &config,
    FactoryList &factories,
    const std::string &integration_name,
    F factory_creator
) {
    if (!config.integration_enabled(integration_name)) {
        VLOG(1) << "[" << integration_name << "] integration disabled";
        factories.push_back(
            std::make_unique<UnavailableFactory>(
                integration_name,
                "The configuration of this Driver turns off the '" + integration_name +
                    "' integration. Turn it on in the Driver configuration, then restart "
                    "the Driver."
            )
        );
        return;
    }
    VLOG(1) << "[" << integration_name << "] integration enabled";
    factories.push_back(factory_creator());
}

/// @brief adds the NI factory, or a stand-in where this platform's build leaves NI out.
void configure_ni(const Config &config, FactoryList &factories);

/// @brief adds the LabJack factory, or a stand-in where this platform's build leaves
/// LabJack out.
void configure_labjack(const Config &config, FactoryList &factories);
}
