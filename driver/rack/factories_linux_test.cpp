// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/labjack/names.h"
#include "driver/rack/rack.h"

namespace driver::rack {
/// @brief it should leave LabJack out of the built integrations and delete the LabJack
/// scanner that a Driver built with LabJack left on the rack.
TEST(RackFactoriesTest, DeletesLegacyLabJackScanner) {
    const auto integrations = built_integrations();
    EXPECT_EQ(
        std::ranges::find(integrations, labjack::INTEGRATION_NAME),
        integrations.end()
    );
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto scanner = synnax::task::Task{
        .name = "LabJack Scanner",
        .type = labjack::SCAN_TASK_TYPE,
        .internal = true,
    };
    ASSERT_NIL(rack.tasks.create(scanner));
    const Config cfg;
    const auto factory = cfg.new_factory(std::make_shared<x::thread::rt::Manager>());
    const auto ctx = std::make_shared<task::MockContext>(client);
    auto tasks = factory->configure_initial_tasks(ctx, rack);
    for (const auto &[_, tsk]: tasks)
        tsk->stop(false);
    ASSERT_OCCURRED_AS_P(rack.tasks.retrieve(scanner.key), x::errors::NOT_FOUND);
}

/// @brief it should leave LabJack out of the rack's integrations when a config names
/// it.
TEST(RackFactoriesTest, RackListsOnlyBuiltIntegrations) {
    Config cfg{
        .connection = test_client_config(),
        .integrations = {labjack::INTEGRATION_NAME, http::INTEGRATION_NAME},
    };
    ASSERT_NIL(cfg.enroll());
    ASSERT_NIL(cfg.load_remote());
    const auto client = synnax::Synnax(test_client_config());
    const auto rack = ASSERT_NIL_P(client.racks.retrieve(cfg.rack.key));
    EXPECT_EQ(rack.integrations, std::vector<std::string>{http::INTEGRATION_NAME});
}

/// @brief it should fail a start of a LabJack task and name the platforms that run it.
TEST(RackFactoriesTest, LabJackTaskNamesThePlatforms) {
    const Config cfg;
    const auto factory = cfg.new_factory(std::make_shared<x::thread::rt::Manager>());
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const auto [tsk, handled] = factory->configure_task(
        ctx,
        synnax::task::Task{.name = "t", .type = labjack::READ_TASK_TYPE},
        "start-cmd"
    );
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 1);
    EXPECT_EQ(statuses[0].message, "Failed to configure task");
    EXPECT_EQ(
        statuses[0].description,
        "LabJack tasks do not run on Linux. Connect the device from a Driver on Windows "
        "or macOS."
    );
    EXPECT_EQ(statuses[0].details.cmd, "start-cmd");
}
}
