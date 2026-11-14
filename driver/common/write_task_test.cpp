// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <future>
#include <iterator>
#include <thread>

#include "gtest/gtest.h"

#include "x/cpp/sync/list.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/common/testutil.h"
#include "driver/common/write_task.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::common {
class MockSink final : public Sink, public pipeline::testutil::Sink {
public:
    /// @brief number of times start() was called.
    size_t start_count = 0;
    /// @brief number of times stop() was called.
    size_t stop_count = 0;

    MockSink(
        const x::telem::Rate state_rate,
        const std::vector<Output> &outputs,
        const bool data_saving_disabled,
        const std::shared_ptr<x::sync::List<x::telem::Frame>> &writes,
        const std::shared_ptr<std::vector<x::errors::Error>> &errors
    ):
        common::Sink(state_rate, outputs, data_saving_disabled),
        driver::pipeline::testutil::Sink(writes, errors) {}

    x::errors::Error start() override {
        this->start_count++;
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        this->stop_count++;
        return x::errors::NIL;
    }

    x::errors::Error write(x::telem::Frame &frame) override {
        auto err = pipeline::testutil::Sink::write(frame);
        this->set_state(frame);
        return err;
    }
};

/// @brief a sink that skips each command of 0 and reports a warning for it. A command
/// of 2 also warns, but fails with a temporary error.
class SkippingSink final : public Sink {
public:
    /// @brief the frames the sink did not skip.
    std::shared_ptr<x::sync::List<x::telem::Frame>> writes;

    explicit SkippingSink(
        const std::shared_ptr<x::sync::List<x::telem::Frame>> &writes
    ):
        Sink(std::vector<synnax::channel::Key>{1}), writes(writes) {}

    x::errors::Error write(x::telem::Frame &frame) override {
        const auto value = frame.at<uint8_t>(1, 0);
        if (value == 0 || value == 2)
            this->warn({
                .message = "Skipped invalid command",
                .description = "Channel 1: 0 is not valid. Write 1.",
            });
        if (value == 0) return x::errors::NIL;
        if (value == 2) return {errors::TEMPORARY_HARDWARE_ERROR, "device unreachable"};
        this->writes->push_back(frame.deep_copy());
        return x::errors::NIL;
    }
};

/// @brief set_state should use the last sample from a multi-sample series.
TEST(TestCommonWriteTask, testSetStateUsesLastSample) {
    synnax::channel::Channel cmd_channel;
    cmd_channel.key = 1;
    cmd_channel.data_type = x::telem::UINT8_T;
    cmd_channel.is_virtual = true;

    synnax::channel::Channel state_index;
    state_index.key = 2;
    state_index.data_type = x::telem::TIMESTAMP_T;
    state_index.index = 2;

    synnax::channel::Channel state;
    state.key = 3;
    state.data_type = x::telem::UINT8_T;
    state.index = 2;

    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    auto errors = std::make_shared<std::vector<x::errors::Error>>();

    auto sink = std::make_unique<MockSink>(
        x::telem::HERTZ * 10,
        std::vector<Sink::Output>{{1, state}},
        false,
        writes,
        errors
    );

    x::telem::Frame fr(1);
    fr.emplace(1, x::telem::Series(std::vector<uint8_t>{10, 20, 30}));
    ASSERT_NIL(sink->write(fr));

    auto mock_writer_factory = std::make_shared<pipeline::testutil::WriterFactory>();
    auto cmd_reads = std::make_shared<std::vector<x::telem::Frame>>();
    cmd_reads->push_back(std::move(fr));
    auto mock_streamer_factory = pipeline::testutil::simple_streamer_factory(
        std::vector<synnax::channel::Key>{1},
        cmd_reads
    );

    synnax::task::Task task;
    task.key = x::uuid::create();
    auto ctx = std::make_shared<task::MockContext>(nullptr);

    WriteTask write_task(
        task,
        ctx,
        x::breaker::default_config("cat"),
        std::move(sink),
        mock_writer_factory,
        mock_streamer_factory
    );

    std::string cmd_key = "cmd";
    ASSERT_TRUE(write_task.start(cmd_key));
    ASSERT_EVENTUALLY_GE(mock_writer_factory->writes->size(), 1);

    auto check_state = [&]() -> uint8_t {
        const auto &state_fr = mock_writer_factory->writes->at(
            mock_writer_factory->writes->size() - 1
        );
        if (!state_fr.contains(3)) return 0;
        return state_fr.at<uint8_t>(3, 0);
    };
    ASSERT_EVENTUALLY_EQ_F(check_state, 30);

    const std::string stop_cmd_key = "stop";
    ASSERT_TRUE(write_task.stop(stop_cmd_key, true));
}

/// @brief a command that arrives while the state loop is not waiting should still wake
/// the next state read instead of waiting out the full state period.
TEST(TestCommonWriteTask, ReadWakesForCommandSetBeforeWait) {
    synnax::channel::Channel state;
    state.key = 3;
    state.data_type = x::telem::UINT8_T;
    state.index = 2;
    for (const float hz: {1.0F, 10.0F, 100.0F}) {
        const auto rate = x::telem::HERTZ * hz;
        SCOPED_TRACE(std::to_string(hz) + " Hz");
        MockSink sink(
            rate,
            {{1, state}},
            false,
            std::make_shared<x::sync::List<x::telem::Frame>>(),
            std::make_shared<std::vector<x::errors::Error>>()
        );
        x::telem::Frame cmd(1);
        cmd.emplace(1, x::telem::Series(static_cast<uint8_t>(1)));
        ASSERT_NIL(sink.write(cmd));
        x::breaker::Breaker breaker;
        x::telem::Frame fr;
        pipeline::Authorities authorities;
        const auto start = std::chrono::steady_clock::now();
        ASSERT_NIL(sink.read(breaker, fr, authorities));
        const auto elapsed = std::chrono::steady_clock::now() - start;
        EXPECT_EQ(fr.at<uint8_t>(3, 0), 1);
        EXPECT_LT(elapsed, rate.period().chrono() / 2);
    }
}

