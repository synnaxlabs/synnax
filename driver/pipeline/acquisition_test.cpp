// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>

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

    xerrors::Error read(breaker::Breaker &breaker, telem::Frame &fr) override {
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
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

/// @brief MockSource that returns frames with multiple timestamp series
class MultiTimestampSource final : public pipeline::Source {
public:
    std::vector<telem::TimeStamp> timestamps;
    xerrors::Error read_err = xerrors::NIL;

    explicit MultiTimestampSource(
        std::vector<telem::TimeStamp> timestamps,
        const xerrors::Error &read_err = xerrors::NIL
    ):
        timestamps(std::move(timestamps)), read_err(read_err) {}

    xerrors::Error read(breaker::Breaker &breaker, telem::Frame &fr) override {
        if (read_err != xerrors::NIL) return read_err;
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.clear();

        for (size_t i = 0; i < timestamps.size(); i++) {
            fr.emplace(i + 1, telem::Series(timestamps[i]));
        }

        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {}
};

/// @brief MockSource that returns frames with non-timestamp data
class NonTimestampSource final : public pipeline::Source {
public:
    xerrors::Error read(breaker::Breaker &breaker, telem::Frame &fr) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.clear();
        fr.emplace(1, telem::Series(std::vector<float>{1.0f, 2.0f, 3.0f}));
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {}
};

/// @brief MockSource that returns frames with empty timestamp series
class EmptyTimestampSource final : public pipeline::Source {
public:
    xerrors::Error read(breaker::Breaker &breaker, telem::Frame &fr) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.clear();
        fr.emplace(1, telem::Series(telem::TIMESTAMP_T, 0));
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {}
};

/// @brief it should resolve the minimum timestamp when multiple timestamp series exist
TEST(AcquisitionPipeline, testStartResolutionMultipleTimestamps) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    auto ts1 = telem::TimeStamp(1000000000);
    auto ts2 = telem::TimeStamp(2000000000);
    auto ts3 = telem::TimeStamp(1500000000);

    const auto source = std::make_shared<MultiTimestampSource>(
        std::vector{ts2, ts1, ts3}
    );

    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );

    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());

    ASSERT_EQ(mock_factory->config.start, ts1);
}

/// @brief it should fall back to now() when no timestamp series exist
TEST(AcquisitionPipeline, testStartResolutionNoTimestamps) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    auto before = telem::TimeStamp::now();
    const auto source = std::make_shared<NonTimestampSource>();

    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );

    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());

    auto after = telem::TimeStamp::now();

    ASSERT_GE(mock_factory->config.start, before);
    ASSERT_LE(mock_factory->config.start, after);
}

/// @brief a source that returns NOMINAL_SHUTDOWN_ERROR on the first read, used to
/// test that nominal shutdowns don't trigger stopped_with_err.
class NominalShutdownSource final : public pipeline::Source {
public:
    std::atomic<bool> read_called{false};
    xerrors::Error stopped_err = xerrors::NIL;

    xerrors::Error read(breaker::Breaker &breaker, telem::Frame &fr) override {
        read_called.store(true);
        return driver::NOMINAL_SHUTDOWN_ERROR;
    }

    void stopped_with_err(const xerrors::Error &err) override {
        this->stopped_err = err;
    }
};

/// @brief stopped_with_err should not be called when the source returns
/// NOMINAL_SHUTDOWN_ERROR, as this represents a clean shutdown, not a failure.
TEST(AcquisitionPipeline, testNominalShutdownDoesNotCallStoppedWithErr) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);
    const auto source = std::make_shared<NominalShutdownSource>();
    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );
    pipeline.start();
    ASSERT_EVENTUALLY_TRUE(source->read_called.load());
    pipeline.stop();
    ASSERT_EQ(source->stopped_err, xerrors::NIL);
}

/// @brief it should ignore empty timestamp series and fall back to now()
TEST(AcquisitionPipeline, testStartResolutionEmptyTimestamps) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    auto before = telem::TimeStamp::now();
    const auto source = std::make_shared<EmptyTimestampSource>();

    auto pipeline = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );

    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 5);
    ASSERT_TRUE(pipeline.stop());

    auto after = telem::TimeStamp::now();

    ASSERT_GE(mock_factory->config.start, before);
    ASSERT_LE(mock_factory->config.start, after);
}
