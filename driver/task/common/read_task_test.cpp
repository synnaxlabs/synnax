// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// ReSharper disable CppUseStructuredBinding

/// external
#include "gtest/gtest.h"

/// internal
#include "driver/pipeline/mock/pipeline.h"
#include "driver/task/common/read_task.h"

#include "x/cpp/xtest/xtest.h"

class MockSource final : public pipeline::mock::Source, public common::Source {
    size_t start_count = 0;
    const std::vector<xerrors::Error> start_errs;
    size_t stop_count = 0;
    const std::vector<xerrors::Error> stop_errs;

    synnax::WriterConfig writer_config() const override {
        return synnax::WriterConfig();
    }

public:
    explicit MockSource(
        const std::shared_ptr<std::vector<synnax::Frame>> &reads,
        const std::shared_ptr<std::vector<xerrors::Error>> &read_errors = nullptr,
        const std::vector<xerrors::Error> &start_err = {},
        const std::vector<xerrors::Error> &stop_err = {}
    ): pipeline::mock::Source(reads, read_errors), start_errs(start_err),
       stop_errs(stop_err) {
    }

    xerrors::Error start() override {
        if (start_count >= start_errs.size()) return xerrors::NIL;
        return start_errs[start_count++];
    }

    xerrors::Error stop() override {
        if (stop_count >= stop_errs.size()) return xerrors::NIL;
        return stop_errs[stop_count++];
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        return pipeline::mock::Source::read(breaker);
    }
};

TEST(TestCommonReadTask, testBasicOperation) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(synnax::Frame(0, std::move(s)));
    auto mock_source = std::make_unique<MockSource>(reads);
    common::ReadTask read_task(
        t,
        ctx,
        breaker::default_config("cat"),
        std::move(mock_source),
        mock_writer_factory
    );
    const std::string start_cmd_key = "start_cmd";
    read_task.start(start_cmd_key);
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 1);
    auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.key, start_cmd_key);
    EXPECT_EQ(start_state.task, t.key);
    EXPECT_EQ(start_state.variant, "success");
    EXPECT_EQ(start_state.details["message"], "Task started successfully");
    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 1);
    read_task.stop("stop_cmd", true);
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 2);
    auto stop_state = ctx->states[1];
    EXPECT_EQ(stop_state.key, "stop_cmd");
    EXPECT_EQ(stop_state.task, t.key);
    EXPECT_EQ(stop_state.variant, "success");
}

TEST(TestCommonReadTask, testErrorOnStartup) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(synnax::Frame(0, std::move(s)));
    auto mock_source = std::make_unique<MockSource>(
        reads,
        nullptr,
        std::vector{xerrors::Error("base", "start error")}
    );
    common::ReadTask read_task(
        t,
        ctx,
        breaker::default_config("cat"),
        std::move(mock_source),
        mock_writer_factory
    );
    const std::string start_cmd_key = "start_cmd";
    ASSERT_FALSE(read_task.start(start_cmd_key));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 1);
    auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.key, start_cmd_key);
    EXPECT_EQ(start_state.task, t.key);
    EXPECT_EQ(start_state.variant, "error");
    EXPECT_EQ(start_state.details["message"], "start error");
}

TEST(TestCommonReadTask, testErrorOnStop) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(synnax::Frame(0, std::move(s)));
    auto mock_source = std::make_unique<MockSource>(
        reads,
        nullptr,
        std::vector<xerrors::Error>{},
        std::vector{xerrors::Error("base", "stop error")}
    );
    common::ReadTask read_task(
        t,
        ctx,
        breaker::default_config("cat"),
        std::move(mock_source),
        mock_writer_factory
    );
    const std::string start_cmd_key = "start_cmd";
    ASSERT_TRUE(read_task.start(start_cmd_key));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 1);
    auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.key, start_cmd_key);
    EXPECT_EQ(start_state.task, t.key);
    EXPECT_EQ(start_state.variant, "success");
    
    const std::string stop_cmd_key = "stop_cmd";
    ASSERT_TRUE(read_task.stop(stop_cmd_key, true));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 2);
    auto stop_state = ctx->states[1];
    EXPECT_EQ(stop_state.key, stop_cmd_key);
    EXPECT_EQ(stop_state.task, t.key);
    EXPECT_EQ(stop_state.variant, "error");
    EXPECT_EQ(stop_state.details["message"], "stop error");
}