/// @brief after a read reports a command, the next read should wait for a new command
/// instead of returning at once.
TEST(TestCommonWriteTask, ReadWaitsForNextCommandAfterReport) {
    synnax::channel::Channel state;
    state.key = 3;
    state.data_type = x::telem::UINT8_T;
    state.index = 2;
    const auto rate = x::telem::HERTZ * 10;
    const auto half_period = rate.period().chrono() / 2;
    MockSink sink(
        rate,
        {{1, state}},
        false,
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::make_shared<std::vector<x::errors::Error>>()
    );
    x::breaker::Breaker breaker;
    x::telem::Frame fr;
    pipeline::Authorities authorities;
    x::telem::Frame first(1);
    first.emplace(1, x::telem::Series(static_cast<uint8_t>(1)));
    ASSERT_NIL(sink.write(first));
    ASSERT_NIL(sink.read(breaker, fr, authorities));

    auto start = std::chrono::steady_clock::now();
    ASSERT_NIL(sink.read(breaker, fr, authorities));
    EXPECT_GE(std::chrono::steady_clock::now() - start, half_period);

    x::telem::Frame second(1);
    second.emplace(1, x::telem::Series(static_cast<uint8_t>(0)));
    ASSERT_NIL(sink.write(second));
    start = std::chrono::steady_clock::now();
    ASSERT_NIL(sink.read(breaker, fr, authorities));
    EXPECT_LT(std::chrono::steady_clock::now() - start, half_period);
    EXPECT_EQ(fr.at<uint8_t>(3, 0), 0);
}

/// @brief it should stream every command channel but report state only for the outputs
/// that have a state channel.
TEST(TestCommonWriteTask, OutputWithoutStateChannelReportsNoState) {
    synnax::channel::Channel state;
    state.key = 3;
    state.data_type = x::telem::UINT8_T;
    state.index = 2;
    MockSink sink(
        x::telem::HERTZ * 10,
        {{1, std::nullopt}, {4, state}},
        false,
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::make_shared<std::vector<x::errors::Error>>()
    );
    EXPECT_EQ(
        sink.streamer_config().channels,
        (std::vector<synnax::channel::Key>{1, 4})
    );
    EXPECT_EQ(sink.writer_config().channels, (std::vector<synnax::channel::Key>{3, 2}));
    x::telem::Frame fr(1);
    fr.emplace(1, x::telem::Series(static_cast<uint8_t>(10)));
    ASSERT_NIL(sink.write(fr));
    EXPECT_EQ(std::get<uint8_t>(sink.chan_state.at(3)), 0);
    x::telem::Frame fr_2(1);
    fr_2.emplace(4, x::telem::Series(static_cast<uint8_t>(20)));
    ASSERT_NIL(sink.write(fr_2));
    EXPECT_EQ(std::get<uint8_t>(sink.chan_state.at(3)), 20);
}

