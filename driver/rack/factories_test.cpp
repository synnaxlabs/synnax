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

#include "driver/labjack/names.h"
#include "driver/ni/names.h"
#include "driver/rack/rack.h"
#include "driver/version/version.h"

namespace driver::rack {
/// @brief it should build a factory for each integration in the build whether or not
/// its library loads.
TEST(RackFactoriesTest, BuildsFactoriesForBuiltIntegrations) {
    x::breaker::Breaker cancel;
    const std::vector<std::pair<std::string, std::string>> scanners = {
        {ni::INTEGRATION_NAME, ni::SCAN_TASK_TYPE},
        {labjack::INTEGRATION_NAME, labjack::SCAN_TASK_TYPE},
    };
    const Config cfg{.integrations = {ni::INTEGRATION_NAME, labjack::INTEGRATION_NAME}};
    const auto factory = cfg.new_factory(
        nullptr,
        std::make_shared<x::thread::rt::Manager>()
    );
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const auto built = built_integrations();
    for (const auto &[integration, type]: scanners) {
        if (std::ranges::find(built, integration) == built.end()) continue;
        // An invalid config stops the configure before it loads a library or builds a
        // scanner, which needs a client.
        cancel.start();
        const auto [tsk, handled] = factory->configure_task(
            ctx,
            synnax::task::Task{
                .name = type,
                .type = type,
                .config = {{"rate", "fast"}}
            },
            task::NO_COMMAND,
            cancel
        );
        cancel.stop();
        EXPECT_TRUE(handled) << type;
        EXPECT_EQ(tsk, nullptr) << type;
    }
}

/// @brief it should fail a start of a task whose integration the configuration turns
/// off, and name the integration.
TEST(RackFactoriesTest, DisabledIntegrationNamesItself) {
    x::breaker::Breaker cancel;
    const Config cfg{.integrations = {}};
    const auto factory = cfg.new_factory(
        nullptr,
        std::make_shared<x::thread::rt::Manager>()
    );
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    for (const auto &type: {std::string("modbus_read"), arc::TASK_TYPE}) {
        cancel.start();
        const auto [tsk, handled] = factory->configure_task(
            ctx,
            synnax::task::Task{.name = type, .type = type},
            "start-cmd",
            cancel
        );
        cancel.stop();
        EXPECT_TRUE(handled) << type;
        EXPECT_EQ(tsk, nullptr) << type;
    }
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 2);
    EXPECT_EQ(statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(statuses[0].message, "Failed to configure task");
    EXPECT_EQ(
        statuses[0].description,
        "The configuration of this Driver turns off the 'modbus' integration. Turn it "
        "on in the Driver configuration, then restart the Driver."
    );
    EXPECT_EQ(statuses[0].details.cmd, "start-cmd");
    EXPECT_EQ(
        statuses[1].description,
        "The configuration of this Driver turns off the 'arc' integration. Turn it on "
        "in the Driver configuration, then restart the Driver."
    );
}

/// @brief it should leave a task of a type that no integration knows to the task
/// manager.
TEST(RackFactoriesTest, UnknownTypeIsNotHandled) {
    x::breaker::Breaker cancel;
    const Config cfg{.integrations = {}};
    const auto factory = cfg.new_factory(
        nullptr,
        std::make_shared<x::thread::rt::Manager>()
    );
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    cancel.start();
    const auto [tsk, handled] = factory->configure_task(
        ctx,
        synnax::task::Task{.name = "t", .type = "nimbus_read"},
        "start-cmd",
        cancel
    );
    cancel.stop();
    EXPECT_FALSE(handled);
    EXPECT_EQ(tsk, nullptr);
    EXPECT_TRUE(ctx->snapshot().empty());
}

/// @brief it should refuse a task of every integration and name both versions when the
/// Core runs another minor.
TEST(RackFactoriesTest, IncompatibleCoreRefusesEveryIntegration) {
    x::breaker::Breaker cancel;
    Config cfg{.integrations = built_integrations()};
    cfg.incompatible_core_version = "99.1.0";
    const auto factory = cfg.new_factory(
        nullptr,
        std::make_shared<x::thread::rt::Manager>()
    );
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    const auto built = built_integrations();
    for (const auto &integration: built) {
        const auto type = integration + "_read";
        cancel.start();
        const auto [tsk, handled] = factory->configure_task(
            ctx,
            synnax::task::Task{.name = type, .type = type},
            "start-cmd",
            cancel
        );
        cancel.stop();
        EXPECT_TRUE(handled) << type;
        EXPECT_EQ(tsk, nullptr) << type;
    }
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), built.size());
    for (const auto &stat: statuses) {
        EXPECT_EQ(stat.variant, synnax::status::VARIANT_ERROR);
        EXPECT_EQ(stat.message, "Failed to configure task");
        EXPECT_EQ(
            stat.description,
            "Driver " + std::string(SYNNAX_DRIVER_VERSION) +
                " cannot work with Core 99.1.0. Install the Driver release that "
                "matches the Core, then restart the Driver."
        );
        EXPECT_EQ(stat.details.cmd, "start-cmd");
    }
}

/// @brief it should write a stopped error status for a task that it refuses at boot
/// when the Core runs another minor.
TEST(RackFactoriesTest, IncompatibleCoreStopsTasksAtBoot) {
    x::breaker::Breaker cancel;
    Config cfg{.integrations = built_integrations()};
    cfg.incompatible_core_version = "99.1.0";
    const auto factory = cfg.new_factory(
        nullptr,
        std::make_shared<x::thread::rt::Manager>()
    );
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    cancel.start();
    const auto [tsk, handled] = factory->configure_task(
        ctx,
        synnax::task::Task{.name = "t", .type = http::INTEGRATION_NAME + "_read"},
        task::NO_COMMAND,
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    const auto statuses = ctx->snapshot();
    ASSERT_EQ(statuses.size(), 1);
    EXPECT_EQ(statuses[0].variant, synnax::status::VARIANT_ERROR);
    EXPECT_FALSE(statuses[0].details.running);
    EXPECT_EQ(
        statuses[0].description,
        "Driver " + std::string(SYNNAX_DRIVER_VERSION) +
            " cannot work with Core 99.1.0. Install the Driver release that matches "
            "the Core, then restart the Driver."
    );
}

/// @brief it should only log a boot refusal of a task whose integration the
/// configuration turns off.
TEST(RackFactoriesTest, DisabledIntegrationLogsAtBoot) {
    x::breaker::Breaker cancel;
    const Config cfg{.integrations = {}};
    const auto factory = cfg.new_factory(
        nullptr,
        std::make_shared<x::thread::rt::Manager>()
    );
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    cancel.start();
    const auto [tsk, handled] = factory->configure_task(
        ctx,
        synnax::task::Task{.name = "t", .type = http::INTEGRATION_NAME + "_read"},
        task::NO_COMMAND,
        cancel
    );
    cancel.stop();
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    EXPECT_TRUE(ctx->snapshot().empty());
}
}
