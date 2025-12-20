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
#include "driver/task/common/read_task.h"

class MockSource final : public common::Source {
    size_t start_count = 0;
    const std::vector<xerrors::Error> start_errs;
    size_t stop_count = 0;
    const std::vector<xerrors::Error> stop_errs;
    pipeline::mock::Source wrapped;

    synnax::WriterConfig writer_config() const override {
        return synnax::WriterConfig();
    }

    std::vector<synnax::Channel> channels() const override { return {}; }

public:
    explicit MockSource(
        const std::shared_ptr<std::vector<telem::Frame>> &reads,
        const std::shared_ptr<std::vector<xerrors::Error>> &read_errors = nullptr,
        const std::vector<xerrors::Error> &start_err = {},
        const std::vector<xerrors::Error> &stop_err = {}
    ):
        start_errs(start_err), stop_errs(stop_err), wrapped(reads, read_errors) {}

    xerrors::Error start() override {
        if (start_count >= start_errs.size()) return xerrors::NIL;
        return start_errs[start_count++];
    }

    xerrors::Error stop() override {
        if (stop_count >= stop_errs.size()) return xerrors::NIL;
        return stop_errs[stop_count++];
    }

    common::ReadResult read(breaker::Breaker &breaker, telem::Frame &data) override {
        common::ReadResult res;
        res.error = this->wrapped.read(breaker, data);
        return res;
    }
};

/// @brief it should start and stop the read task with successful status.
TEST(TestCommonReadTask, testBasicOperation) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<telem::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(telem::Frame(0, std::move(s)));
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
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.key, t.status_key());
    EXPECT_EQ(start_state.details.cmd, start_cmd_key);
    EXPECT_EQ(start_state.details.task, t.key);
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(start_state.message, "Task started successfully");
    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 1);
    read_task.stop("stop_cmd", true);
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 2);
    auto stop_state = ctx->statuses[1];
    EXPECT_EQ(stop_state.key, t.status_key());
    EXPECT_EQ(stop_state.details.cmd, "stop_cmd");
    EXPECT_EQ(stop_state.details.task, t.key);
    EXPECT_EQ(stop_state.variant, status::variant::SUCCESS);
}

/// @brief it should report error status when source fails to start.
TEST(TestCommonReadTask, testErrorOnStart) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<telem::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(telem::Frame(0, std::move(s)));
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
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.key, t.status_key());
    EXPECT_EQ(start_state.details.cmd, start_cmd_key);
    EXPECT_EQ(start_state.details.task, t.key);
    EXPECT_EQ(start_state.variant, status::variant::ERR);
    EXPECT_EQ(start_state.message, "start error");
}

/// @brief it should report error status when source fails to stop.
TEST(TestCommonReadTask, testErrorOnStop) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<telem::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(telem::Frame(0, std::move(s)));
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
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.key, t.status_key());
    EXPECT_EQ(start_state.details.cmd, start_cmd_key);
    EXPECT_EQ(start_state.details.task, t.key);
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);

    const std::string stop_cmd_key = "stop_cmd";
    ASSERT_TRUE(read_task.stop(stop_cmd_key, true));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 2);
    auto stop_state = ctx->statuses[1];
    EXPECT_EQ(stop_state.key, t.status_key());
    EXPECT_EQ(stop_state.details.cmd, stop_cmd_key);
    EXPECT_EQ(stop_state.details.task, t.key);
    EXPECT_EQ(stop_state.variant, status::variant::ERR);
    EXPECT_EQ(stop_state.message, "stop error");
}

