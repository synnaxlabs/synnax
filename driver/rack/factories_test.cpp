// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <string>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "driver/rack/rack.h"

namespace driver::rack {
/// @brief it should build a factory for each integration in the build whether or not
/// its library loads, and no factory for an integration the build leaves out.
TEST(RackFactoriesTest, BuildsFactoriesForBuiltIntegrations) {
    // The names are literals because a build can leave either integration out.
    const std::vector<std::pair<std::string, std::string>> scanners = {
        {"ni", "ni_scanner"},
        {"labjack", "labjack_scan"},
    };
    const Config cfg{.integrations = {"ni", "labjack"}};
    const auto factory = cfg.new_factory(std::make_shared<x::thread::rt::Manager>());
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const auto built = built_integrations();
    for (const auto &[integration, type]: scanners) {
        // An invalid config stops the configure before it loads a library or builds a
        // scanner, which needs a client.
        const auto [tsk, handled] = factory->configure_task(
            ctx,
            synnax::task::Task{
                .name = type,
                .type = type,
                .config = {{"rate", "fast"}}
            },
            task::NO_COMMAND
        );
        EXPECT_EQ(handled, std::ranges::find(built, integration) != built.end())
            << type;
        EXPECT_EQ(tsk, nullptr) << type;
    }
}
}