/// @brief it should process command frames and write state updates.
TEST(TestCommonWriteTask, testBasicOperation) {
    auto mock_writer_factory = std::make_shared<pipeline::testutil::WriterFactory>();
    const auto cmd_reads = std::make_shared<std::vector<x::telem::Frame>>();
    const auto s = x::telem::Series(static_cast<uint8_t>(1), x::telem::UINT8_T);
    cmd_reads->emplace_back(1, s.deep_copy());
    auto mock_streamer_factory = pipeline::testutil::simple_streamer_factory(
        std::vector<synnax::channel::Key>{1},
        cmd_reads
    );
    synnax::channel::Channel cmd_channel;
    cmd_channel.key = 1;
    cmd_channel.data_type = x::telem::UINT8_T;
    cmd_channel.is_virtual = true;

    synnax::channel::Channel state_index;
    state_index.key = 2;
    state_index.data_type = x::telem::TIMESTAMP_T;
    state_index.index = 2;

    synnax::channel::Channel state;
    state.key = 3;
    state.data_type = x::telem::UINT8_T;
    state.index = 2;

    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    auto errors = std::make_shared<std::vector<x::errors::Error>>();

    auto sink = std::make_unique<MockSink>(
        x::telem::HERTZ * 10,
        std::vector<Sink::Output>{{1, state}},
        false,
        writes,
        errors
    );

    synnax::task::Task task;
    task.key = x::uuid::create();

    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);

    WriteTask write_task(
        task,
        ctx,
        x::breaker::default_config("cat"),
        std::move(sink),
        mock_writer_factory,
        mock_streamer_factory
    );

    auto start_ts = x::telem::TimeStamp::now();

    std::string cmd_key = "cmd";
    ASSERT_TRUE(write_task.start(cmd_key));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(start_state.details.cmd, cmd_key);
    EXPECT_EQ(start_state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(start_state.message, "Started task");

    ASSERT_EVENTUALLY_GE(
        mock_writer_factory->writer_opens.load(std::memory_order_acquire),
        1
    );
    ASSERT_EVENTUALLY_GE(
        mock_streamer_factory->streamer_opens.load(std::memory_order_acquire),
        1
    );

    ASSERT_EVENTUALLY_GE(mock_writer_factory->writes->size(), 1);
    ASSERT_EVENTUALLY_EQ(writes->size(), 1);
    auto check_state_writes = [&]() -> uint8_t {
        const auto fr = mock_writer_factory->writes
                            ->at(mock_writer_factory->writes->size() - 1)
                            .deep_copy();
        if (fr.size() < 2) return 0;
        if (fr.empty()) return 0;
        if (!fr.contains(3)) return 0;
        return fr.at<uint8_t>(3, 0);
    };
    ASSERT_EVENTUALLY_EQ_F(check_state_writes, 1);

    const std::string stop_cmd_key = "stop_cmd";
    ASSERT_TRUE(write_task.stop(stop_cmd_key, true));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 2);
    auto stop_state = ctx->statuses[1];
    EXPECT_EQ(stop_state.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(stop_state.details.cmd, stop_cmd_key);
    EXPECT_EQ(stop_state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(stop_state.message, "Stopped task");

    auto write_fr = writes->at(0).deep_copy();
    ASSERT_EQ(write_fr.size(), 1);
    ASSERT_EQ(write_fr.length(), 1);
    ASSERT_EQ(write_fr.contains(1), true);
    ASSERT_EQ(write_fr.contains(2), false);
    ASSERT_EQ(write_fr.contains(3), false);
    ASSERT_GE(write_fr.at<uint8_t>(1, 0), 1);

    auto state_fr = mock_writer_factory->writes
                        ->at(mock_writer_factory->writes->size() - 1)
                        .deep_copy();
    ASSERT_EQ(state_fr.size(), 2);
    ASSERT_EQ(state_fr.length(), 1);
    ASSERT_EQ(state_fr.contains(1), false);
    ASSERT_EQ(state_fr.contains(2), true);
    ASSERT_EQ(state_fr.contains(3), true);
    ASSERT_EQ(state_fr.at<uint8_t>(3, 0), 1);
    ASSERT_GE(state_fr.at<x::telem::TimeStamp>(2, 0), start_ts);
}

/// @brief a start on a running task should ack the command without reopening
/// the sink.
TEST(TestCommonWriteTask, testStartWhileRunningAcks) {
    auto mock_writer_factory = std::make_shared<pipeline::testutil::WriterFactory>();
    const auto cmd_reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_streamer_factory = pipeline::testutil::simple_streamer_factory(
        std::vector<synnax::channel::Key>{1},
        cmd_reads
    );
    synnax::channel::Channel state;
    state.key = 3;
    state.data_type = x::telem::UINT8_T;
    state.index = 2;
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    auto errors = std::make_shared<std::vector<x::errors::Error>>();
    auto sink = std::make_unique<MockSink>(
        x::telem::HERTZ * 10,
        std::vector<Sink::Output>{{1, state}},
        false,
        writes,
        errors
    );
    synnax::task::Task task;
    task.key = x::uuid::create();
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    WriteTask write_task(
        task,
        ctx,
        x::breaker::default_config("cat"),
        std::move(sink),
        mock_writer_factory,
        mock_streamer_factory
    );
    ASSERT_TRUE(write_task.start("cmd"));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    ASSERT_EVENTUALLY_GE(
        mock_streamer_factory->streamer_opens.load(std::memory_order_acquire),
        1
    );
    ASSERT_FALSE(write_task.start("cmd_2"));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 2);
    auto ack_state = ctx->statuses[1];
    EXPECT_EQ(ack_state.owner, synnax::task::resource_id(task.key));
    EXPECT_EQ(ack_state.details.cmd, "cmd_2");
    EXPECT_EQ(ack_state.variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_EQ(ack_state.message, "Started task");
    EXPECT_EQ(ack_state.details.running, true);
    // The live sink was not closed and reopened.
    EXPECT_EQ(mock_streamer_factory->streamer_opens.load(std::memory_order_acquire), 1);
    ASSERT_TRUE(write_task.stop("stop_cmd", true));
}

namespace {
/// @brief creates a write task with one command channel and no state channels.
std::unique_ptr<WriteTask> create_write_task(
    const std::shared_ptr<driver::task::MockContext> &ctx,
    const std::shared_ptr<pipeline::testutil::StreamerFactory> &streamer_factory,
    MockSink *&sink
) {
    auto s = std::make_unique<MockSink>(
        x::telem::Rate(0),
        std::vector<Sink::Output>{{1, std::nullopt}},
        false,
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::make_shared<std::vector<x::errors::Error>>()
    );
    sink = s.get();
    synnax::task::Task task;
    task.key = x::uuid::create();
    return std::make_unique<WriteTask>(
        task,
        ctx,
        x::breaker::default_config("cat"),
        std::move(s),
        std::make_shared<pipeline::testutil::WriterFactory>(),
        streamer_factory
    );
}
}

/// @brief it should answer a start only after the command streamer opens, so a
/// command written after the answer reaches the sink.
TEST(TestCommonWriteTask, StartAnsweredAfterStreamerOpens) {
    auto streamer_factory = pipeline::testutil::simple_streamer_factory(
        std::vector<synnax::channel::Key>{1},
        std::make_shared<std::vector<x::telem::Frame>>()
    );
    std::promise<void> open;
    streamer_factory->open_gate = open.get_future().share();
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    MockSink *sink = nullptr;
    auto write_task = create_write_task(ctx, streamer_factory, sink);

    ASSERT_TRUE(write_task->start("start_cmd"));
    ASSERT_FALSE(write_task->start("start_cmd_2"));
    EXPECT_TRUE(ctx->snapshot().empty());
    open.set_value();
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    const auto statuses = ctx->snapshot();
    EXPECT_EQ(statuses[0].details.cmd, "start_cmd");
    EXPECT_EQ(statuses[1].details.cmd, "start_cmd_2");
    for (const auto &status: statuses) {
        EXPECT_EQ(status.variant, synnax::status::VARIANT_SUCCESS);
        EXPECT_EQ(status.message, "Started task");
        EXPECT_TRUE(status.details.running);
    }
    ASSERT_TRUE(write_task->stop("stop_cmd", true));
}

/// @brief it should report a failed start and release the sink when the command
/// streamer fails to open.
TEST(TestCommonWriteTask, StreamerOpenErrorFailsStart) {
    auto streamer_factory = std::make_shared<pipeline::testutil::StreamerFactory>(
        std::vector{x::errors::UNKNOWN},
        std::make_shared<std::vector<pipeline::testutil::StreamerConfig>>()
    );
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    MockSink *sink = nullptr;
    auto write_task = create_write_task(ctx, streamer_factory, sink);

    write_task->start("start_cmd");
    // The failed start sends its reply, then a stop status.
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 2);
    const auto status = ctx->snapshot()[0];
    EXPECT_EQ(status.details.cmd, "start_cmd");
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to start task");
    EXPECT_FALSE(status.details.running);
    EXPECT_EQ(sink->stop_count, 1);
    write_task->stop("stop_cmd", true);
    EXPECT_EQ(sink->stop_count, 1);
}

