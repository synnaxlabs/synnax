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

#include "driver/rack/rack.h"

namespace driver::rack {
/// @brief it should not report NI as supported when a config names it.
TEST(RackFactoriesTest, NamedNIIsNotSupported) {
    const Config cfg{.integrations = {"ni", http::INTEGRATION_NAME}};
    EXPECT_FALSE(cfg.integration_supported("ni"));
    EXPECT_EQ(cfg.supported_integrations(), std::vector{http::INTEGRATION_NAME});
}

/// @brief it should leave NI out of the default integrations and delete the NI scanner
/// that a Driver built with NI left on the rack.
TEST(RackFactoriesTest, DeletesLegacyNIScanner) {
    const auto integrations = default_integrations();
    EXPECT_EQ(std::ranges::find(integrations, "ni"), integrations.end());
    const auto client = std::make_shared<synnax::Synnax>(new_test_client());
    auto rack = ASSERT_NIL_P(client->racks.create("test_rack"));
    auto scanner = synnax::task::Task{
        .name = "NI Device Scanner",
        .type = "ni_scanner",
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
}
