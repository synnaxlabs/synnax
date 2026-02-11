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

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        if (read_err != x::errors::NIL) return read_err;
        std::this_thread::sleep_for(std::chrono::microseconds(100));
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
        std::vector{freighter::UNREACHABLE, freighter::UNREACHABLE, x::errors::NIL}
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
        std::vector{freighter::UNREACHABLE},
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
    ASSERT_EQ(mock_factory->writer_opens.load(std::memory_order_acquire), 2);
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
    ASSERT_EVENTUALLY_GE(mock_factory->writer_opens.load(std::memory_order_acquire), 1);
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
    auto critical_error = x::errors::Error(errors::CRITICAL_HARDWARE_ERROR);
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

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        if (read_err != x::errors::NIL) return read_err;
        std::this_thread::sleep_for(std::chrono::microseconds(100));

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
    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.emplace(1, x::telem::Series(std::vector<float>{1.0f, 2.0f, 3.0f}));
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief MockSource that returns frames with empty timestamp series
class EmptyTimestampSource final : public Source {
public:
    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
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

/// @brief a source that returns NOMINAL_SHUTDOWN_ERROR on the first read, used to
/// test that nominal shutdowns don't trigger stopped_with_err.
class NominalShutdownSource final : public Source {
public:
    std::atomic<bool> read_called{false};
    x::errors::Error stopped_err = x::errors::NIL;

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        read_called.store(true);
        return errors::NOMINAL_SHUTDOWN_ERROR;
    }

    void stopped_with_err(const x::errors::Error &err) override {
        this->stopped_err = err;
    }
};

/// @brief stopped_with_err should not be called when the source returns
/// NOMINAL_SHUTDOWN_ERROR, as this represents a clean shutdown, not a failure.
TEST(AcquisitionPipeline, testNominalShutdownDoesNotCallStoppedWithErr) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);
    const auto source = std::make_shared<NominalShutdownSource>();
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    pipeline.start();
    ASSERT_EVENTUALLY_TRUE(source->read_called.load());
    pipeline.stop();
    ASSERT_EQ(source->stopped_err, x::errors::NIL);
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

class AuthoritySource final : public Source {
    x::telem::TimeStamp start_ts;
    Authorities auth;
    bool sent_auth = false;

public:
    explicit AuthoritySource(const x::telem::TimeStamp start_ts, Authorities auth):
        start_ts(start_ts), auth(std::move(auth)) {}

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.emplace(1, x::telem::Series(this->start_ts));
        if (!this->sent_auth) {
            authorities = this->auth;
            this->sent_auth = true;
        }
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief it should forward authority changes from Source to Writer
TEST(AcquisitionPipeline, testAuthorityForwarding) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    Authorities auth{
        .keys = {1, 2},
        .authorities = {100, 200},
    };
    const auto source = std::make_shared<AuthoritySource>(
        x::telem::TimeStamp::now(),
        auth
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    const auto &change = mock_factory->authority_changes->at(0);
    ASSERT_EQ(change.keys.size(), 2);
    EXPECT_EQ(change.keys[0], 1);
    EXPECT_EQ(change.keys[1], 2);
    ASSERT_EQ(change.authorities.size(), 2);
    EXPECT_EQ(change.authorities[0], 100);
    EXPECT_EQ(change.authorities[1], 200);
}

class MultiAuthoritySource final : public Source {
    x::telem::TimeStamp start_ts;
    std::vector<Authorities> pending;
    std::atomic<size_t> read_count{0};

public:
    explicit MultiAuthoritySource(
        const x::telem::TimeStamp start_ts,
        std::vector<Authorities> pending
    ):
        start_ts(start_ts), pending(std::move(pending)) {}

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.emplace(1, x::telem::Series(this->start_ts));
        const auto idx = this->read_count.fetch_add(1);
        if (idx < this->pending.size()) authorities = this->pending[idx];
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief it should forward multiple sequential authority changes across reads.
TEST(AcquisitionPipeline, testMultipleAuthorityChanges) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    std::vector<Authorities> changes = {
        {.keys = {1}, .authorities = {100}},
        {.keys = {2}, .authorities = {200}},
        {.keys = {3}, .authorities = {50}},
    };
    const auto source = std::make_shared<MultiAuthoritySource>(
        x::telem::TimeStamp::now(),
        changes
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 3);
    ASSERT_TRUE(pipe.stop());