/// @brief it should restart a task whose command pipeline failed, whether or not the
/// failed run has stopped yet.
TEST(TestCommonWriteTask, StartAfterFailureRestarts) {
    auto streamer_factory = pipeline::testutil::simple_streamer_factory(
        std::vector<synnax::channel::Key>{1},
        std::make_shared<std::vector<x::telem::Frame>>()
    );
    streamer_factory->open_errors = {x::errors::UNKNOWN};
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    MockSink *sink = nullptr;
    auto write_task = create_write_task(ctx, streamer_factory, sink);

    ASSERT_TRUE(write_task->start("start_cmd"));
    ASSERT_EVENTUALLY_GE(ctx->snapshot().size(), 1);
    EXPECT_EQ(ctx->snapshot()[0].message, "Failed to start task");
    ASSERT_TRUE(write_task->start("start_cmd_2"));
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().back().details.cmd, "start_cmd_2");
    const auto status = ctx->snapshot().back();
    EXPECT_EQ(status.message, "Started task");
    EXPECT_TRUE(status.details.running);
    EXPECT_EQ(sink->start_count, 2);
    EXPECT_EQ(sink->stop_count, 1);
    ASSERT_TRUE(write_task->stop("stop_cmd", true));
    EXPECT_EQ(sink->stop_count, 2);
}

/// @brief it should stop the sink before it restarts a task whose command pipeline
/// exited without an error.
TEST(TestCommonWriteTask, StartAfterCleanExitStopsSink) {
    auto streamer_factory = std::make_shared<pipeline::testutil::StreamerFactory>(
        std::vector<x::errors::Error>{},
        std::make_shared<std::vector<pipeline::testutil::StreamerConfig>>(
            std::vector{pipeline::testutil::StreamerConfig{
                std::make_shared<std::vector<x::telem::Frame>>(),
                std::make_shared<std::vector<x::errors::Error>>(
                    std::vector{freighter::STREAM_CLOSED}
                ),
                x::errors::NIL,
            }}
        )
    );
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    MockSink *sink = nullptr;
    auto write_task = create_write_task(ctx, streamer_factory, sink);

    ASSERT_TRUE(write_task->start("start_cmd"));
    // A start while the command pipeline still runs is answered without a restart.
    ASSERT_EVENTUALLY_TRUE(write_task->start("start_cmd_2"));
    EXPECT_EQ(sink->stop_count, 1);
    EXPECT_EQ(sink->start_count, 2);
    ASSERT_TRUE(write_task->stop("stop_cmd", true));
    EXPECT_EQ(sink->stop_count, 2);
}

namespace {
const auto OPEN_ERR = x::errors::Error(x::errors::UNAUTHORIZED, "no control authority");

/// @brief creates a write task with one state channel.
std::unique_ptr<WriteTask> create_state_write_task(
    const std::shared_ptr<driver::task::MockContext> &ctx,
    const std::shared_ptr<pipeline::testutil::WriterFactory> &writer_factory,
    const std::shared_ptr<std::vector<x::telem::Frame>> &cmd_reads,
    const std::shared_ptr<x::sync::List<x::telem::Frame>> &writes,
    const std::shared_ptr<std::vector<x::errors::Error>> &write_errors
) {
    synnax::channel::Channel state;
    state.key = 2;
    state.data_type = x::telem::UINT8_T;
    state.is_virtual = true;
    synnax::task::Task task;
    task.key = x::uuid::create();
    return std::make_unique<WriteTask>(
        task,
        ctx,
        x::breaker::Config{.name = "cat", .base_interval = x::telem::MILLISECOND},
        std::make_unique<MockSink>(
            x::telem::HERTZ * 100,
            std::vector<Sink::Output>{{1, state}},
            false,
            writes,
            write_errors
        ),
        writer_factory,
        pipeline::testutil::simple_streamer_factory(
            std::vector<synnax::channel::Key>{1},
            cmd_reads
        )
    );
}
}