/// @brief it should support multiple start-stop cycles.
TEST(TestCommonReadTask, testMultiStartStop) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<telem::Frame>>();

    auto s = telem::Series(telem::TimeStamp::now());
    for (int i = 0; i < 30; i++)
        reads->emplace_back(telem::Frame(i, s.deep_copy()));

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
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state1 = ctx->statuses[0];
    EXPECT_EQ(start_state1.key, t.status_key());
    EXPECT_EQ(start_state1.details.cmd, start_cmd_key1);
    EXPECT_EQ(start_state1.details.task, t.key);
    EXPECT_EQ(start_state1.variant, status::variant::SUCCESS);

    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 1);

    const std::string stop_cmd_key1 = "stop_cmd1";
    ASSERT_TRUE(read_task.stop(stop_cmd_key1, true));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 2);
    auto stop_state1 = ctx->statuses[1];
    EXPECT_EQ(stop_state1.key, t.status_key());
    EXPECT_EQ(stop_state1.details.cmd, stop_cmd_key1);
    EXPECT_EQ(stop_state1.details.task, t.key);
    EXPECT_EQ(stop_state1.variant, status::variant::SUCCESS);

    // Second start-stop cycle
    const std::string start_cmd_key2 = "start_cmd2";
    ASSERT_TRUE(read_task.start(start_cmd_key2));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 3);
    auto start_state2 = ctx->statuses[2];
    EXPECT_EQ(start_state2.key, t.status_key());
    EXPECT_EQ(start_state2.details.cmd, start_cmd_key2);
    EXPECT_EQ(start_state2.details.task, t.key);
    EXPECT_EQ(start_state2.variant, status::variant::SUCCESS);

    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 2);

    const std::string stop_cmd_key2 = "stop_cmd2";
    ASSERT_TRUE(read_task.stop(stop_cmd_key2, true));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 4);
    auto stop_state2 = ctx->statuses[3];
    EXPECT_EQ(stop_state2.key, t.status_key());
    EXPECT_EQ(stop_state2.details.cmd, stop_cmd_key2);
    EXPECT_EQ(stop_state2.details.task, t.key);
    EXPECT_EQ(stop_state2.variant, status::variant::SUCCESS);
}

/// @brief it should report error status when read fails during operation.
TEST(TestCommonReadTask, testReadError) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<telem::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(telem::Frame(0, s.deep_copy()));
    reads->emplace_back(telem::Frame(1, s.deep_copy()));
    auto mock_source = std::make_unique<MockSource>(
        reads,
        std::make_shared<std::vector<xerrors::Error>>(
            std::vector{xerrors::NIL, xerrors::Error("base", "read error")}
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
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.key, t.status_key());
    EXPECT_EQ(start_state.details.cmd, start_cmd_key);
    EXPECT_EQ(start_state.details.task, t.key);
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(start_state.message, "Task started successfully");

    ASSERT_EVENTUALLY_GE(mock_writer_factory->writer_opens, 1);
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 2);
    auto run_err = ctx->statuses[1];
    ASSERT_EQ(run_err.key, t.status_key());
    ASSERT_EQ(run_err.details.task, t.key);
    ASSERT_EQ(run_err.variant, status::variant::ERR);
    ASSERT_EQ(run_err.message, "read error");

    ASSERT_FALSE(read_task.stop("stop_cmd", true));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 3);
    auto stop_state = ctx->statuses[2];
    EXPECT_EQ(stop_state.key, t.status_key());
    EXPECT_EQ(stop_state.details.cmd, "stop_cmd");
    EXPECT_EQ(stop_state.details.task, t.key);
    EXPECT_EQ(stop_state.variant, status::variant::ERR);
    EXPECT_EQ(stop_state.message, "read error");
}

/// @brief it should recover on second start after first start failure.
TEST(TestCommonReadTask, testErrorOnFirstStartupNominalSecondStartup) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<telem::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    reads->emplace_back(telem::Frame(0, std::move(s)));

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
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state1 = ctx->statuses[0];
    EXPECT_EQ(start_state1.key, t.status_key());
    EXPECT_EQ(start_state1.details.cmd, start_cmd_key1);
    EXPECT_EQ(start_state1.details.task, t.key);
    EXPECT_EQ(start_state1.variant, status::variant::ERR);
    EXPECT_EQ(start_state1.message, "first start error");

    // Second start attempt - should succeed
    const std::string start_cmd_key2 = "start_cmd2";
    ASSERT_TRUE(read_task.start(start_cmd_key2));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 2);
    auto start_state2 = ctx->statuses[1];
    EXPECT_EQ(start_state2.key, t.status_key());
    EXPECT_EQ(start_state2.details.cmd, start_cmd_key2);
    EXPECT_EQ(start_state2.details.task, t.key);
    EXPECT_EQ(start_state2.variant, status::variant::SUCCESS);
    EXPECT_EQ(start_state2.message, "Task started successfully");

    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 1);

    // Stop the task
    const std::string stop_cmd_key = "stop_cmd";
    ASSERT_TRUE(read_task.stop(stop_cmd_key, true));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 3);
    auto stop_state = ctx->statuses[2];
    EXPECT_EQ(stop_state.key, t.status_key());
    EXPECT_EQ(stop_state.details.cmd, stop_cmd_key);
    EXPECT_EQ(stop_state.details.task, t.key);
    EXPECT_EQ(stop_state.variant, status::variant::SUCCESS);
}

