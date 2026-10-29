// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/labjack.h"
#include "driver/rack/factories.h"

namespace driver::rack {
std::vector<std::string> platform_integrations() {
    return {labjack::INTEGRATION_NAME};
}

void configure_ni(const Config &, FactoryList &factories) {
    // The macOS build leaves NI out, so its names are literals.
    factories.push_back(std::make_unique<LegacyFactory>("ni", "ni_scanner"));
}

void configure_labjack(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, labjack::INTEGRATION_NAME, [&config]() {
        return labjack::Factory::create(config.timing);
    });
}
}