/// @brief it should report a failed start when the state writer fails to open after
/// the command streamer opens.
TEST(TestCommonWriteTask, StateWriterOpenErrorFailsStart) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    std::promise<void> open;
    const auto writer_factory = std::make_shared<pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector{OPEN_ERR}
    );
    writer_factory->open_gate = open.get_future().share();
    auto write_task = create_state_write_task(
        ctx,
        writer_factory,
        std::make_shared<std::vector<x::telem::Frame>>(),
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::make_shared<std::vector<x::errors::Error>>()
    );

    ASSERT_TRUE(write_task->start("start_cmd"));
    ASSERT_EVENTUALLY_EQ(ctx->snapshot().size(), 1);
    EXPECT_EQ(ctx->snapshot()[0].message, "Started task");
    open.set_value();
    ASSERT_EVENTUALLY_GE(ctx->snapshot().size(), 2);
    const auto status = ctx->snapshot()[1];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to start task");
    EXPECT_EQ(status.description, "no control authority");
    write_task->stop("stop_cmd", true);
}

/// @brief it should report a failed command while the state writer is still opening.
TEST(TestCommonWriteTask, CommandErrorWhileStateWriterOpens) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    std::promise<void> open;
    const auto cmd_reads = std::make_shared<std::vector<x::telem::Frame>>();
    cmd_reads->emplace_back(1, x::telem::Series(static_cast<uint8_t>(1)));
    const auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto writer_factory = std::make_shared<pipeline::testutil::WriterFactory>();
    writer_factory->open_gate = open.get_future().share();
    auto write_task = create_state_write_task(
        ctx,
        writer_factory,
        cmd_reads,
        writes,
        std::make_shared<std::vector<x::errors::Error>>(
            std::vector{x::errors::Error("base", "write error")}
        )
    );

    ASSERT_TRUE(write_task->start("start_cmd"));
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    // The failed command stops the task, which waits for the state writer to open.
    open.set_value();
    ASSERT_EVENTUALLY_GE(ctx->snapshot().size(), 2);
    const auto status = ctx->snapshot()[1];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to write commands");
    EXPECT_EQ(status.description, "write error");
    write_task->stop("stop_cmd", true);
}

/// @brief it should report a failed write when the state writer fails to reopen
/// after it opened.
TEST(TestCommonWriteTask, StateWriterReopenErrorFailsWrite) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    // The second state write fails, the writer closes as unreachable, and the reopen
    // fails.
    const auto writer_factory = std::make_shared<pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::vector{x::errors::NIL, OPEN_ERR},
        std::vector{freighter::UNREACHABLE},
        std::vector{1}
    );
    auto write_task = create_state_write_task(
        ctx,
        writer_factory,
        std::make_shared<std::vector<x::telem::Frame>>(),
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::make_shared<std::vector<x::errors::Error>>()
    );

    ASSERT_TRUE(write_task->start("start_cmd"));
    ASSERT_EVENTUALLY_GE(ctx->snapshot().size(), 2);
    const auto status = ctx->snapshot()[1];
    EXPECT_EQ(status.variant, synnax::status::VARIANT_ERROR);
    EXPECT_EQ(status.message, "Failed to write commands");
    EXPECT_EQ(status.description, "no control authority");
    write_task->stop("stop_cmd", true);
}

/// @brief it should stop when both pipelines run out of retries while the Core is
/// unreachable.
TEST(TestCommonWriteTask, CoreOutageStopsBothPipelines) {
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    const std::vector unreachable(10, freighter::UNREACHABLE);
    synnax::channel::Channel state;
    state.key = 2;
    state.data_type = x::telem::UINT8_T;
    state.is_virtual = true;
    synnax::task::Task task;
    task.key = x::uuid::create();
    auto write_task = std::make_unique<WriteTask>(
        task,
        ctx,
        x::breaker::Config{
            .name = "cat",
            .base_interval = x::telem::MILLISECOND,
            .max_retries = 2,
        },
        std::make_unique<MockSink>(
            x::telem::HERTZ * 100,
            std::vector<Sink::Output>{{1, state}},
            false,
            std::make_shared<x::sync::List<x::telem::Frame>>(),
            std::make_shared<std::vector<x::errors::Error>>()
        ),
        std::make_shared<pipeline::testutil::WriterFactory>(
            std::make_shared<x::sync::List<x::telem::Frame>>(),
            unreachable
        ),
        std::make_shared<pipeline::testutil::StreamerFactory>(
            unreachable,
            std::make_shared<std::vector<pipeline::testutil::StreamerConfig>>()
        )
    );

    ASSERT_TRUE(write_task->start("start_cmd"));
    // The failed start sends its reply, then a stop status.
    ASSERT_EVENTUALLY_GE(ctx->snapshot().size(), 2);
    EXPECT_EQ(ctx->snapshot()[0].message, "Failed to start task");
    EXPECT_FALSE(write_task->stop("stop_cmd", true));
    const auto status = ctx->snapshot().back();
    EXPECT_EQ(status.details.cmd, "stop_cmd");
    EXPECT_FALSE(status.details.running);
}