/// @brief it should recover on second stop after first stop failure.
TEST(TestCommonReadTask, testErrorOnFirstStopNominalSecondStop) {
    auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<telem::Frame>>();
    auto s = telem::Series(telem::TimeStamp::now());
    // Give the pipeline essentially infinite reads.
    for (int i = 0; i < 30; i++)
        reads->emplace_back(telem::Frame(0, s.deep_copy()));

    // Create a source that fails on first stop but succeeds on second stop
    auto mock_source = std::make_unique<MockSource>(
        reads,
        nullptr,
        std::vector<xerrors::Error>{}, // No start errors
        std::vector{xerrors::Error("base", "first stop error"), xerrors::NIL}
        // First stop fails, second succeeds
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
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.key, t.status_key());
    EXPECT_EQ(start_state.details.cmd, start_cmd_key);
    EXPECT_EQ(start_state.details.task, t.key);
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(start_state.message, "Task started successfully");

    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 1);

    // First stop attempt - should report error but return true
    const std::string stop_cmd_key1 = "stop_cmd1";
    ASSERT_TRUE(read_task.stop(stop_cmd_key1, true));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 2);
    auto stop_state1 = ctx->statuses[1];
    EXPECT_EQ(stop_state1.key, t.status_key());
    EXPECT_EQ(stop_state1.details.cmd, stop_cmd_key1);
    EXPECT_EQ(stop_state1.details.task, t.key);
    EXPECT_EQ(stop_state1.variant, status::variant::ERR);
    EXPECT_EQ(stop_state1.message, "first stop error");

    // Start the task again
    const std::string start_cmd_key2 = "start_cmd2";
    ASSERT_TRUE(read_task.start(start_cmd_key2));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 3);
    auto start_state2 = ctx->statuses[2];
    EXPECT_EQ(start_state2.key, t.status_key());
    EXPECT_EQ(start_state2.details.cmd, start_cmd_key2);
    EXPECT_EQ(start_state2.details.task, t.key);
    EXPECT_EQ(start_state2.variant, status::variant::SUCCESS);

    ASSERT_EVENTUALLY_EQ(mock_writer_factory->writer_opens, 2);

    // Second stop attempt - should succeed
    const std::string stop_cmd_key2 = "stop_cmd2";
    ASSERT_TRUE(read_task.stop(stop_cmd_key2, true));
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 4);
    auto stop_state2 = ctx->statuses[3];
    EXPECT_EQ(stop_state2.key, t.status_key());
    EXPECT_EQ(stop_state2.details.cmd, stop_cmd_key2);
    EXPECT_EQ(stop_state2.details.task, t.key);
    EXPECT_EQ(stop_state2.variant, status::variant::SUCCESS);
}

