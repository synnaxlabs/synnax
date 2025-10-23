// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// GTest
#include "gtest/gtest.h"

#include "x/cpp/xtest/xtest.h"

#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/mock/pipeline.h"

class MockSource final : public pipeline::Source {
public:
    telem::TimeStamp start_ts;
    xerrors::Error stopped_err = xerrors::NIL;
    xerrors::Error read_err = xerrors::NIL;

    explicit MockSource(
        const telem::TimeStamp start_ts,
        const xerrors::Error &read_err = xerrors::NIL
    ):
        start_ts(start_ts), read_err(read_err) {}

    xerrors::Error read(breaker::Breaker &breaker, synnax::Frame &fr) override {
        if (read_err != xerrors::NIL) return read_err;
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.clear();
        fr.emplace(1, telem::Series(start_ts));
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {
        this->stopped_err = err;
    }
};

/// @brief it should correctly resolve the start timestamp for the pipeline from the
/// first frame written.
TEST(AcquisitionPipeline, testStartResolution) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    auto start_ts = telem::TimeStamp::now();
    const auto source = std::make_shared<MockSource>(start_ts);
    synnax::WriterConfig writer_config{.channels = {1}};
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());
    ASSERT_EQ(mock_factory->config.start, start_ts);
}

/// @brief it should correctly retry opening the writer when an unreachable error
/// occurs.
TEST(AcquisitionPipeline, testUnreachableRetrySuccess) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        writes,
        std::vector{freighter::UNREACHABLE, freighter::UNREACHABLE, xerrors::NIL}
    );
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config{
            .name = "pipeline",
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 3,
            .scale = 0,
        }
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    ASSERT_TRUE(pipeline.stop());
}

/// @brief it should not retry when a non-unreachable error occurs.
TEST(AcquisitionPipeline, testUnreachableUnauthorized) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        writes,
        std::vector{xerrors::Error(xerrors::UNAUTHORIZED), xerrors::NIL}
    );
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config{
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 3,
            .scale = 0,
        }
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_EQ(writes->size(), 0);
    ASSERT_TRUE(pipeline.stop());
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should retry opening the writer when write returns false and the
/// error is unreachable.
TEST(AcquisitionPipeline, testWriteRetrySuccess) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        writes,
        std::vector<xerrors::Error>{},
        std::vector{freighter::UNREACHABLE},
        std::vector{1}
    );
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config{
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 1,
            .scale = 0,
        }
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 3);
    ASSERT_TRUE(pipeline.stop());
    ASSERT_EQ(mock_factory->writer_opens, 2);
}

/// @brief it should not retry opening the writer when write returns false and the
/// error is not unreachable.
TEST(AcquisitionPipeline, testWriteRetryUnauthorized) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        writes,
        std::vector<xerrors::Error>{},
        std::vector{xerrors::Error(xerrors::UNAUTHORIZED)},
        std::vector{0}
    );
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config{
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 1,
            .scale = 0,
        }
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    ASSERT_EQ(source->stopped_err, xerrors::UNAUTHORIZED);
    ASSERT_TRUE(pipeline.stop());
}

/// @brief it should not restart the pipeline if it has already been started.
TEST(AcquisitionPipeline, testStartAlreadyStartedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_FALSE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());
}

/// @brief it should not stop the pipeline if it has already been stopped.
TEST(AcquisitionPipeline, testStopAlreadyStoppedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_EQ(writes->size(), 0);
    ASSERT_TRUE(pipeline.stop());
    ASSERT_FALSE(pipeline.stop());
}

/// @brief it should stop the pipeline when the source returns an error on read,
/// and communicate the error back to the source.
TEST(AcquisitionPipeline, testErrorCommunicationOnReadCriticalHardwareError) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    auto critical_error = xerrors::Error(driver::CRITICAL_HARDWARE_ERROR);
    const auto source = std::make_shared<MockSource>(
        telem::TimeStamp::now(),
        critical_error
    );
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );

    pipeline.start();
    ASSERT_EVENTUALLY_EQ(source->stopped_err, critical_error);
    ASSERT_EQ(writes->size(), 0);
    pipeline.stop();
}

/// @brief it should not stop the pipeline if it was never started.
TEST(AcquisitionPipeline, testStopNeverStartedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );
    ASSERT_FALSE(pipeline.stop());
}

/// @brief Regression test: it should throw an error if enable_auto_commit is false
/// when mode is PersistStream. This ensures all drivers set this flag correctly.
TEST(AcquisitionPipeline, testEnableAutoCommitValidation) {
    auto writes = std::make_shared<std::vector<synnax::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());

    // This should throw because enable_auto_commit is false with PersistStream mode
    synnax::WriterConfig bad_config{
        .channels = {1},
        .mode = synnax::WriterMode::PersistStream,
        .enable_auto_commit = false,
    };

    EXPECT_THROW(
        {
            pipeline::Acquisition pipeline(
                mock_factory,
                bad_config,
                source,
                breaker::Config()
            );
        },
        std::runtime_error
    );

    // This should NOT throw because enable_auto_commit is true
    synnax::WriterConfig good_config{
        .channels = {1},
        .mode = synnax::WriterMode::PersistStream,
        .enable_auto_commit = true,
    };

    EXPECT_NO_THROW(
        {
            pipeline::Acquisition pipeline(
                mock_factory,
                good_config,
                source,
                breaker::Config()
            );
        }
    );

    // This should also NOT throw because StreamOnly mode doesn't require auto-commit
    synnax::WriterConfig stream_only_config{
        .channels = {1},
        .mode = synnax::WriterMode::StreamOnly,
        .enable_auto_commit = false,
    };

    EXPECT_NO_THROW(
        {
            pipeline::Acquisition pipeline(
                mock_factory,
                stream_only_config,
                source,
                breaker::Config()
            );
        }
    );
}
