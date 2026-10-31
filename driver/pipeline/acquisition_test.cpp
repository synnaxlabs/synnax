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

#include "x/cpp/sync/list.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/testutil/pipeline.h"

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

/// @brief a source that fails or returns empty frames per a script, then returns a
/// frame on every read.
class ScriptedSource final : public Source {
    /// @brief the result of each scripted read, in order.
    std::vector<x::errors::Error> script;
    /// @brief the number of reads so far.
    size_t reads = 0;

public:
    /// @brief the error the pipeline stopped with, or nil.
    x::errors::Error stopped_err = x::errors::NIL;

    /// @param script one entry per read, in order. A nil entry reads an empty frame.
    explicit ScriptedSource(std::vector<x::errors::Error> script):
        script(std::move(script)) {}

    x::errors::Error
    read(x::breaker::Breaker &, x::telem::Frame &fr, Authorities &) override {
        if (this->reads < this->script.size()) return this->script[this->reads++];
        fr.emplace(1, x::telem::Series(x::telem::TimeStamp::now()));
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {
        this->stopped_err = err;
    }
};

const auto TEMPORARY = x::errors::Error(errors::TEMPORARY_HARDWARE_ERROR);

x::breaker::Config fast_breaker(const int max_retries) {
    return x::breaker::Config{
        .name = "pipeline",
        .base_interval = x::telem::MICROSECOND * 10,
        .max_retries = max_retries,
        .scale = 0,
    };
}

/// @brief it should correctly retry opening the writer when an unreachable error
/// occurs.
TEST(AcquisitionPipeline, RetriesWriterOpenOnUnreachable) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(
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
TEST(AcquisitionPipeline, DoesNotRetryWriterOpenOnUnauthorized) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(
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
    ASSERT_EVENTUALLY_TRUE(!pipeline.running());
    pipeline.stop();
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should retry opening the writer when write returns false and the
/// error is unreachable.
TEST(AcquisitionPipeline, ReopensWriterOnUnreachableWrite) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(
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
TEST(AcquisitionPipeline, DoesNotReopenWriterOnUnauthorizedWrite) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(
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
    ASSERT_EVENTUALLY_FALSE(pipeline.running());
    ASSERT_FALSE(pipeline.stop());
    ASSERT_EQ(mock_factory->writer_opens.load(std::memory_order_acquire), 1);
    ASSERT_EQ(source->stopped_err, x::errors::UNAUTHORIZED);
}

/// @brief it should not restart the pipeline if it has already been started.
TEST(AcquisitionPipeline, DoesNotRestartStartedPipeline) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
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
TEST(AcquisitionPipeline, DoesNotStopStoppedPipeline) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
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
TEST(AcquisitionPipeline, StopsAndReportsCriticalReadError) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
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

    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_FALSE(pipeline.running());
    ASSERT_FALSE(pipeline.stop());
    ASSERT_EQ(source->stopped_err, critical_error);
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should back off on a temporary read error and resume reading when the
/// source recovers.
TEST(AcquisitionPipeline, ResumesAfterTemporaryReadError) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
    const auto source = std::make_shared<ScriptedSource>(
        std::vector{TEMPORARY, TEMPORARY}
    );
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        fast_breaker(3)
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    ASSERT_TRUE(pipeline.stop());
    ASSERT_NIL(source->stopped_err);
}

/// @brief a successful read with an empty frame should reset the retry count, so
/// separate outages do not add up to the retry limit.
TEST(AcquisitionPipeline, EmptyReadResetsRetries) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
    const auto source = std::make_shared<ScriptedSource>(
        std::vector{TEMPORARY, x::errors::NIL, TEMPORARY, x::errors::NIL}
    );
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        fast_breaker(1)
    );
    ASSERT_TRUE(pipeline.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    ASSERT_TRUE(pipeline.stop());
    ASSERT_NIL(source->stopped_err);
}

/// @brief it should not stop the pipeline if it was never started.
TEST(AcquisitionPipeline, DoesNotStopUnstartedPipeline) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipeline = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_FALSE(pipeline.stop());
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
TEST(AcquisitionPipeline, NominalShutdownDoesNotReportError) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
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
TEST(AcquisitionPipeline, ForwardsAuthorityChanges) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);

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
TEST(AcquisitionPipeline, ForwardsSequentialAuthorityChanges) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);

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
TEST(AcquisitionPipeline, ForwardsGlobalAuthorityChanges) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);

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
TEST(AcquisitionPipeline, ForwardsAuthorityWithEmptyFrame) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);

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

