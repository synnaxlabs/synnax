// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "driver/rack/rack.h"

namespace driver::rack {
/// @brief it should build no NI or LabJack factory when their libraries do not load,
/// and still build the factory of an integration whose libraries load.
TEST(RackFactoriesTest, testSkipsIntegrationsWithoutLibraries) {
    const Config cfg{
        .integrations =
            {ni::INTEGRATION_NAME, labjack::INTEGRATION_NAME, http::INTEGRATION_NAME},
        .libraries_loaded = [](const std::string &i) {
            return i != ni::INTEGRATION_NAME && i != labjack::INTEGRATION_NAME;
        },
    };
    ASSERT_EQ(cfg.supported_integrations(), std::vector{http::INTEGRATION_NAME});
    const auto factory = cfg.new_factory(std::make_shared<x::thread::rt::Manager>());
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    for (const auto &type: {
             ni::SCAN_TASK_TYPE,
             ni::ANALOG_READ_TASK_TYPE,
             labjack::SCAN_TASK_TYPE,
             labjack::READ_TASK_TYPE,
         }) {
        const auto [tsk, handled] = factory->configure_task(
            ctx,
            synnax::task::Task{.name = type, .type = type},
            task::NO_COMMAND
        );
        EXPECT_FALSE(handled) << type;
        EXPECT_EQ(tsk, nullptr) << type;
    }
    // An invalid config stops the configure before it builds a scanner, which needs a
    // client.
    const auto [scanner, handled] = factory->configure_task(
        ctx,
        synnax::task::Task{
            .name = "http_scan",
            .type = "http_scan",
            .config = {{"rate", "fast"}},
        },
        task::NO_COMMAND
    );
    EXPECT_TRUE(handled);
    EXPECT_EQ(scanner, nullptr);
    EXPECT_TRUE(ctx->statuses.empty());
}
}