/// @brief a task whose sink has no state channels never starts the state pipeline.
/// Stopping it must still release the sink, and must release it exactly once, so a
/// sink that claims hardware on start does not hold it while the task is stopped.
TEST(TestCommonWriteTask, testStopReleasesSinkWithoutStateChannels) {
    auto mock_writer_factory = std::make_shared<pipeline::testutil::WriterFactory>();
    const auto cmd_reads = std::make_shared<std::vector<x::telem::Frame>>();
    auto mock_streamer_factory = pipeline::testutil::simple_streamer_factory(
        std::vector<synnax::channel::Key>{1},
        cmd_reads
    );
    auto sink = std::make_unique<MockSink>(
        x::telem::Rate(0),
        std::vector<Sink::Output>{{1, std::nullopt}},
        false,
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::make_shared<std::vector<x::errors::Error>>()
    );
    auto *raw_sink = sink.get();
    synnax::task::Task task;
    task.key = x::uuid::create();
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    WriteTask write_task(
        task,
        ctx,
        x::breaker::default_config("cat"),
        std::move(sink),
        mock_writer_factory,
        mock_streamer_factory
    );

    ASSERT_TRUE(write_task.start("start_cmd"));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(raw_sink->start_count, 1);
    EXPECT_EQ(raw_sink->stop_count, 0);

    ASSERT_TRUE(write_task.stop("stop_cmd", true));
    EXPECT_EQ(raw_sink->stop_count, 1);

    ASSERT_FALSE(write_task.stop("stop_cmd_2", true));
    EXPECT_EQ(raw_sink->stop_count, 1);

    ASSERT_TRUE(write_task.start("start_cmd_2"));
    EXPECT_EQ(raw_sink->start_count, 2);
    ASSERT_TRUE(write_task.stop("stop_cmd_3", true));
    EXPECT_EQ(raw_sink->stop_count, 2);
}

/// @brief it should run a sink with no outputs from start to stop with no streamer or
/// writer, and release its hardware exactly once.
TEST(TestCommonWriteTask, testRunsWithoutCommandChannels) {
    auto writer_factory = std::make_shared<pipeline::testutil::WriterFactory>();
    auto streamer_factory = pipeline::testutil::simple_streamer_factory(
        {},
        std::make_shared<std::vector<x::telem::Frame>>()
    );
    auto sink = std::make_unique<MockSink>(
        x::telem::Rate(0),
        std::vector<Sink::Output>{},
        false,
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::make_shared<std::vector<x::errors::Error>>()
    );
    auto *raw_sink = sink.get();
    synnax::task::Task task;
    task.key = x::uuid::create();
    auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    WriteTask write_task(
        task,
        ctx,
        x::breaker::default_config("cat"),
        std::move(sink),
        writer_factory,
        streamer_factory
    );

    ASSERT_TRUE(write_task.start("start_cmd"));
    ASSERT_EQ(ctx->statuses.size(), 1);
    EXPECT_EQ(ctx->statuses[0].message, "Started task");
    EXPECT_TRUE(ctx->statuses[0].details.running);
    EXPECT_EQ(raw_sink->start_count, 1);

    ASSERT_FALSE(write_task.start("start_cmd_2"));
    EXPECT_EQ(raw_sink->start_count, 1);

    ASSERT_TRUE(write_task.stop("stop_cmd", true));
    EXPECT_EQ(ctx->statuses.back().message, "Stopped task");
    EXPECT_EQ(raw_sink->stop_count, 1);
    ASSERT_FALSE(write_task.stop("stop_cmd_2", true));
    EXPECT_EQ(raw_sink->stop_count, 1);

    EXPECT_EQ(streamer_factory->streamer_opens.load(), 0);
    EXPECT_EQ(writer_factory->writer_opens.load(), 0);
}

