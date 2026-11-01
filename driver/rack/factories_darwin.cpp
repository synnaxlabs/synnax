// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/labjack.h"
#include "driver/ni/names.h"
#include "driver/rack/factories.h"

namespace driver::rack {
std::vector<std::string> platform_integrations() {
    return {labjack::INTEGRATION_NAME};
}

void configure_ni(const Config &, FactoryList &factories) {
    factories.push_back(
        std::make_unique<UnavailableFactory>(
            ni::INTEGRATION_NAME,
            "NI tasks do not run on macOS. Connect the device from a Driver on Windows "
            "or Linux.",
            ni::SCAN_TASK_TYPE
        )
    );
}

void configure_labjack(const Config &config, FactoryList &factories) {
    configure_integration(config, factories, labjack::INTEGRATION_NAME, [&config]() {
        return labjack::Factory::create(config.timing);
    });
}
}
