// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <memory>
#include <utility>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/labjack/device/mock.h"
#include "driver/labjack/labjack.h"

namespace driver::labjack {
/// @brief the description of a configure or scan that fails because LJM does not load.
const std::string LJM_MISSING = "LabJack LJM shared library is not installed. Download "
                                "here: https://support.labjack.com/docs/"
                                "ljm-software-installer-downloads-t4-t7-t8-digit.";

/// @brief a LabJack factory whose LJM library fails to load once, then loads.
class FactoryTest : public ::testing::Test {
protected:
    std::shared_ptr<synnax::Synnax> client;
    std::shared_ptr<task::MockContext> ctx;
    synnax::task::Task task;
    std::atomic<int> ljm_loads = 0;
    Factory factory = Factory(
        [this]() -> std::pair<std::shared_ptr<device::Manager>, x::errors::Error> {
            if (this->ljm_loads++ == 0)
                return {nullptr, driver::errors::missing_lib(ljm::LIBRARY_INFO)};
            return {std::make_shared<device::MockManager>(), x::errors::NIL};
        }
    );

    void SetUp() override {
        this->client = std::make_shared<synnax::Synnax>(new_test_client());
        this->ctx = std::make_shared<task::MockContext>(this->client);
        const auto rack = ASSERT_NIL_P(this->client->racks.create("labjack_factory"));
        auto dev = synnax::device::Device{
            .key = "labjack_factory_device",
            .rack = rack.key,
            .location = "dev1",
            .make = "labjack",
            .model = "T7",
            .name = "my_device",
        };
        ASSERT_NIL(this->client->devices.create(dev));
        const auto ch = ASSERT_NIL_P(this->client->channels.create(
            make_unique_channel_name("labjack_factory"),
            x::telem::FLOAT64_T,
            true
        ));
        this->task = synnax::task::Task{
            .rack = rack.key,
            .name = "my_task",
            .type = READ_TASK_TYPE,
            .config = {
                {"device", dev.key},
                {"sample_rate", 10},
                {"stream_rate", 5},
                {"data_saving_disabled", true},
                {"channels",
                 x::json::json::array({{
                     {"port", "AIN0"},
                     {"disabled", false},
                     {"key", "8hYJO9zt6eS"},
                     {"channel", ch.key},
                     {"type", "analog"},
                     {"range", 5},
                     {"scale", {{"type", "none"}}},
                 }})},
            },
        };
    }
};

/// @brief it should fail to configure a read task while LJM does not load, and
/// configure it once LJM loads.
TEST_F(FactoryTest, ConfigureFailsUntilLJMLoads) {
    const auto [failed, failed_handled] = this->factory.configure_task(
        this->ctx,
        this->task,
        "configure_cmd"
    );
    EXPECT_TRUE(failed_handled);
    EXPECT_EQ(failed, nullptr);
    ASSERT_EQ(this->ctx->statuses.size(), 1);
    const auto &status = this->ctx->statuses[0];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to configure task");
    EXPECT_EQ(status.description, LJM_MISSING);
    EXPECT_EQ(status.details.cmd, "configure_cmd");
    const auto [configured, handled] = this->factory.configure_task(
        this->ctx,
        this->task,
        "configure_cmd"
    );
    EXPECT_TRUE(handled);
    EXPECT_NE(configured, nullptr);
    EXPECT_EQ(this->ljm_loads.load(), 2);
}

/// @brief it should report the missing LJM library at boot for a task that starts on
/// its own.
TEST_F(FactoryTest, AutoStartReportsMissingLJMAtBoot) {
    this->task.config["auto_start"] = true;
    const auto [tsk, handled] = this->factory.configure_task(
        this->ctx,
        this->task,
        task::NO_COMMAND
    );
    EXPECT_TRUE(handled);
    EXPECT_EQ(tsk, nullptr);
    ASSERT_EQ(this->ctx->statuses.size(), 1);
    const auto &status = this->ctx->statuses[0];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to configure task");
    EXPECT_EQ(status.description, LJM_MISSING);
    EXPECT_EQ(status.details.cmd, task::NO_COMMAND);
}

/// @brief it should start the scanner without LJM, warn while LJM does not load, and
/// clear the warning on the first scan after it loads.
TEST_F(FactoryTest, ScannerWarnsUntilLJMLoads) {
    auto [scan_task, handled] = this->factory.configure_task(
        this->ctx,
        synnax::task::Task{
            .name = "LabJack Scanner",
            .type = SCAN_TASK_TYPE,
            .config = {{"rate", 20}},
        },
        task::NO_COMMAND
    );
    ASSERT_TRUE(handled);
    ASSERT_NE(scan_task, nullptr);
    ASSERT_EVENTUALLY_GE_WITH_TIMEOUT(
        this->ctx->snapshot().size(),
        3,
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    scan_task->stop(false);
    const auto statuses = this->ctx->snapshot();
    EXPECT_EQ(statuses[0].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(statuses[0].message, "Started scan task");
    EXPECT_EQ(statuses[1].variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(statuses[1].message, "Failed to scan for devices");
    EXPECT_EQ(statuses[1].description, LJM_MISSING);
    EXPECT_EQ(statuses[2].variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(statuses[2].message, "Scan task running");
}
}
