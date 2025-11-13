// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/pipeline/mock/pipeline.h"
#include "driver/task/common/write_task.h"

class MockSink final : public common::Sink, public pipeline::mock::Sink {
public:
    MockSink(
        const telem::Rate state_rate,
        const std::set<synnax::ChannelKey> &state_indexes,
        const std::vector<synnax::Channel> &state_channels,
        const std::vector<synnax::ChannelKey> &cmd_channels,
        const bool data_saving,
        const std::shared_ptr<std::vector<telem::Frame>> &writes,
        const std::shared_ptr<std::vector<xerrors::Error>> &errors
    ):
        common::Sink(
            state_rate,
            state_indexes,
            state_channels,
            cmd_channels,
            data_saving
        ),
        pipeline::mock::Sink(writes, errors) {}

    xerrors::Error write(const telem::Frame &frame) override {
        auto err = pipeline::mock::Sink::write(frame);
        this->set_state(frame);
        return err;
    }
};

TEST(TestCommonWriteTask, testBasicOperation) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    const auto cmd_reads = std::make_shared<std::vector<telem::Frame>>();
    const auto s = telem::Series(static_cast<uint8_t>(1), telem::UINT8_T);
    cmd_reads->emplace_back(telem::Frame(1, s.deep_copy()));
    auto mock_streamer_factory = pipeline::mock::simple_streamer_factory(
        std::vector<synnax::ChannelKey>{1},
        cmd_reads
    );
    synnax::Channel cmd_channel;
    cmd_channel.key = 1;
    cmd_channel.data_type = telem::UINT8_T;
    cmd_channel.is_virtual = true;

    synnax::Channel state_index;
    state_index.key = 2;
    state_index.data_type = telem::TIMESTAMP_T;
    state_index.index = 2;

    synnax::Channel state;
    state.key = 3;
    state.data_type = telem::UINT8_T;
    state.index = 2;

    auto writes = std::make_shared<std::vector<telem::Frame>>();
    auto errors = std::make_shared<std::vector<xerrors::Error>>();

    auto sink = std::make_unique<MockSink>(
        telem::HERTZ * 10,
        std::set<synnax::ChannelKey>{2},
        std::vector{state},
        std::vector<synnax::ChannelKey>{1},
        false,
        writes,
        errors
    );

    synnax::Task task;
    task.key = 12345;

    auto ctx = std::make_shared<task::MockContext>(nullptr);

    common::WriteTask write_task(
        task,
        ctx,
        breaker::default_config("cat"),
        std::move(sink),
        mock_writer_factory,
        mock_streamer_factory
    );

    auto start_ts = telem::TimeStamp::now();

    std::string cmd_key = "cmd";
    ASSERT_TRUE(write_task.start(cmd_key));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 1);
    auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.key, cmd_key);
    EXPECT_EQ(start_state.details.task, task.key);
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(start_state.message, "Task started successfully");

    ASSERT_EVENTUALLY_GE(mock_writer_factory->writer_opens, 1);
    ASSERT_EVENTUALLY_GE(mock_streamer_factory->streamer_opens, 1);

    ASSERT_EVENTUALLY_GE(mock_writer_factory->writes->size(), 1);
    ASSERT_EVENTUALLY_EQ(writes->size(), 1);
    auto check_state_writes = [&]() -> uint8_t {
        const auto fr = std::move(
            mock_writer_factory->writes->at(mock_writer_factory->writes->size() - 1)
        );
        if (fr.size() < 2) return 0;
        if (fr.length() < 1) return 0;
        if (!fr.contains(3)) return 0;
        return fr.at<uint8_t>(3, 0);
    };
    ASSERT_EVENTUALLY_EQ_F(check_state_writes, 1);

    const std::string stop_cmd_key = "stop_cmd";
    ASSERT_TRUE(write_task.stop(stop_cmd_key, true));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 2);
    auto stop_state = ctx->states[1];
    EXPECT_EQ(stop_state.key, stop_cmd_key);
    EXPECT_EQ(stop_state.details.task, task.key);
    EXPECT_EQ(stop_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(stop_state.message, "Task stopped successfully");

    auto write_fr = std::move(writes->at(0));
    ASSERT_EQ(write_fr.size(), 1);
    ASSERT_EQ(write_fr.length(), 1);
    ASSERT_EQ(write_fr.contains(1), true);
    ASSERT_EQ(write_fr.contains(2), false);
    ASSERT_EQ(write_fr.contains(3), false);
    ASSERT_GE(write_fr.at<uint8_t>(1, 0), 1);

    auto state_fr = std::move(
        mock_writer_factory->writes->at(mock_writer_factory->writes->size() - 1)
    );
    ASSERT_EQ(state_fr.size(), 2);
    ASSERT_EQ(state_fr.length(), 1);
    ASSERT_EQ(state_fr.contains(1), false);
    ASSERT_EQ(state_fr.contains(2), true);
    ASSERT_EQ(state_fr.contains(3), true);
    ASSERT_EQ(state_fr.at<uint8_t>(3, 0), 1);
    ASSERT_GE(state_fr.at<telem::TimeStamp>(2, 0), start_ts);
}