TEST(TestCommonReadTask, testMultiStartStop) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    
    auto s = telem::Series(telem::TimeStamp::now());
    for (int i = 0; i < 30; i++)
        reads->emplace_back(synnax::Frame(i, s.deep_copy()));

    auto mock_source = std::make_unique<MockSource>(reads);
    common::ReadTask read_task(
        t,
        ctx,
        breaker::default_config("cat"),
        std::move(mock_source),
        mock_writer_factory
    );
    
    // First start-stop cycle
    const std::string start_cmd_key1 = "start_cmd1";
    ASSERT_TRUE(read_task.start(start_cmd_key1));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 1);
    auto start_state1 = ctx->states[0];
    EXPECT_EQ(start_state1.key, start_cmd_key1);
    EXPECT_EQ(start_state1.task, t.key);
    EXPECT_EQ(start_state1.variant, "success");

    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 1);

    const std::string stop_cmd_key1 = "stop_cmd1";
    ASSERT_TRUE(read_task.stop(stop_cmd_key1, true));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 2);
    auto stop_state1 = ctx->states[1];
    EXPECT_EQ(stop_state1.key, stop_cmd_key1);
    EXPECT_EQ(stop_state1.task, t.key);
    EXPECT_EQ(stop_state1.variant, "success");

    // Second start-stop cycle
    const std::string start_cmd_key2 = "start_cmd2";
    ASSERT_TRUE(read_task.start(start_cmd_key2));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 3);
    auto start_state2 = ctx->states[2];
    EXPECT_EQ(start_state2.key, start_cmd_key2);
    EXPECT_EQ(start_state2.task, t.key);
    EXPECT_EQ(start_state2.variant, "success");

    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 2);

    const std::string stop_cmd_key2 = "stop_cmd2";
    ASSERT_TRUE(read_task.stop(stop_cmd_key2, true));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 4);
    auto stop_state2 = ctx->states[3];
    EXPECT_EQ(stop_state2.key, stop_cmd_key2);
    EXPECT_EQ(stop_state2.task, t.key);
    EXPECT_EQ(stop_state2.variant, "success");
    
}

TEST(TestCommonReadTask, testReadError) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(synnax::Frame(0, s.deep_copy()));
    reads->emplace_back(synnax::Frame(1, s.deep_copy()));
    auto mock_source = std::make_unique<MockSource>(
        reads,
        std::make_shared<std::vector<xerrors::Error>>(
            std::vector{
                xerrors::NIL,
                xerrors::Error("base", "read error")
            }
        )
    );

    common::ReadTask read_task(
        t,
        ctx,
        breaker::default_config("cat"),
        std::move(mock_source),
        mock_writer_factory
    );

    const std::string start_cmd_key = "start_cmd";
    ASSERT_TRUE(read_task.start(start_cmd_key));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 1);
    auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.key, start_cmd_key);
    EXPECT_EQ(start_state.task, t.key);
    EXPECT_EQ(start_state.variant, "success");
    EXPECT_EQ(start_state.details["message"], "Task started successfully");

    ASSERT_EVENTUALLY_GE(mock_writer_factory->writer_opens, 1);
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 2);
    auto run_err = ctx->states[1];
    ASSERT_EQ(run_err.key, "");
    ASSERT_EQ(run_err.task, t.key);
    ASSERT_EQ(run_err.variant, "error");
    ASSERT_EQ(run_err.details["message"], "read error");

    ASSERT_FALSE(read_task.stop("stop_cmd", true));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 3);
    auto stop_state = ctx->states[2];
    EXPECT_EQ(stop_state.key, "stop_cmd");
    EXPECT_EQ(stop_state.task, t.key);
    EXPECT_EQ(stop_state.variant, "error");
    EXPECT_EQ(stop_state.details["message"], "read error");
}