/// @brief a source that reads authority changes, one per read, before it reads any
/// frame.
class AuthorityBeforeDataSource final : public Source {
    x::telem::TimeStamp start_ts;
    std::vector<Authorities> leading_auths;
    std::atomic<size_t> read_count{0};

public:
    explicit AuthorityBeforeDataSource(
        const x::telem::TimeStamp start_ts,
        std::vector<Authorities> leading_auths
    ):
        start_ts(start_ts), leading_auths(std::move(leading_auths)) {}

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        std::this_thread::sleep_for(std::chrono::microseconds(100));
        auto idx = this->read_count.fetch_add(1);
        if (idx < this->leading_auths.size()) {
            authorities = this->leading_auths[idx];
            return x::errors::NIL;
        }
        fr.emplace(1, x::telem::Series(this->start_ts));
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief it should forward each authority change that arrives before the first
/// frame, in order.
TEST(AcquisitionPipeline, ForwardsAuthorityBeforeFirstFrame) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
    const auto source = std::make_shared<AuthorityBeforeDataSource>(
        x::telem::TimeStamp::now(),
        std::vector<Authorities>{
            {.keys = {1, 2}, .authorities = {100, 200}},
            {.keys = {1}, .authorities = {50}},
        }
    );
    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    ASSERT_TRUE(pipe.stop());
    ASSERT_EQ(mock_factory->authority_changes->size(), 2);
    const auto &first = mock_factory->authority_changes->at(0);
    EXPECT_EQ(first.keys, (std::vector<synnax::channel::Key>{1, 2}));
    EXPECT_EQ(first.authorities, (std::vector<x::control::Authority>{100, 200}));
    const auto &second = mock_factory->authority_changes->at(1);
    EXPECT_EQ(second.keys, (std::vector<synnax::channel::Key>{1}));
    EXPECT_EQ(second.authorities, (std::vector<x::control::Authority>{50}));
}

/// @brief when a source returns both a frame and authority changes in the same
/// read, set_authority must be called BEFORE write so the frame is sent at the
/// new authority level.
TEST(AcquisitionPipeline, SetsAuthorityBeforeWrite) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);

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
        if (mock_factory->ops->at(i) == testutil::OpType::SetAuthority &&
            first_set_auth == SIZE_MAX)
            first_set_auth = i;
        if (mock_factory->ops->at(i) == testutil::OpType::Write &&
            first_write == SIZE_MAX)
            first_write = i;
    }
    ASSERT_NE(first_set_auth, SIZE_MAX);
    ASSERT_NE(first_write, SIZE_MAX);
    EXPECT_LT(first_set_auth, first_write)
        << "set_authority (index " << first_set_auth
        << ") must be called before write (index " << first_write << ")";
}

/// @brief err_on_unauthorized should default to true on the writer config when no
/// explicit value is provided (the default for hardware acquisition tasks).
TEST(AcquisitionPipeline, ErrOnUnauthorizedDefaultsToTrue) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    ASSERT_TRUE(pipe.stop());
    ASSERT_TRUE(mock_factory->config.err_on_unauthorized);
}

/// @brief err_on_unauthorized should be set to false on the writer config when the
/// pipeline is constructed with err_on_unauthorized = false (e.g. for Arc tasks that
/// need authority handoff).
TEST(AcquisitionPipeline, ErrOnUnauthorizedCanBeDisabled) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config(),
        "",
        false
    );
    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 1);
    ASSERT_TRUE(pipe.stop());
    ASSERT_FALSE(mock_factory->config.err_on_unauthorized);
}

class NeverProducesSource final : public Source {
    std::atomic<size_t> read_count{0};

public:
    std::atomic<size_t> &reads() { return this->read_count; }