/// @brief it should report warning status on temporary hardware error and recover.
TEST(TestCommonReadTask, testTemporaryErrorWarning) {
    const auto mock_writer_factory = std::make_shared<pipeline::mock::WriterFactory>();
    synnax::Task t;
    t.key = 12345;
    const auto ctx = std::make_shared<task::MockContext>(nullptr);
    auto reads = std::make_shared<std::vector<telem::Frame>>();
    const auto s = telem::Series(telem::TimeStamp::now());
    for (int i = 0; i < 30; i++)
        reads->emplace_back(telem::Frame(i, s.deep_copy()));
    auto mock_source = std::make_unique<MockSource>(
        reads,
        std::make_shared<std::vector<xerrors::Error>>(
            std::vector{xerrors::NIL, driver::TEMPORARY_HARDWARE_ERROR, xerrors::NIL}
        ),
        std::vector{xerrors::NIL}
    );
    auto breaker_config = breaker::default_config("cat");
    breaker_config.base_interval = 10 * telem::MILLISECOND;
    common::ReadTask
        read_task(t, ctx, breaker_config, std::move(mock_source), mock_writer_factory);
    read_task.start("start_cmd");
    ASSERT_EVENTUALLY_EQ(ctx->statuses.size(), 1);
    auto start_state = ctx->statuses[0];
    EXPECT_EQ(start_state.key, t.status_key());
    EXPECT_EQ(start_state.details.cmd, "start_cmd");
    EXPECT_EQ(start_state.variant, status::variant::SUCCESS);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 2);
    auto warning_state = ctx->statuses[1];
    EXPECT_EQ(warning_state.key, t.status_key());
    EXPECT_EQ(warning_state.details.cmd, "start_cmd");
    EXPECT_EQ(warning_state.variant, status::variant::WARNING);
    EXPECT_EQ(warning_state.message, driver::TEMPORARY_HARDWARE_ERROR.message());

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 3);
    auto recovered_state = ctx->statuses[2];
    EXPECT_EQ(recovered_state.key, t.status_key());
    EXPECT_EQ(recovered_state.details.cmd, "start_cmd");
    EXPECT_EQ(recovered_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(recovered_state.message, "Task running");

    read_task.stop("stop_cmd", true);

    ASSERT_EVENTUALLY_GE(ctx->statuses.size(), 4);
    auto stop_state = ctx->statuses[3];
    EXPECT_EQ(stop_state.key, t.status_key());
    EXPECT_EQ(stop_state.details.cmd, "stop_cmd");
    EXPECT_EQ(stop_state.variant, status::variant::SUCCESS);
    EXPECT_EQ(stop_state.message, "Task stopped successfully");
}

/// @brief it should parse valid base read task configuration.
TEST(BaseReadTaskConfigTest, testValidConfig) {
    const json j{{"data_saving", true}, {"sample_rate", 100.0}, {"stream_rate", 50.0}};

    auto p = xjson::Parser(j);
    const auto cfg = common::BaseReadTaskConfig(p);
    ASSERT_FALSE(p.error()) << p.error();
    EXPECT_TRUE(cfg.data_saving);
    EXPECT_EQ(cfg.sample_rate, telem::Rate(100.0));
    EXPECT_EQ(cfg.stream_rate, telem::Rate(50.0));
}

/// @brief it should default data_saving to true when not specified.
TEST(BaseReadTaskConfigTest, testDefaultDataSaving) {
    const json j{{"sample_rate", 100.0}, {"stream_rate", 50.0}};

    auto p = xjson::Parser(j);
    const auto cfg = common::BaseReadTaskConfig(p);
    ASSERT_FALSE(p.error()) << p.error();
    EXPECT_TRUE(cfg.data_saving);
    EXPECT_EQ(cfg.sample_rate, telem::Rate(100.0));
    EXPECT_EQ(cfg.stream_rate, telem::Rate(50.0));
}

/// @brief it should accept equal sample and stream rates.
TEST(BaseReadTaskConfigTest, testEqualRates) {
    const json j{{"sample_rate", 100.0}, {"stream_rate", 100.0}};

    auto p = xjson::Parser(j);
    const auto cfg = common::BaseReadTaskConfig(p);
    ASSERT_NIL(p.error());
    EXPECT_EQ(cfg.sample_rate, telem::Rate(100.0));
    EXPECT_EQ(cfg.stream_rate, telem::Rate(100.0));
}

/// @brief it should return validation error when sample_rate is missing.
TEST(BaseReadTaskConfigTest, testMissingSampleRate) {
    const json j{{"stream_rate", 50.0}};

    auto p = xjson::Parser(j);
    [[maybe_unused]] auto _ = common::BaseReadTaskConfig(p);
    ASSERT_MATCHES(p.error(), xerrors::VALIDATION);
}

/// @brief it should return validation error when stream_rate is missing.
TEST(BaseReadTaskConfigTest, testMissingStreamRate) {
    const json j{{"sample_rate", 100.0}};

    auto p = xjson::Parser(j);
    [[maybe_unused]] auto _ = common::BaseReadTaskConfig(p);
    ASSERT_MATCHES(p.error(), xerrors::VALIDATION);
}

/// @brief it should return validation error for negative sample_rate.
TEST(BaseReadTaskConfigTest, testNegativeSampleRate) {
    const json j{{"sample_rate", -100.0}, {"stream_rate", 50.0}};

    auto p = xjson::Parser(j);
    [[maybe_unused]] auto _ = common::BaseReadTaskConfig(p);
    ASSERT_MATCHES(p.error(), xerrors::VALIDATION);
}

