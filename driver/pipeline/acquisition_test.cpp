// Copyright 2026 Synnax Labs, Inc.
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

    xerrors::Error read(
        breaker::Breaker &breaker,
        telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        if (read_err != xerrors::NIL) return read_err;
        std::this_thread::sleep_for(std::chrono::microseconds(100));
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

    xerrors::Error read(
        breaker::Breaker &breaker,
        telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        if (read_err != xerrors::NIL) return read_err;
        std::this_thread::sleep_for(std::chrono::microseconds(100));

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
    xerrors::Error read(
        breaker::Breaker &breaker,
        telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.emplace(1, telem::Series(std::vector<float>{1.0f, 2.0f, 3.0f}));
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {}
};

/// @brief MockSource that returns frames with empty timestamp series
class EmptyTimestampSource final : public pipeline::Source {
public:
    xerrors::Error read(
        breaker::Breaker &breaker,
        telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
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

class AuthoritySource final : public pipeline::Source {
    telem::TimeStamp start_ts;
    pipeline::Authorities auth;
    bool sent_auth = false;

public:
    explicit AuthoritySource(
        const telem::TimeStamp start_ts,
        pipeline::Authorities auth
    ):
        start_ts(start_ts), auth(std::move(auth)) {}

    xerrors::Error read(
        breaker::Breaker &breaker,
        telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.emplace(1, telem::Series(this->start_ts));
        if (!this->sent_auth) {
            authorities = this->auth;
            this->sent_auth = true;
        }
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {}
};

/// @brief it should forward authority changes from Source to Writer
TEST(AcquisitionPipeline, testAuthorityForwarding) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    pipeline::Authorities auth{
        .keys = {1, 2},
        .authorities = {100, 200},
    };
    const auto source = std::make_shared<AuthoritySource>(
        telem::TimeStamp::now(),
        auth
    );

    auto pipe = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
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

class MultiAuthoritySource final : public pipeline::Source {
    telem::TimeStamp start_ts;
    std::vector<pipeline::Authorities> pending;
    std::atomic<size_t> read_count{0};

public:
    explicit MultiAuthoritySource(
        const telem::TimeStamp start_ts,
        std::vector<pipeline::Authorities> pending
    ):
        start_ts(start_ts), pending(std::move(pending)) {}

    xerrors::Error read(
        breaker::Breaker &breaker,
        telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        fr.emplace(1, telem::Series(this->start_ts));
        auto idx = this->read_count.fetch_add(1);
        if (idx < this->pending.size()) authorities = this->pending[idx];
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {}
};

/// @brief it should forward multiple sequential authority changes across reads.
TEST(AcquisitionPipeline, testMultipleAuthorityChanges) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    std::vector<pipeline::Authorities> changes = {
        {.keys = {1}, .authorities = {100}},
        {.keys = {2}, .authorities = {200}},
        {.keys = {3}, .authorities = {50}},
    };
    const auto source = std::make_shared<MultiAuthoritySource>(
        telem::TimeStamp::now(),
        changes
    );

    auto pipe = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    pipeline::Authorities auth{
        .keys = {},
        .authorities = {150},
    };
    const auto source = std::make_shared<AuthoritySource>(
        telem::TimeStamp::now(),
        auth
    );

    auto pipe = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    pipeline::Authorities auth{
        .keys = {42},
        .authorities = {250},
    };
    const auto source = std::make_shared<AuthoritySource>(
        telem::TimeStamp::now(),
        auth
    );

    auto pipe = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
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

class EmptyFrameAuthoritySource final : public pipeline::Source {
    telem::TimeStamp start_ts;
    pipeline::Authorities auth;
    std::atomic<size_t> read_count{0};

public:
    explicit EmptyFrameAuthoritySource(
        const telem::TimeStamp start_ts,
        pipeline::Authorities auth
    ):
        start_ts(start_ts), auth(std::move(auth)) {}

    xerrors::Error read(
        breaker::Breaker &breaker,
        telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        auto idx = this->read_count.fetch_add(1);
        if (idx == 0) {
            fr.emplace(1, telem::Series(this->start_ts));
            return xerrors::NIL;
        }
        if (idx == 1) authorities = this->auth;
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {}
};

/// @brief it should forward authority changes even when the frame is empty.
TEST(AcquisitionPipeline, testAuthorityWithEmptyFrame) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    pipeline::Authorities auth{
        .keys = {10},
        .authorities = {200},
    };
    const auto source = std::make_shared<EmptyFrameAuthoritySource>(
        telem::TimeStamp::now(),
        auth
    );

    auto pipe = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    const auto &change = mock_factory->authority_changes->at(0);
    ASSERT_EQ(change.keys.size(), 1);
    EXPECT_EQ(change.keys[0], 10);
    EXPECT_EQ(change.authorities[0], 200);
}

class PreWriterAuthoritySource final : public pipeline::Source {
    telem::TimeStamp start_ts;
    std::vector<pipeline::Authorities> pre_writer_auths;
    std::atomic<size_t> read_count{0};

public:
    explicit PreWriterAuthoritySource(
        const telem::TimeStamp start_ts,
        std::vector<pipeline::Authorities> pre_writer_auths
    ):
        start_ts(start_ts), pre_writer_auths(std::move(pre_writer_auths)) {}

    xerrors::Error read(
        breaker::Breaker &breaker,
        telem::Frame &fr,
        pipeline::Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        auto idx = this->read_count.fetch_add(1);
        if (idx < this->pre_writer_auths.size()) {
            authorities = this->pre_writer_auths[idx];
            return xerrors::NIL;
        }
        fr.emplace(1, telem::Series(this->start_ts));
        return xerrors::NIL;
    }

    void stopped_with_err(const xerrors::Error &err) override {}
};

/// @brief it should buffer authority changes that arrive before the writer opens
/// and apply them once the first frame opens the writer.
TEST(AcquisitionPipeline, testAuthorityBufferedBeforeWriter) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    std::vector<pipeline::Authorities> pre_auths = {
        {.keys = {1}, .authorities = {100}},
    };
    const auto source = std::make_shared<PreWriterAuthoritySource>(
        telem::TimeStamp::now(),
        pre_auths
    );

    auto pipe = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
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
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    std::vector<pipeline::Authorities> pre_auths = {
        {.keys = {1, 2}, .authorities = {100, 200}},
        {.keys = {1}, .authorities = {50}},
    };
    const auto source = std::make_shared<PreWriterAuthoritySource>(
        telem::TimeStamp::now(),
        pre_auths
    );

    auto pipe = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(mock_factory->authority_changes->size(), 1);
    ASSERT_TRUE(pipe.stop());

    const auto &change = mock_factory->authority_changes->at(0);
    ASSERT_EQ(change.keys.size(), 2);
    std::map<synnax::ChannelKey, telem::Authority> merged;
    for (size_t i = 0; i < change.keys.size(); i++)
        merged[change.keys[i]] = change.authorities[i];
    EXPECT_EQ(merged[1], 50);
    EXPECT_EQ(merged[2], 200);
}

/// @brief a global authority change buffered before the writer opens should clear
/// any previously buffered per-channel changes.
TEST(AcquisitionPipeline, testAuthorityBufferGlobalClearsChannels) {
    auto writes = std::make_shared<std::vector<telem::Frame>>();
    const auto mock_factory = std::make_shared<pipeline::mock::WriterFactory>(writes);

    std::vector<pipeline::Authorities> pre_auths = {
        {.keys = {1, 2}, .authorities = {100, 200}},
        {.keys = {}, .authorities = {75}},
    };
    const auto source = std::make_shared<PreWriterAuthoritySource>(
        telem::TimeStamp::now(),
        pre_auths
    );

    auto pipe = pipeline::Acquisition(
        mock_factory,
        synnax::WriterConfig(),
        source,
        breaker::Config()
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
