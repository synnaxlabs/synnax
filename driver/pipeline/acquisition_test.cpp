// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/mock/pipeline.h"

namespace driver::pipeline {
class MockSource final : public Source {
public:
    x::telem::TimeStamp start_ts;
    x::errors::Error stopped_err = x::errors::NIL;
    x::errors::Error read_err = x::errors::NIL;

    explicit MockSource(
        const x::telem::TimeStamp start_ts,
        const x::errors::Error &read_err = x::errors::NIL
    ):
        start_ts(start_ts), read_err(read_err) {}

    x::errors::Error read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        if (read_err != x::errors::NIL) return read_err;
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.clear();
        fr.emplace(1, x::telem::Series(start_ts));
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {
        this->stopped_err = err;
    }
};

/// @brief it should correctly resolve the start timestamp for the pipeline from the
/// first frame written.
TEST(AcquisitionPipeline, testStartResolution) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);
    auto start_ts = x::telem::TimeStamp::now();
    const auto source = std::make_shared<MockSource>(start_ts);
    synnax::framer::WriterConfig writer_config{.channels = {1}};
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());
    ASSERT_EQ(mock_factory->config.start, start_ts);
}

/// @brief it should correctly retry opening the writer when an unreachable error
/// occurs.
TEST(AcquisitionPipeline, testUnreachableRetrySuccess) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(
        writes,
        std::vector{
            freighter::ERR_UNREACHABLE,
            freighter::ERR_UNREACHABLE,
            x::errors::NIL
        }
    );
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config{
            .name = "pipeline",
            .base_interval = x::telem::MICROSECOND * 10,
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
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(
        writes,
        std::vector{x::errors::Error(x::errors::UNAUTHORIZED), x::errors::NIL}
    );
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config{
            .base_interval = x::telem::MICROSECOND * 10,
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
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(
        writes,
        std::vector<x::errors::Error>{},
        std::vector{freighter::ERR_UNREACHABLE},
        std::vector{1}
    );
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config{
            .base_interval = x::telem::MICROSECOND * 10,
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
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(
        writes,
        std::vector<x::errors::Error>{},
        std::vector{x::errors::Error(x::errors::UNAUTHORIZED)},
        std::vector{0}
    );
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config{
            .base_interval = x::telem::MICROSECOND * 10,
            .max_retries = 1,
            .scale = 0,
        }
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens, 1);
    ASSERT_EQ(source->stopped_err, x::errors::UNAUTHORIZED);
    ASSERT_TRUE(pipeline.stop());
}

/// @brief it should not restart the pipeline if it has already been started.
TEST(AcquisitionPipeline, testStartAlreadyStartedPipeline) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_FALSE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());
}

/// @brief it should not stop the pipeline if it has already been stopped.
TEST(AcquisitionPipeline, testStopAlreadyStoppedPipeline) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_EQ(writes->size(), 0);
    ASSERT_TRUE(pipeline.stop());
    ASSERT_FALSE(pipeline.stop());
}

/// @brief it should stop the pipeline when the source returns an error on read,
/// and communicate the error back to the source.
TEST(AcquisitionPipeline, testErrorCommunicationOnReadCriticalHardwareError) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);
    auto critical_error = x::errors::Error(CRITICAL_HARDWARE_ERROR);
    const auto source = std::make_shared<MockSource>(
        x::telem::TimeStamp::now(),
        critical_error
    );
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    pipeline.start();
    ASSERT_EVENTUALLY_EQ(source->stopped_err, critical_error);
    ASSERT_EQ(writes->size(), 0);
    pipeline.stop();
}

/// @brief it should not stop the pipeline if it was never started.
TEST(AcquisitionPipeline, testStopNeverStartedPipeline) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_FALSE(pipeline.stop());
}

/// @brief MockSource that returns frames with multiple timestamp series
class MultiTimestampSource final : public Source {
public:
    std::vector<x::telem::TimeStamp> timestamps;
    x::errors::Error read_err = x::errors::NIL;

    explicit MultiTimestampSource(
        std::vector<x::telem::TimeStamp> timestamps,
        const x::errors::Error &read_err = x::errors::NIL
    ):
        timestamps(std::move(timestamps)), read_err(read_err) {}

    x::errors::Error read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        if (read_err != x::errors::NIL) return read_err;
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.clear();

        for (size_t i = 0; i < timestamps.size(); i++) {
            fr.emplace(i + 1, x::telem::Series(timestamps[i]));
        }

        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief MockSource that returns frames with non-timestamp data
class NonTimestampSource final : public Source {
public:
    x::errors::Error read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.clear();
        fr.emplace(1, x::telem::Series(std::vector<float>{1.0f, 2.0f, 3.0f}));
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief MockSource that returns frames with empty timestamp series
class EmptyTimestampSource final : public Source {
public:
    x::errors::Error read(x::breaker::Breaker &breaker, x::telem::Frame &fr) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.clear();
        fr.emplace(1, x::telem::Series(x::telem::TIMESTAMP_T, 0));
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief it should resolve the minimum timestamp when multiple timestamp series exist
TEST(AcquisitionPipeline, testStartResolutionMultipleTimestamps) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    auto ts1 = x::telem::TimeStamp(1000000000);
    auto ts2 = x::telem::TimeStamp(2000000000);
    auto ts3 = x::telem::TimeStamp(1500000000);

    const auto source = std::make_shared<MultiTimestampSource>(
        std::vector{ts2, ts1, ts3}
    );

    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());

    ASSERT_EQ(mock_factory->config.start, ts1);
}

/// @brief it should fall back to now() when no timestamp series exist
TEST(AcquisitionPipeline, testStartResolutionNoTimestamps) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    auto before = x::telem::TimeStamp::now();
    const auto source = std::make_shared<NonTimestampSource>();

    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());

    auto after = x::telem::TimeStamp::now();

    ASSERT_GE(mock_factory->config.start, before);
    ASSERT_LE(mock_factory->config.start, after);
}

/// @brief it should ignore empty timestamp series and fall back to now()
TEST(AcquisitionPipeline, testStartResolutionEmptyTimestamps) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    auto before = x::telem::TimeStamp::now();
    const auto source = std::make_shared<EmptyTimestampSource>();

    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());

    auto after = x::telem::TimeStamp::now();

    ASSERT_GE(mock_factory->config.start, before);
    ASSERT_LE(mock_factory->config.start, after);
}
}