    EXPECT_EQ(mock_factory->authority_changes->at(0).keys[0], 1);
    EXPECT_EQ(mock_factory->authority_changes->at(0).authorities[0], 100);
    EXPECT_EQ(mock_factory->authority_changes->at(1).keys[0], 2);
    EXPECT_EQ(mock_factory->authority_changes->at(1).authorities[0], 200);
    EXPECT_EQ(mock_factory->authority_changes->at(2).keys[0], 3);
    EXPECT_EQ(mock_factory->authority_changes->at(2).authorities[0], 50);
}

/// @brief it should forward global authority changes (empty keys) correctly.
TEST(AcquisitionPipeline, testGlobalAuthorityForwarding) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    Authorities auth{
        .keys = {},
        .authorities = {150},
    };
    const auto source = std::make_shared<AuthoritySource>(
        x::telem::TimeStamp::now(),
        auth
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    const auto &change = mock_factory->authority_changes->at(0);
    EXPECT_TRUE(change.keys.empty());
    ASSERT_EQ(change.authorities.size(), 1);
    EXPECT_EQ(change.authorities[0], 150);
}

/// @brief it should forward authority changes from the very first read that also
/// triggers the writer to open.
TEST(AcquisitionPipeline, testAuthorityOnFirstFrame) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    Authorities auth{
        .keys = {42},
        .authorities = {250},
    };
    const auto source = std::make_shared<AuthoritySource>(
        x::telem::TimeStamp::now(),
        auth
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    const auto &change = mock_factory->authority_changes->at(0);
    ASSERT_EQ(change.keys.size(), 1);
    EXPECT_EQ(change.keys[0], 42);
    EXPECT_EQ(change.authorities[0], 250);
}

class EmptyFrameAuthoritySource final : public Source {
    x::telem::TimeStamp start_ts;
    Authorities auth;
    std::atomic<size_t> read_count{0};

public:
    explicit EmptyFrameAuthoritySource(
        const x::telem::TimeStamp start_ts,
        Authorities auth
    ):
        start_ts(start_ts), auth(std::move(auth)) {}

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        auto idx = this->read_count.fetch_add(1);
        if (idx == 0) {
            fr.emplace(1, x::telem::Series(this->start_ts));
            return x::errors::NIL;
        }
        if (idx == 1) authorities = this->auth;
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief it should forward authority changes even when the frame is empty.
TEST(AcquisitionPipeline, testAuthorityWithEmptyFrame) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    Authorities auth{
        .keys = {10},
        .authorities = {200},
    };
    const auto source = std::make_shared<EmptyFrameAuthoritySource>(
        x::telem::TimeStamp::now(),
        auth
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    const auto &change = mock_factory->authority_changes->at(0);
    ASSERT_EQ(change.keys.size(), 1);
    EXPECT_EQ(change.keys[0], 10);
    EXPECT_EQ(change.authorities[0], 200);
}

class PreWriterAuthoritySource final : public Source {
    x::telem::TimeStamp start_ts;
    std::vector<Authorities> pre_writer_auths;
    std::atomic<size_t> read_count{0};

public:
    explicit PreWriterAuthoritySource(
        const x::telem::TimeStamp start_ts,
        std::vector<Authorities> pre_writer_auths
    ):
        start_ts(start_ts), pre_writer_auths(std::move(pre_writer_auths)) {}

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        auto idx = this->read_count.fetch_add(1);
        if (idx < this->pre_writer_auths.size()) {
            authorities = this->pre_writer_auths[idx];
            return x::errors::NIL;
        }
        fr.emplace(1, x::telem::Series(this->start_ts));
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief it should buffer authority changes that arrive before the writer opens
/// and apply them once the first frame opens the writer.
TEST(AcquisitionPipeline, testAuthorityBufferedBeforeWriter) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    std::vector<Authorities> pre_auths = {
        {.keys = {1}, .authorities = {100}},
    };
    const auto source = std::make_shared<PreWriterAuthoritySource>(
        x::telem::TimeStamp::now(),
        pre_auths
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    const auto &change = mock_factory->authority_changes->at(0);
    ASSERT_EQ(change.keys.size(), 1);
    EXPECT_EQ(change.keys[0], 1);
    EXPECT_EQ(change.authorities[0], 100);
}

/// @brief it should dedupe per-channel authority changes buffered before the writer
/// opens, keeping only the last value for each channel.
TEST(AcquisitionPipeline, testAuthorityBufferDedupes) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    std::vector<Authorities> pre_auths = {
        {.keys = {1, 2}, .authorities = {100, 200}},
        {.keys = {1}, .authorities = {50}},
    };
    const auto source = std::make_shared<PreWriterAuthoritySource>(
        x::telem::TimeStamp::now(),
        pre_auths
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    const auto &change = mock_factory->authority_changes->at(0);
    ASSERT_EQ(change.keys.size(), 2);
    std::map<synnax::channel::Key, x::telem::Authority> merged;
    for (size_t i = 0; i < change.keys.size(); i++)
        merged[change.keys[i]] = change.authorities[i];
    EXPECT_EQ(merged[1], 50);
    EXPECT_EQ(merged[2], 200);
}

/// @brief when a source returns both a frame and authority changes in the same
/// read, set_authority must be called BEFORE write so the frame is sent at the
/// new authority level.
TEST(AcquisitionPipeline, testAuthorityAppliedBeforeWrite) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    Authorities auth{
        .keys = {1},
        .authorities = {250},
    };
    const auto source = std::make_shared<AuthoritySource>(
        x::telem::TimeStamp::now(),
        auth
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    // Find the first SetAuthority and first Write in the ops log.
    // SetAuthority must come before Write.
    size_t first_set_auth = SIZE_MAX;
    size_t first_write = SIZE_MAX;
    for (size_t i = 0; i < mock_factory->ops->size(); i++) {
        if (mock_factory->ops->at(i) == mock::OpType::SetAuthority &&
            first_set_auth == SIZE_MAX)
            first_set_auth = i;
        if (mock_factory->ops->at(i) == mock::OpType::Write && first_write == SIZE_MAX)
            first_write = i;
    }
    ASSERT_NE(first_set_auth, SIZE_MAX);
    ASSERT_NE(first_write, SIZE_MAX);
    EXPECT_LT(first_set_auth, first_write)
        << "set_authority (index " << first_set_auth
        << ") must be called before write (index " << first_write << ")";
}

/// @brief a global authority change buffered before the writer opens should clear
/// any previously buffered per-channel changes.
TEST(AcquisitionPipeline, testAuthorityBufferGlobalClearsChannels) {
    auto writes = std::make_shared<std::vector<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<mock::WriterFactory>(writes);

    std::vector<Authorities> pre_auths = {
        {.keys = {1, 2}, .authorities = {100, 200}},
        {.keys = {}, .authorities = {75}},
    };
    const auto source = std::make_shared<PreWriterAuthoritySource>(
        x::telem::TimeStamp::now(),
        pre_auths
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    ASSERT_EQ(mock_factory->authority_changes->size(), 1);
    const auto &change = mock_factory->authority_changes->at(0);
    EXPECT_TRUE(change.keys.empty());
    ASSERT_EQ(change.authorities.size(), 1);
    EXPECT_EQ(change.authorities[0], 75);
}
}