/// @brief it should return validation error for negative stream_rate.
TEST(BaseReadTaskConfigTest, testNegativeStreamRate) {
    const json j{{"sample_rate", 100.0}, {"stream_rate", -50.0}};

    auto p = xjson::Parser(j);
    [[maybe_unused]] auto _ = common::BaseReadTaskConfig(p);
    ASSERT_MATCHES(p.error(), xerrors::VALIDATION);
}

/// @brief it should return validation error when sample_rate is less than stream_rate.
TEST(BaseReadTaskConfigTest, testSampleRateLessThanStreamRate) {
    const json j{{"sample_rate", 25.0}, {"stream_rate", 50.0}};

    auto p = xjson::Parser(j);
    [[maybe_unused]] auto _ = common::BaseReadTaskConfig(p);
    ASSERT_MATCHES(p.error(), xerrors::VALIDATION);
}

/// @brief it should accept missing stream_rate when marked as optional.
TEST(BaseReadTaskConfigTest, testStreamRateOptional) {
    const json j{
        {"sample_rate", 100.0},
        {"data_saving", true}
        // No stream_rate provided
    };

    auto p = xjson::Parser(j);
    const auto cfg = common::BaseReadTaskConfig(p, common::TimingConfig(), false);
    ASSERT_NIL(p.error());
    EXPECT_EQ(cfg.sample_rate, telem::Rate(100.0));
    EXPECT_TRUE(cfg.data_saving);
}

/// @brief it should transfer buffer data to frame for single channel.
TEST(TestCommonReadTask, testTransferBufSingleChannel) {
    const std::vector buf = {1.0, 2.0, 3.0};
    telem::Frame fr;
    fr.reserve(1);
    fr.emplace(1, telem::Series(telem::FLOAT64_T, 3));

    common::transfer_buf(buf, fr, 1, 3);

    EXPECT_EQ(fr.series->at(0).size(), 3);
    EXPECT_EQ(fr.series->at(0).at<double>(0), 1.0);
    EXPECT_EQ(fr.series->at(0).at<double>(1), 2.0);
    EXPECT_EQ(fr.series->at(0).at<double>(2), 3.0);
}

/// @brief it should transfer buffer data to frame for multiple channels.
TEST(TestCommonReadTask, testTransferBufMultipleChannels) {
    const std::vector buf =
        {1.0, 2.0, 3.0, 4.0, 5.0, 6.0}; // 2 channels, 3 samples each
    telem::Frame fr;
    fr.reserve(2);
    fr.emplace(1, telem::Series(telem::FLOAT64_T, 3));
    fr.emplace(2, telem::Series(telem::FLOAT64_T, 3));

    common::transfer_buf(buf, fr, 2, 3);

    EXPECT_EQ(fr.series->at(0).size(), 3);
    EXPECT_EQ(fr.series->at(0).at<double>(0), 1.0);
    EXPECT_EQ(fr.series->at(0).at<double>(1), 2.0);
    EXPECT_EQ(fr.series->at(0).at<double>(2), 3.0);

    EXPECT_EQ(fr.series->at(1).size(), 3);
    EXPECT_EQ(fr.series->at(1).at<double>(0), 4.0);
    EXPECT_EQ(fr.series->at(1).at<double>(1), 5.0);
    EXPECT_EQ(fr.series->at(1).at<double>(2), 6.0);
}

/// @brief it should transfer buffer data to frame for integer type channels.
TEST(TestCommonReadTask, testTransferBufIntegerType) {
    const std::vector buf = {1, 2, 3, 4}; // 2 channels, 2 samples each
    telem::Frame fr;
    fr.reserve(2);
    fr.emplace(1, telem::Series(telem::INT32_T, 2));
    fr.emplace(2, telem::Series(telem::INT32_T, 2));

    common::transfer_buf(buf, fr, 2, 2);

    // Check first channel
    EXPECT_EQ(fr.series->at(0).size(), 2);
    EXPECT_EQ(fr.series->at(0).at<int32_t>(0), 1);
    EXPECT_EQ(fr.series->at(0).at<int32_t>(1), 2);

    // Check second channel
    EXPECT_EQ(fr.series->at(1).size(), 2);
    EXPECT_EQ(fr.series->at(1).at<int32_t>(0), 3);
    EXPECT_EQ(fr.series->at(1).at<int32_t>(1), 4);
}
