// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/names.h"
#include "driver/ni/ni.h"
#include "driver/rack/factories.h"

namespace driver::rack {
std::vector<std::string> platform_integrations() {
    return {ni::INTEGRATION_NAME};
}

void configure_ni(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, ni::INTEGRATION_NAME, [&config]() {
        return ni::Factory::create(config.timing);
    });
}

void configure_labjack(const Config &, FactoryList &factories) {
    factories.push_back(
        std::make_unique<UnavailableFactory>(
            labjack::INTEGRATION_NAME,
            "LabJack tasks do not run on Linux. Connect the device from a Driver on "
            "Windows or macOS.",
            labjack::SCAN_TASK_TYPE
        )
    );
}
}
