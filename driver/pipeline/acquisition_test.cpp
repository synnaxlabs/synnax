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

#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/mock/pipeline.h"

class MockSource final: public pipeline::Source {
public:
    synnax::TimeStamp start_ts;

    explicit MockSource(const synnax::TimeStamp start_ts) : start_ts(start_ts) {
    }

   std::pair<Frame, freighter::Error> read(breaker::Breaker &breaker) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        auto fr = Frame(1);
        fr.emplace(1, synnax::Series(start_ts));
        return {std::move(fr), freighter::Error()};
    }
};


/// @brief it should correctly resolve the start timestamp for the pipeline from the
/// first frame written.
TEST(AcquisitionPipeline, testStartResolution) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<MockWriterFactory>(writes);
    auto start_ts = synnax::TimeStamp::now();
    const auto source = std::make_shared<MockSource>(start_ts);
    synnax::WriterConfig writer_config{.channels = {1}};
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(600));
    pipeline.stop();
    ASSERT_GE(writes->size(), 5);
    ASSERT_LE(writes->size(), 7);
    // ASSERT_EQ(writes->at(0).at<double>(1, 0), 0);
    ASSERT_EQ(mock_factory->config.start.value, start_ts.value);
}

/// @brief it should correctly retry opening the writer when an unreachable error occurs.
TEST(AcquisitionPipeline, testUnreachableRetrySuccess) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<MockWriterFactory>(
        writes,
        std::vector{
            freighter::UNREACHABLE, freighter::UNREACHABLE, freighter::NIL
        });
    const auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .max_retries = 3,
            .scale = 0,
            .base_interval = synnax::MICROSECOND * 10
        }
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(550));
    pipeline.stop();
    ASSERT_GE(writes->size(), 1);
}

/// @brief it should not retry when a non-unreachable error occurs.
TEST(AcquisitionPipeline, testUnreachableUnauthorized) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<MockWriterFactory>(
        writes,
        std::vector{
            freighter::Error(synnax::UNAUTHORIZED_ERROR), freighter::NIL
        }
    );
    const auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .max_retries = 3,
            .scale = 0,
            .base_interval = synnax::MICROSECOND * 10
        }
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(550));
    pipeline.stop();
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should retry opening the writer when write returns false and the
/// error is unreachable.
TEST(AcquisitionPipeline, testWriteRetrySuccess) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<MockWriterFactory>(
        writes,
        std::vector<freighter::Error>{},
        std::vector{freighter::UNREACHABLE},
        std::vector{1}
    );
    const auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .max_retries = 1,
            .scale = 0,
            .base_interval = synnax::MICROSECOND * 10
        }
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(500));
    pipeline.stop();
    ASSERT_EQ(mock_factory->writer_opens, 2);
    ASSERT_GE(writes->size(), 3);
}

/// @brief it should not retry opening the writer when write returns false and the
/// error is not unreachable.
TEST(AcquisitionPipeline, testWriteRetryUnauthorized) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<MockWriterFactory>(
        writes,
        std::vector<freighter::Error>{},
        std::vector{freighter::Error(synnax::UNAUTHORIZED_ERROR)},
        std::vector{0}
    );
    const auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config{
            .max_retries = 1,
            .scale = 0,
            .base_interval = synnax::MICROSECOND * 10
        }
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(500));
    pipeline.stop();
    ASSERT_EQ(mock_factory->writer_opens, 1);
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should not restart the pipeline if it has already been started.
TEST(AcquisitionPipeline, testStartAlreadyStartedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<MockWriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(550));
    pipeline.stop();
    ASSERT_GE(writes->size(), 5);
    ASSERT_LE(writes->size(), 7);
}

/// @brief it should not stop the pipeline if it has already been stopped.
TEST(AcquisitionPipeline, testStopAlreadyStoppedPipeline) {
    auto writes = std::make_shared<std::vector<synnax::Frame> >();
    const auto mock_factory = std::make_shared<MockWriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(synnax::TimeStamp::now());
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    std::this_thread::sleep_for(std::chrono::microseconds(550));
    pipeline.stop();
    pipeline.stop();
    ASSERT_GE(writes->size(), 5);
    ASSERT_LE(writes->size(), 7);
}
