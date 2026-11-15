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
#include "x/cpp/json/json.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/labjack/device/mock.h"
#include "driver/labjack/write_task.h"

namespace driver::labjack {
/// @brief the error LJM returns while the device is unplugged.
const auto UNREACHABLE = x::errors::Error(
    ljm::UNREACHABLE.sub("LJME_DEVICE_NOT_FOUND"),
    ljm::UNREACHABLE_MESSAGE
);

class SinkClaimTest : public ::testing::Test {
protected:
    /// @brief the cancel token given to each start.
    x::breaker::Breaker cancel;
    std::unique_ptr<WriteTaskConfig> cfg;
    std::shared_ptr<device::MockManager> devs;
    synnax::channel::Channel state_idx_ch = synnax::channel::Channel{
        .name = make_unique_channel_name("state_idx_ch"),
        .data_type = x::telem::TIMESTAMP_T,
        .is_index = true
    };
    synnax::channel::Channel state_ch = synnax::channel::Channel{
        .name = make_unique_channel_name("state_ch"),
        .data_type = x::telem::FLOAT64_T
    };
    synnax::channel::Channel cmd_ch = synnax::channel::Channel{
        .name = make_unique_channel_name("cmd_ch"),
        .data_type = x::telem::FLOAT64_T,
        .is_virtual = true
    };

    void parse_config() {
        auto client = std::make_shared<synnax::Synnax>(new_test_client());
        auto rack = ASSERT_NIL_P(client->racks.create("cat"));
        auto dev = synnax::device::Device{
            .key = make_unique_channel_name("labjack"),
            .rack = rack.key,
            .location = "dev1",
            .make = "labjack",
            .model = "T7",
            .name = "my_device",
        };
        ASSERT_NIL(client->devices.create(dev));
        ASSERT_NIL(client->channels.create(state_idx_ch));
        state_ch.index = state_idx_ch.key;
        ASSERT_NIL(client->channels.create(state_ch));
        ASSERT_NIL(client->channels.create(cmd_ch));

        const x::json::json j{
            {"device", dev.key},
            {"state_rate", 10},
            {"data_saving_disabled", false},
            {"channels",
             x::json::json::array(
                 {{{"type", "analog"},
                   {"key", "dac0"},
                   {"port", "DAC0"},
                   {"disabled", false},
                   {"cmd_channel", cmd_ch.key},
                   {"state_channel", state_ch.key}}}
             )}
        };
        auto p = x::json::Parser(j);
        this->cfg = std::make_unique<WriteTaskConfig>(client, p);
        ASSERT_NIL(p.error());
        this->devs = std::make_shared<device::MockManager>();
    }
};

/// @brief it should claim the device on start and release it on stop, so a device
/// that disconnects and returns between runs gets a fresh handle on the next start.
TEST_F(SinkClaimTest, testSinkClaimsDeviceOnStart) {
    parse_config();
    WriteSink sink(devs, std::move(*cfg));
    EXPECT_EQ(devs->acquire_call_count, 0);
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    EXPECT_EQ(devs->acquire_call_count, 1);
    EXPECT_EQ(devs->dev.use_count(), 2);
    ASSERT_NIL(sink.stop());
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    EXPECT_EQ(devs->acquire_call_count, 2);
    ASSERT_NIL(sink.stop());
}

/// @brief it should surface an acquire failure on start and retry the acquisition
/// on the next start.
TEST_F(SinkClaimTest, testSinkAcquireFailureRetriesOnNextStart) {
    parse_config();
    devs->acquire_errors = {UNREACHABLE};
    WriteSink sink(devs, std::move(*cfg));
    ASSERT_OCCURRED_AS(sink.start("start_cmd", this->cancel), ljm::UNREACHABLE);
    EXPECT_EQ(devs->dev.use_count(), 1);
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    EXPECT_EQ(devs->acquire_call_count, 2);
    ASSERT_NIL(sink.stop());
}

/// @brief it should release the claim when the initial state write fails so the
/// next start acquires again.
TEST_F(SinkClaimTest, testSinkReleasesClaimOnFailedInitialWrite) {
    parse_config();
    auto mock_dev = std::make_shared<device::Mock>();
    mock_dev->set_should_fail(true);
    devs = std::make_shared<device::MockManager>(mock_dev);
    WriteSink sink(devs, std::move(*cfg));
    // The local mock_dev and the manager both hold the device.
    EXPECT_EQ(devs->dev.use_count(), 2);
    ASSERT_OCCURRED_AS(
        sink.start("start_cmd", this->cancel),
        x::errors::Error("mock failure")
    );
    EXPECT_EQ(devs->dev.use_count(), 2);
    mock_dev->set_should_fail(false);
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    EXPECT_EQ(devs->acquire_call_count, 2);
    EXPECT_EQ(devs->dev.use_count(), 3);
    ASSERT_NIL(sink.stop());
    EXPECT_EQ(devs->dev.use_count(), 2);
}

/// @brief it should release the device after a temporary write error, then claim it
/// and write the current state of every output before the next command.
TEST_F(SinkClaimTest, testSinkWritesStateAfterReconnect) {
    parse_config();
    const auto mock = std::make_shared<device::Mock>();
    devs = std::make_shared<device::MockManager>(mock);
    WriteSink sink(devs, std::move(*cfg));
    ASSERT_NIL(sink.start("start_cmd", this->cancel));
    auto command = [this](const double value) {
        return x::telem::Frame(this->cmd_ch.key, x::telem::Series(value));
    };
    auto first = command(3);
    ASSERT_NIL(sink.write(first));
    mock->write_errors = {UNREACHABLE};
    auto lost = command(4);
    ASSERT_OCCURRED_AS(sink.write(lost), ljm::UNREACHABLE);
    EXPECT_EQ(mock.use_count(), 2);
    auto next = command(5);
    ASSERT_NIL(sink.write(next));
    EXPECT_EQ(devs->acquire_call_count, 2);
    ASSERT_EQ(mock->name_write_calls.size(), 4);
    EXPECT_EQ(mock->name_write_calls[2].at("DAC0"), 3);
    EXPECT_EQ(mock->name_write_calls[3].at("DAC0"), 5);
    ASSERT_NIL(sink.stop());
}
}
