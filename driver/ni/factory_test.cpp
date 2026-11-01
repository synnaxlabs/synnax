// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <utility>

#include "gtest/gtest.h"

#include "client/cpp/testutil/testutil.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/ni/daqmx/mock.h"
#include "driver/ni/ni.h"

namespace driver::ni {
/// @brief the description of a configure that fails because DAQmx does not load.
const std::string DAQMX_MISSING = "National Instruments NI-DAQmx shared library is not "
                                  "installed. Download here: "
                                  "https://www.ni.com/en/support/downloads/drivers/"
                                  "download.ni-daq-mx.html.";

/// @brief an NI factory whose DAQmx library fails to load once, then loads.
class FactoryTest : public ::testing::Test {
protected:
    std::shared_ptr<task::MockContext> ctx;
    synnax::task::Task task;
    int dmx_loads = 0;
    Factory factory = Factory(
        [this]() -> std::pair<std::shared_ptr<daqmx::SugaredAPI>, x::errors::Error> {
            if (this->dmx_loads++ == 0)
                return {nullptr, driver::errors::missing_lib(daqmx::LIBRARY_INFO)};
            return {
                std::make_shared<daqmx::SugaredAPI>(std::make_shared<daqmx::MockAPI>()),
                x::errors::NIL
            };
        },
        []() -> std::pair<std::shared_ptr<syscfg::SugaredAPI>, x::errors::Error> {
            return {nullptr, driver::errors::missing_lib(syscfg::LIBRARY_INFO)};
        }
    );

    void SetUp() override {
        const auto client = std::make_shared<synnax::Synnax>(new_test_client());
        this->ctx = std::make_shared<task::MockContext>(client);
        const auto rack = ASSERT_NIL_P(client->racks.create("ni_factory"));
        auto dev = synnax::device::Device{
            .key = "ni_factory_device",
            .rack = rack.key,
            .location = "dev1",
            .make = "ni",
            .model = "PXI-6255",
            .name = "my_device",
        };
        ASSERT_NIL(client->devices.create(dev));
        const auto ch = ASSERT_NIL_P(client->channels.create(
            make_unique_channel_name("ni_factory"),
            x::telem::FLOAT64_T,
            true
        ));
        this->task = synnax::task::Task{
            .rack = rack.key,
            .name = "my_task",
            .type = ANALOG_READ_TASK_TYPE,
            .config = {
                {"data_saving_disabled", true},
                {"sample_rate", 25},
                {"stream_rate", 25},
                {"channels",
                 x::json::json::array({{
                     {"type", "ai_accel"},
                     {"key", "ks1VnWdrSVA"},
                     {"port", 0},
                     {"disabled", false},
                     {"name", ""},
                     {"channel", ch.key},
                     {"terminal_config", "Cfg_Default"},
                     {"min_val", 0},
                     {"max_val", 1},
                     {"sensitivity", 0},
                     {"current_excit_source", "Internal"},
                     {"current_excit_val", 0},
                     {"custom_scale", {{"type", "none"}}},
                     {"units", "g"},
                     {"sensitivity_units", "mVoltsPerG"},
                     {"device", dev.key},
                 }})},
            },
        };
    }
};

/// @brief it should fail to configure a read task while DAQmx does not load, and
/// configure it once DAQmx loads.
TEST_F(FactoryTest, ConfigureFailsUntilDAQmxLoads) {
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
    EXPECT_EQ(status.description, DAQMX_MISSING);
    EXPECT_EQ(status.details.cmd, "configure_cmd");
    const auto [configured, handled] = this->factory.configure_task(
        this->ctx,
        this->task,
        "configure_cmd"
    );
    EXPECT_TRUE(handled);
    EXPECT_NE(configured, nullptr);
    EXPECT_EQ(this->dmx_loads, 2);
}

/// @brief it should report the missing DAQmx library at boot for a task that starts on
/// its own.
TEST_F(FactoryTest, AutoStartReportsMissingDAQmxAtBoot) {
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
    EXPECT_EQ(status.description, DAQMX_MISSING);
    EXPECT_EQ(status.details.cmd, task::NO_COMMAND);
}
}