/// @brief it should warn once without a retry wait for a skipped command, and clear the
/// warning on the next command.
TEST(TestCommonWriteTask, SkippedCommandWarnsWithoutBreaker) {
    const auto reads = std::make_shared<std::vector<x::telem::Frame>>();
    for (const uint8_t v: {0, 0, 1})
        reads->emplace_back(1, x::telem::Series(v, x::telem::UINT8_T));
    const auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    synnax::task::Task task;
    task.key = x::uuid::create();
    const auto ctx = std::make_shared<driver::task::MockContext>(nullptr);
    // A breaker wait would outlast the timeout below.
    WriteTask write_task(
        task,
        ctx,
        x::breaker::Config{.name = "test", .base_interval = 30 * x::telem::SECOND},
        std::make_unique<SkippingSink>(writes),
        std::make_shared<pipeline::testutil::WriterFactory>(),
        pipeline::testutil::simple_streamer_factory({1}, reads)
    );
    ASSERT_TRUE(write_task.start("start"));
    ASSERT_EVENTUALLY_EQ_WITH_TIMEOUT(
        writes->size(),
        1,
        std::chrono::seconds(2),
        std::chrono::milliseconds(5)
    );
    ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(
        [&] {
            const auto statuses = ctx->snapshot();
            return statuses.empty() ? "" : statuses.back().message;
        },
        std::string("Task running"),
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    const auto statuses = ctx->snapshot();
    std::vector<synnax::task::Status> warnings;
    std::ranges::copy_if(statuses, std::back_inserter(warnings), [](const auto &s) {
        return s.variant == synnax::status::VARIANT_WARNING;
    });
    ASSERT_EQ(warnings.size(), 1);
    EXPECT_EQ(warnings[0].message, "Skipped invalid command");
    EXPECT_EQ(warnings[0].description, "Channel 1: 0 is not valid. Write 1.");
    EXPECT_TRUE(warnings[0].details.running);
    EXPECT_EQ(statuses.back().variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_TRUE(statuses.back().details.running);
    ASSERT_TRUE(write_task.stop("stop", true));
}

/// @brief a write task that runs a SkippingSink on the given commands.
struct SkippingWrite {
    std::shared_ptr<x::sync::List<x::telem::Frame>>
        writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    std::shared_ptr<driver::task::MockContext>
        ctx = std::make_shared<driver::task::MockContext>(nullptr);
    std::unique_ptr<WriteTask> task;

    explicit SkippingWrite(const std::vector<uint8_t> &commands) {
        const auto reads = std::make_shared<std::vector<x::telem::Frame>>();
        for (const auto v: commands)
            reads->emplace_back(1, x::telem::Series(v, x::telem::UINT8_T));
        synnax::task::Task t;
        t.key = x::uuid::create();
        this->task = std::make_unique<WriteTask>(
            t,
            this->ctx,
            x::breaker::Config{.name = "test", .base_interval = x::telem::MILLISECOND},
            std::make_unique<SkippingSink>(this->writes),
            std::make_shared<pipeline::testutil::WriterFactory>(),
            pipeline::testutil::simple_streamer_factory({1}, reads)
        );
    }
};

/// @brief it should report a temporary error instead of a warning from the same write.
TEST(TestCommonWriteTask, TemporaryErrorTakesPrecedenceOverWarning) {
    SkippingWrite w({2});
    ASSERT_TRUE(w.task->start("start"));
    ASSERT_EVENTUALLY_EQ_F(
        [&] {
            const auto statuses = w.ctx->snapshot();
            return statuses.empty() ? "" : statuses.back().variant;
        },
        synnax::status::VARIANT_WARNING
    );
    const auto status = w.ctx->snapshot().back();
    EXPECT_EQ(status.message, "Failed to write commands");
    EXPECT_EQ(status.description, "device unreachable");
    ASSERT_TRUE(w.task->stop("stop", true));
}

/// @brief it should log a repeated warning once, and log it again after a clean write.
TEST(TestCommonWriteTask, RepeatedWarningLogsOnce) {
    WarningSink log;
    SkippingWrite w({0, 0, 1, 0});
    ASSERT_TRUE(w.task->start("start"));
    ASSERT_EVENTUALLY_EQ(log.count("0 is not valid"), 2);
    EXPECT_EQ(w.writes->size(), 1);
    ASSERT_TRUE(w.task->stop("stop", true));
}

/// @brief a sink whose hardware a spec loses and gets back.
class HoldingSink final : public Sink {
    std::mutex mu;
    Warning held;

public:
    /// @brief the frames the sink wrote.
    std::shared_ptr<x::sync::List<x::telem::Frame>>
        writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    std::atomic<bool> running{false};
    /// @brief the calls of held_warning().
    std::atomic<size_t> reads{0};
    /// @brief the calls of held_warning() while the sink was stopped.
    std::atomic<size_t> stopped_reads{0};

    explicit HoldingSink(Warning held, std::vector<synnax::channel::Key> commands):
        Sink(std::move(commands)), held(std::move(held)) {}

    x::errors::Error start() override {
        this->running = true;
        return x::errors::NIL;
    }

    x::errors::Error stop() override {
        this->running = false;
        return x::errors::NIL;
    }

    /// @brief skips a command of 0 with a warning of its own.
    x::errors::Error write(x::telem::Frame &frame) override {
        if (frame.at<uint8_t>(1, 0) == 0) {
            this->warn({.message = "Skipped invalid command"});
            return x::errors::NIL;
        }
        this->writes->push_back(frame.deep_copy());
        return x::errors::NIL;
    }

    Warning held_warning() override {
        ++this->reads;
        if (!this->running) ++this->stopped_reads;
        std::lock_guard lock(this->mu);
        return this->held;
    }

    /// @brief holds warning, as the hardware thread of a sink does.
    void hold(Warning warning) {
        {
            std::lock_guard lock(this->mu);
            this->held = std::move(warning);
        }
        this->warning_changed();
    }
};

/// @brief a write task that runs a HoldingSink on the given commands to channel 1, or
/// with no command channels when commanded is false.
struct HoldingWrite {
    std::shared_ptr<driver::task::MockContext>
        ctx = std::make_shared<driver::task::MockContext>(nullptr);
    HoldingSink *sink;
    std::unique_ptr<WriteTask> task;

    explicit HoldingWrite(
        const std::vector<uint8_t> &commands,
        Warning held = {},
        const bool commanded = true
    ) {
        const auto reads = std::make_shared<std::vector<x::telem::Frame>>();
        for (const auto v: commands)
            reads->emplace_back(1, x::telem::Series(v, x::telem::UINT8_T));
        auto s = std::make_unique<HoldingSink>(
            std::move(held),
            commanded ? std::vector<synnax::channel::Key>{1}
                      : std::vector<synnax::channel::Key>{}
        );
        this->sink = s.get();
        synnax::task::Task t;
        t.key = x::uuid::create();
        this->task = std::make_unique<WriteTask>(
            t,
            this->ctx,
            x::breaker::Config{.name = "test", .base_interval = x::telem::MILLISECOND},
            std::move(s),
            std::make_shared<pipeline::testutil::WriterFactory>(),
            pipeline::testutil::simple_streamer_factory({1}, reads)
        );
    }

    /// @brief returns the message of the latest status.
    [[nodiscard]] std::string message() const {
        const auto statuses = this->ctx->snapshot();
        return statuses.empty() ? "" : statuses.back().message;
    }

    /// @brief returns the description of the latest status.
    [[nodiscard]] std::string description() const {
        const auto statuses = this->ctx->snapshot();
        return statuses.empty() ? "" : statuses.back().description;
    }
};

/// @brief it should show a held warning as soon as the sink reports it, without a
/// write, and clear it once the sink reports that it ended.
TEST(TestCommonWriteTask, HeldWarningShowsWithoutAWrite) {
    HoldingWrite w({});
    ASSERT_TRUE(w.task->start("start"));
    ASSERT_EVENTUALLY_EQ_F([&] { return w.message(); }, std::string("Started task"));
    w.sink->hold(
        {.message = "Lost device", .description = "Device 1 is not answering."}
    );
    ASSERT_EVENTUALLY_EQ_F([&] { return w.message(); }, std::string("Lost device"));
    const auto warning = w.ctx->snapshot().back();
    EXPECT_EQ(warning.variant, synnax::status::VARIANT_WARNING);
    EXPECT_EQ(warning.description, "Device 1 is not answering.");
    EXPECT_TRUE(warning.details.running);
    w.sink->hold({});
    // The clear waits for the status window that the warning opened.
    ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(
        [&] { return w.message(); },
        std::string("Task running"),
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(w.ctx->snapshot().back().variant, synnax::status::VARIANT_SUCCESS);
    EXPECT_TRUE(w.sink->writes->empty());
    ASSERT_TRUE(w.task->stop("stop", true));
}

/// @brief it should show a warning that the sink held before the task started.
TEST(TestCommonWriteTask, HeldWarningShowsWhenTheTaskStarts) {
    HoldingWrite w({}, {.message = "Lost device"});
    ASSERT_TRUE(w.task->start("start"));
    ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(
        [&] { return w.message(); },
        std::string("Lost device"),
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    ASSERT_TRUE(w.task->stop("stop", true));
}

/// @brief it should show the held warning of a sink with no command channels when the
/// task starts.
TEST(TestCommonWriteTask, HeldWarningShowsWhenATaskWithNoCommandsStarts) {
    HoldingWrite w({}, {.message = "Lost device"}, false);
    ASSERT_TRUE(w.task->start("start"));
    ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(
        [&] { return w.message(); },
        std::string("Lost device"),
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    ASSERT_TRUE(w.task->stop("stop", true));
}

/// @brief it should show the held warning again after a write that reports no warning
/// of its own.
TEST(TestCommonWriteTask, WriteShowsTheHeldWarningAfterItsOwn) {
    WarningSink log;
    HoldingWrite w({0, 1}, {.message = "Lost device"});
    ASSERT_TRUE(w.task->start("start"));
    ASSERT_EVENTUALLY_EQ(w.sink->writes->size(), 1);
    // Each change of the shown warning logs it: at the start, then after each write.
    ASSERT_EVENTUALLY_EQ(log.count("Lost device"), 2);
    EXPECT_EQ(log.count("Skipped invalid command"), 1);
    ASSERT_TRUE(w.task->stop("stop", true));
}

/// @brief it should keep the warning of the latest write when a held warning ends.
TEST(TestCommonWriteTask, EndedHeldWarningKeepsTheWarningOfAWrite) {
    WarningSink log;
    HoldingWrite w({0});
    ASSERT_TRUE(w.task->start("start"));
    ASSERT_EVENTUALLY_EQ(log.count("Skipped invalid command"), 1);
    const size_t reads = w.sink->reads;
    w.sink->hold({});
    ASSERT_EVENTUALLY_GE(w.sink->reads.load(), reads + 1);
    // Waits out the status window, so that a clear would reach the context.
    std::this_thread::sleep_for(std::chrono::milliseconds(1500));
    EXPECT_EQ(w.message(), "Skipped invalid command");
    ASSERT_TRUE(w.task->stop("stop", true));
}

/// @brief it should not read the held warning of a stopped sink.
TEST(TestCommonWriteTask, StoppedSinkGivesNoHeldWarning) {
    HoldingWrite w({});
    ASSERT_TRUE(w.task->start("start"));
    ASSERT_TRUE(w.task->stop("stop", true));
    w.sink->hold(
        {.message = "Lost device", .description = "Device 1 is not answering."}
    );
    // Gives the notifier time to see the stopped sink before the next start.
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    ASSERT_TRUE(w.task->start("start"));
    w.sink->hold(
        {.message = "Lost device", .description = "Device 2 is not answering."}
    );
    ASSERT_EVENTUALLY_EQ_F_WITH_TIMEOUT(
        [&] { return w.description(); },
        std::string("Device 2 is not answering."),
        std::chrono::seconds(5),
        std::chrono::milliseconds(10)
    );
    EXPECT_EQ(w.sink->stopped_reads, 0);
    ASSERT_TRUE(w.task->stop("stop", true));
}

/// @brief it should parse a device key from the config.
TEST(BaseWriteTaskConfigTest, testParse) {
    const x::json::json j{{"device", "abc123"}, {"auto_start", true}};

    auto p = x::json::Parser(j);
    const auto cfg = BaseWriteTaskConfig(p);
    ASSERT_NIL(p.error());
    EXPECT_EQ(cfg.device, "abc123");
    EXPECT_TRUE(cfg.auto_start);
}

/// @brief it should return a validation error when device is missing.
TEST(BaseWriteTaskConfigTest, testMissingDevice) {
    const x::json::json j = x::json::json::object();

    auto p = x::json::Parser(j);
    [[maybe_unused]] auto _ = BaseWriteTaskConfig(p);
    ASSERT_MATCHES(p.error(), x::errors::VALIDATION);
}

/// @brief it should return a validation error when device is empty.
TEST(BaseWriteTaskConfigTest, testEmptyDevice) {
    const x::json::json j{{"device", ""}};

    auto p = x::json::Parser(j);
    [[maybe_unused]] auto _ = BaseWriteTaskConfig(p);
    ASSERT_MATCHES(p.error(), x::errors::VALIDATION);
}
}
