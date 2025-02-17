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

/// module
#include "x/cpp/xtest/xtest.h"

/// internal
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/mock/pipeline.h"

class MockSource final : public pipeline::Source {
public:
    telem::TimeStamp start_ts;

    explicit MockSource(const telem::TimeStamp start_ts) : start_ts(start_ts) {
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        auto fr = Frame(1);
        fr.emplace(1, telem::Series(start_ts));
        return {std::move(fr), xerrors::Error()};
    }
};

/// @brief it should correctly resolve the start timestamp for the pipeline from the
/// first frame written.
TEST(AcquisitionPipeline, testStartResolution) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    auto start_ts = telem::TimeStamp::now();
    const auto source = std::make_shared<MockSource>(start_ts);
    synnax::WriterConfig writer_config{.channels = {1}};
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    pipeline.stop();
    ASSERT_EQ(mock_factory->config.start.value, start_ts.value);
}

/// @brief it should correctly retry opening the writer when an unreachable error occurs.
TEST(AcquisitionPipeline, testUnreachableRetrySuccess) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        writes,
        std::vector{
            freighter::UNREACHABLE, freighter::UNREACHABLE, xerrors::NIL
        });
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .name = "pipeline",
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 3,
            .scale = 0,
        }
    );
    pipeline.start();
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    pipeline.stop();
}

/// @brief it should not retry when a non-unreachable error occurs.
TEST(AcquisitionPipeline, testUnreachableUnauthorized) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        writes,
        std::vector{
            xerrors::Error(xerrors::UNAUTHORIZED_ERROR), xerrors::NIL
        }
    );
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 3,
            .scale = 0,
        }
    );
    pipeline.start();
    ASSERT_EVENTUALLY_EQ(writes->size(), 0);
    pipeline.stop();
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should retry opening the writer when write returns false and the
/// error is unreachable.
TEST(AcquisitionPipeline, testWriteRetrySuccess) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        writes,
        std::vector<xerrors::Error>{},
        std::vector{freighter::UNREACHABLE},
        std::vector{1}
    );
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 1,
            .scale = 0,
        }
    );
    pipeline.start();
    ASSERT_EVENTUALLY_GE(writes->size(), 3);
    pipeline.stop();
    ASSERT_EQ(mock_factory->writer_opens, 2);
}

/// @brief it should not retry opening the writer when write returns false and the
/// error is not unreachable.
TEST(AcquisitionPipeline, testWriteRetryUnauthorized) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(
        writes,
        std::vector<xerrors::Error>{},
        std::vector{xerrors::Error(xerrors::UNAUTHORIZED_ERROR)},
        std::vector{0}
    );
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .base_interval = telem::MICROSECOND * 10,
            .max_retries = 1,
            .scale = 0,
        }
    );
    pipeline.start();
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    pipeline.stop();
}

/// @brief it should not restart the pipeline if it has already been started.
TEST(AcquisitionPipeline, testStartAlreadyStartedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    pipeline.start();
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    pipeline.stop();
}

/// @brief it should not stop the pipeline if it has already been stopped.
TEST(AcquisitionPipeline, testStopAlreadyStoppedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(telem::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    ASSERT_EVENTUALLY_EQ(writes->size(), 0);
    pipeline.stop();
    pipeline.stop();
}
