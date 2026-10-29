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
/// @brief it should build the NI and LabJack factories whether or not their libraries
/// load.
TEST(RackFactoriesTest, BuildsFactoriesWithoutLibraries) {
    const Config cfg{.integrations = {ni::INTEGRATION_NAME, labjack::INTEGRATION_NAME}};
    const auto factory = cfg.new_factory(std::make_shared<x::thread::rt::Manager>());
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    // An invalid config stops the configure before it loads a library or builds a
    // scanner, which needs a client.
    for (const auto &type: {ni::SCAN_TASK_TYPE, labjack::SCAN_TASK_TYPE}) {
        const auto [tsk, handled] = factory->configure_task(
            ctx,
            synnax::task::Task{
                .name = type,
                .type = type,
                .config = {{"rate", "fast"}}
            },
            task::NO_COMMAND
        );
        EXPECT_TRUE(handled) << type;
        EXPECT_EQ(tsk, nullptr) << type;
    }
}
}