TEST(TestCommonReadTask, testErrorOnFirstStartupNominalSecondStartup) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(synnax::Frame(0, std::move(s)));
    
    // Create a source that fails on first start but succeeds on second start
    auto mock_source = std::make_unique<MockSource>(
        reads,
        nullptr,
        std::vector{xerrors::Error("base", "first start error"), xerrors::NIL}
    );
    
    common::ReadTask read_task(
        t,
        ctx,
        breaker::default_config("cat"),
        std::move(mock_source),
        mock_writer_factory
    );
    
    // First start attempt - should fail
    const std::string start_cmd_key1 = "start_cmd1";
    ASSERT_FALSE(read_task.start(start_cmd_key1));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 1);
    auto start_state1 = ctx->states[0];
    EXPECT_EQ(start_state1.key, start_cmd_key1);
    EXPECT_EQ(start_state1.task, t.key);
    EXPECT_EQ(start_state1.variant, "error");
    EXPECT_EQ(start_state1.details["message"], "first start error");
    
    // Second start attempt - should succeed
    const std::string start_cmd_key2 = "start_cmd2";
    ASSERT_TRUE(read_task.start(start_cmd_key2));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 2);
    auto start_state2 = ctx->states[1];
    EXPECT_EQ(start_state2.key, start_cmd_key2);
    EXPECT_EQ(start_state2.task, t.key);
    EXPECT_EQ(start_state2.variant, "success");
    EXPECT_EQ(start_state2.details["message"], "Task started successfully");
    
    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 1);
    
    // Stop the task
    const std::string stop_cmd_key = "stop_cmd";
    ASSERT_TRUE(read_task.stop(stop_cmd_key, true));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 3);
    auto stop_state = ctx->states[2];
    EXPECT_EQ(stop_state.key, stop_cmd_key);
    EXPECT_EQ(stop_state.task, t.key);
    EXPECT_EQ(stop_state.variant, "success");
}

TEST(TestCommonReadTask, testErrorOnFirstStopNominalSecondStop) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<synnax::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    // Give the pipeline essentially infinite reads.
    for (int i = 0; i < 30; i++)
        reads->emplace_back(synnax::Frame(0, s.deep_copy()));
    
    // Create a source that fails on first stop but succeeds on second stop
    auto mock_source = std::make_unique<MockSource>(
        reads,
        nullptr,
        std::vector<xerrors::Error>{},  // No start errors
        std::vector{xerrors::Error("base", "first stop error"), xerrors::NIL}  // First stop fails, second succeeds
    );
    
    common::ReadTask read_task(
        t,
        ctx,
        breaker::default_config("cat"),
        std::move(mock_source),
        mock_writer_factory
    );
    
    // Start the task
    const std::string start_cmd_key = "start_cmd";
    ASSERT_TRUE(read_task.start(start_cmd_key));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 1);
    auto start_state = ctx->states[0];
    EXPECT_EQ(start_state.key, start_cmd_key);
    EXPECT_EQ(start_state.task, t.key);
    EXPECT_EQ(start_state.variant, "success");
    EXPECT_EQ(start_state.details["message"], "Task started successfully");
    
    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 1);
    
    // First stop attempt - should report error but return true
    const std::string stop_cmd_key1 = "stop_cmd1";
    ASSERT_TRUE(read_task.stop(stop_cmd_key1, true));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 2);
    auto stop_state1 = ctx->states[1];
    EXPECT_EQ(stop_state1.key, stop_cmd_key1);
    EXPECT_EQ(stop_state1.task, t.key);
    EXPECT_EQ(stop_state1.variant, "error");
    EXPECT_EQ(stop_state1.details["message"], "first stop error");
    
    // Start the task again
    const std::string start_cmd_key2 = "start_cmd2";
    ASSERT_TRUE(read_task.start(start_cmd_key2));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 3);
    auto start_state2 = ctx->states[2];
    EXPECT_EQ(start_state2.key, start_cmd_key2);
    EXPECT_EQ(start_state2.task, t.key);
    EXPECT_EQ(start_state2.variant, "success");
    
    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 2);
    
    // Second stop attempt - should succeed
    const std::string stop_cmd_key2 = "stop_cmd2";
    ASSERT_TRUE(read_task.stop(stop_cmd_key2, true));
    ASSERT_EVENTUALLY_EQ(ctx->states.size(), 4);
    auto stop_state2 = ctx->states[3];
    EXPECT_EQ(stop_state2.key, stop_cmd_key2);
    EXPECT_EQ(stop_state2.task, t.key);
    EXPECT_EQ(stop_state2.variant, "success");
}