    x::errors::Error read(
        x::breaker::Breaker &breaker,
        x::telem::Frame &fr,
        Authorities &authorities
    ) override {
        this->read_count.fetch_add(1);
        std::this_thread::sleep_for(std::chrono::milliseconds(1));
        return x::errors::NIL;
    }

    void stopped_with_err(const x::errors::Error &err) override {}
};

/// @brief it should open the writer when the pipeline starts, before the source reads
/// a frame, and leave the start of the writer to the Core.
TEST(AcquisitionPipeline, OpensWriterBeforeFirstFrame) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
    const auto source = std::make_shared<NeverProducesSource>();
    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(source->reads().load(), 3);
    ASSERT_TRUE(pipe.stop());
    ASSERT_EQ(mock_factory->writer_opens.load(std::memory_order_acquire), 1);
    ASSERT_EQ(mock_factory->config.start, x::telem::TimeStamp(0));
    ASSERT_EQ(writes->size(), 0);
}

/// @brief it should open a new writer each time the pipeline restarts.
TEST(AcquisitionPipeline, ReopensWriterOnRestart) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);
    const auto source = std::make_shared<MockSource>(x::telem::TimeStamp::now());
    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );
    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 3);
    ASSERT_TRUE(pipe.stop());
    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_GE(writes->size(), 6);
    ASSERT_TRUE(pipe.stop());
    ASSERT_EQ(mock_factory->writer_opens.load(std::memory_order_acquire), 2);
}

/// @brief it should stop the pipeline when the source returns invalid authorities.
TEST(AcquisitionPipeline, InvalidAuthoritiesStopPipeline) {
    auto writes = std::make_shared<x::sync::List<x::telem::Frame>>();
    const auto mock_factory = std::make_shared<testutil::WriterFactory>(writes);

    Authorities invalid_auth{
        .keys = {1, 2, 3},
        .authorities = {100, 200},
    };
    const auto source = std::make_shared<AuthoritySource>(
        x::telem::TimeStamp::now(),
        invalid_auth
    );

    auto pipe = Acquisition(
        mock_factory,
        synnax::framer::WriterConfig(),
        source,
        x::breaker::Config()
    );

    ASSERT_TRUE(pipe.start());
    ASSERT_EVENTUALLY_TRUE(!pipe.running());
    pipe.stop();
    EXPECT_TRUE(mock_factory->authority_changes->empty());
}

/// @brief validate should return nil for empty authorities.
TEST(AuthoritiesValidation, AcceptsEmptyAuthorities) {
    Authorities auth;
    ASSERT_NIL(auth.validate());
}

/// @brief validate should return nil for a single authority (broadcast).
TEST(AuthoritiesValidation, AcceptsSingleAuthorityBroadcast) {
    Authorities auth{.keys = {1, 2, 3}, .authorities = {200}};
    ASSERT_NIL(auth.validate());
}

/// @brief validate should return nil when keys and authorities have matching sizes.
TEST(AuthoritiesValidation, AcceptsMatchingSizes) {
    Authorities auth{.keys = {1, 2, 3}, .authorities = {100, 200, 255}};
    ASSERT_NIL(auth.validate());
}

/// @brief validate should return nil for a single authority with no keys.
TEST(AuthoritiesValidation, AcceptsSingleAuthorityWithoutKeys) {
    Authorities auth{.authorities = {200}};
    ASSERT_NIL(auth.validate());
}

/// @brief validate should return a validation error when sizes mismatch.
TEST(AuthoritiesValidation, RejectsMismatchedSizes) {
    Authorities auth{.keys = {1, 2, 3}, .authorities = {100, 200}};
    ASSERT_OCCURRED_AS(auth.validate(), x::errors::VALIDATION);
}

/// @brief validate should return a validation error for many authorities with no keys.
TEST(AuthoritiesValidation, RejectsManyAuthoritiesWithoutKeys) {
    Authorities auth{.authorities = {100, 200}};
    ASSERT_OCCURRED_AS(auth.validate(), x::errors::VALIDATION);
}
}
