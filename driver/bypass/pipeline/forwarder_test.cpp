// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <condition_variable>
#include <future>
#include <mutex>
#include <thread>

#include "gtest/gtest.h"

#include "client/cpp/errors/errors.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/sync/list.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/bypass/pipeline/forwarder.h"
#include "driver/pipeline/testutil/pipeline.h"

namespace driver::bypass::pipeline {
namespace {
x::telem::Frame frame(const float value) {
    x::telem::Frame fr;
    fr.emplace(1, x::telem::Series(value));
    return fr;
}

/// @brief opens Core writers that take a frame only once the test allows it.
class GatedFactory final : public ::driver::pipeline::WriterFactory {
    std::mutex mu;
    std::condition_variable cv;
    size_t credits = 0;

    class Writer final : public ::driver::pipeline::Writer {
        GatedFactory &factory;

    public:
        explicit Writer(GatedFactory &factory): factory(factory) {}

        x::errors::Error write(const x::telem::Frame &) override {
            std::unique_lock lock(this->factory.mu);
            this->factory.started++;
            this->factory.cv.wait(lock, [this] { return this->factory.credits > 0; });
            this->factory.credits--;
            this->factory.written++;
            return x::errors::NIL;
        }

        x::errors::Error close() override { return x::errors::NIL; }
    };

public:
    /// @brief the writes that began, including one that waits.
    std::atomic<size_t> started = 0;
    std::atomic<size_t> written = 0;

    void allow(const size_t frames) {
        {
            std::lock_guard lock(this->mu);
            this->credits += frames;
        }
        this->cv.notify_all();
    }

    std::pair<std::unique_ptr<::driver::pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &) override {
        return {std::make_unique<Writer>(*this), x::errors::NIL};
    }
};

/// @brief opens one Core writer whose stream the test ends while a write blocks, and
/// records where the forwarder closes it. Later opens find the Core unreachable.
class RacingFactory final : public ::driver::pipeline::WriterFactory {
    std::mutex mu;
    std::condition_variable cv;
    bool released = false;
    bool ended = false;
    bool closed = false;

    class Writer final : public ::driver::pipeline::Writer {
        RacingFactory &factory;

    public:
        explicit Writer(RacingFactory &factory): factory(factory) {}

        x::errors::Error write(const x::telem::Frame &) override {
            std::unique_lock lock(this->factory.mu);
            this->factory.write_thread = std::this_thread::get_id();
            this->factory.writing = true;
            this->factory.cv.wait(lock, [this] { return this->factory.released; });
            this->factory.writing = false;
            return x::errors::NIL;
        }

        x::errors::Error close() override {
            {
                std::lock_guard lock(this->factory.mu);
                this->factory.close_thread = std::this_thread::get_id();
                this->factory.closed_while_writing = this->factory.writing.load();
                this->factory.closed = true;
            }
            this->factory.cv.notify_all();
            return freighter::UNREACHABLE;
        }

        bool wait() override {
            std::unique_lock lock(this->factory.mu);
            this->factory.cv.wait(lock, [this] {
                return this->factory.ended || this->factory.closed;
            });
            return true;
        }
    };

public:
    std::atomic<bool> writing = false;
    std::atomic<bool> closed_while_writing = false;
    std::thread::id write_thread;
    std::thread::id close_thread;
    std::atomic<size_t> opens = 0;

    void end() {
        {
            std::lock_guard lock(this->mu);
            this->ended = true;
        }
        this->cv.notify_all();
    }

    void release() {
        {
            std::lock_guard lock(this->mu);
            this->released = true;
        }
        this->cv.notify_all();
    }

    std::pair<std::unique_ptr<::driver::pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &) override {
        if (this->opens++ > 0) return {nullptr, freighter::UNREACHABLE};
        return {std::make_unique<Writer>(*this), x::errors::NIL};
    }
};

using Warnings = x::sync::List<std::pair<std::string, std::string>>;

Warn record(const std::shared_ptr<Warnings> &warnings) {
    return [warnings](const std::string &message, const std::string &description) {
        warnings->push_back({message, description});
    };
}

constexpr size_t FRAME_SIZE = FRAME_OVERHEAD + SERIES_OVERHEAD + sizeof(float);

/// @brief a drain timeout short enough that a stop against an unreachable Core ends
/// soon, since the forwarder tries the Core until the timeout passes.
const x::telem::TimeSpan SHORT_DRAIN = 10 * x::telem::MILLISECOND;

std::shared_ptr<::driver::pipeline::testutil::WriterFactory>
factory(std::vector<x::errors::Error> open_errors = {}) {
    return std::make_shared<::driver::pipeline::testutil::WriterFactory>(
        std::make_shared<x::sync::List<x::telem::Frame>>(),
        std::move(open_errors)
    );
}
}

/// @brief it should count the memory each queued frame holds, so a stream of
/// one-sample frames fills its queue cap at the expected count while the Core is
/// unreachable.
TEST(ForwarderTest, DropsFramesPastTheQueueCap) {
    const auto server = factory(
        std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)
    );
    constexpr size_t frame_size = FRAME_OVERHEAD + SERIES_OVERHEAD + sizeof(float);
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.budget = std::make_shared<Budget>(10 * frame_size),
         .drain_timeout = SHORT_DRAIN,
         .breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    for (int i = 0; i < 15; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_EQ(forwarder.dropped().full, 5);
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(forwarder.dropped().full, 5);
    ASSERT_EQ(forwarder.dropped().stopped, 10);
    ASSERT_EQ(server->writes->size(), 0);
}

/// @brief it should accept a frame larger than its budget when the queue is empty.
TEST(ForwarderTest, AcceptsALargeFrameIntoAnEmptyQueue) {
    const auto server = factory();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.budget = std::make_shared<Budget>(2)}
    );
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 1);
    ASSERT_EQ(forwarder.dropped().total(), 0);
    ASSERT_NIL(forwarder.close());
}

/// @brief it should drop the frames of one forwarder once another fills the budget
/// that they share.
TEST(ForwarderTest, SharesItsBudgetWithOtherForwarders) {
    const auto unreachable = [] {
        return factory(std::vector<x::errors::Error>(1000, freighter::UNREACHABLE));
    };
    const auto budget = std::make_shared<Budget>(4 * FRAME_SIZE);
    const ForwarderConfig cfg{
        .budget = budget,
        .drain_timeout = SHORT_DRAIN,
        .breaker = {.base_interval = x::telem::MILLISECOND},
    };
    Forwarder first(unreachable(), {.channels = {1}}, cfg);
    Forwarder second(unreachable(), {.channels = {1}}, cfg);
    for (int i = 0; i < 4; i++)
        ASSERT_NIL(first.write(frame(static_cast<float>(i))));
    ASSERT_EQ(budget->used(), 4 * FRAME_SIZE);
    ASSERT_NIL(second.write(frame(0)));
    ASSERT_EQ(first.dropped().full, 0);
    ASSERT_EQ(second.dropped().full, 1);
    ASSERT_NIL(first.close());
    ASSERT_EQ(budget->used(), 0);
    ASSERT_NIL(second.write(frame(1)));
    ASSERT_EQ(second.dropped().full, 1);
    ASSERT_EQ(budget->used(), FRAME_SIZE);
    ASSERT_NIL(second.close());
}

/// @brief it should stop warning that it falls behind once its own queue empties, even
/// while another forwarder holds most of the budget.
TEST(ForwarderTest, ClearsItsLagWarningOnceItsQueueEmpties) {
    const auto budget = std::make_shared<Budget>(8 * FRAME_SIZE);
    Forwarder other(
        factory(std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)),
        {.channels = {1}},
        {.budget = budget,
         .drain_timeout = SHORT_DRAIN,
         .breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    for (int i = 0; i < 5; i++)
        ASSERT_NIL(other.write(frame(static_cast<float>(i))));
    const auto server = std::make_shared<GatedFactory>();
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.budget = budget, .warn = record(warnings)}
    );
    x::defer::defer release([&server] { server->allow(1000); });
    for (int i = 0; i < 2; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    server->allow(1);
    ASSERT_EVENTUALLY_EQ(warnings->size(), 1);
    EXPECT_EQ(warnings->at(0).first, "Falling behind the Core");
    server->allow(1);
    ASSERT_EVENTUALLY_EQ(warnings->size(), 2);
    EXPECT_EQ(warnings->at(1).first, "");
    ASSERT_EQ(budget->used(), 5 * FRAME_SIZE);
    ASSERT_NIL(forwarder.close());
    ASSERT_NIL(other.close());
}

/// @brief it should report the frames it drops while it waits with an empty queue,
/// because other forwarders fill the budget.
TEST(ForwarderTest, ReportsDropsWhileIdle) {
    const auto budget = std::make_shared<Budget>(4 * FRAME_SIZE);
    const auto server = factory();
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.budget = budget,
         .drop_report_interval = 50 * x::telem::MILLISECOND,
         .warn = record(warnings)}
    );
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_EVENTUALLY_EQ(budget->used(), 0);
    Forwarder other(
        factory(std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)),
        {.channels = {1}},
        {.budget = budget,
         .drain_timeout = SHORT_DRAIN,
         .breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    for (int i = 0; i < 4; i++)
        ASSERT_NIL(other.write(frame(static_cast<float>(i))));
    const auto last = [&] {
        if (warnings->empty()) return std::pair<std::string, std::string>{};
        return warnings->at(warnings->size() - 1);
    };
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_EVENTUALLY_EQ(
        last().second,
        "Dropped 1 frame because the Driver's queue for the Core was full."
    );
    ASSERT_NIL(forwarder.write(frame(2)));
    ASSERT_EVENTUALLY_EQ(
        last().second,
        "Dropped 2 frames because the Driver's queue for the Core was full."
    );
    ASSERT_EQ(server->writes->size(), 1);
    ASSERT_NIL(other.close());
    ASSERT_NIL(forwarder.close());
}

/// @brief it should give back the bytes of each op it sends to the Core.
TEST(ForwarderTest, ReleasesItsBudgetOnSend) {
    const auto server = std::make_shared<GatedFactory>();
    const auto budget = std::make_shared<Budget>(8 * FRAME_SIZE);
    Forwarder forwarder(server, {.channels = {1}}, {.budget = budget});
    x::defer::defer release([&server] { server->allow(1000); });
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {200}}));
    const auto held = budget->used();
    ASSERT_GT(held, 3 * FRAME_SIZE);
    server->allow(1);
    ASSERT_EVENTUALLY_EQ(budget->used(), held - FRAME_SIZE);
    server->allow(2);
    ASSERT_EVENTUALLY_EQ(budget->used(), 0);
    ASSERT_NIL(forwarder.close());
}

/// @brief it should give back the bytes of the frames it drops when it stops or a Core
/// error stops it.
TEST(ForwarderTest, ReleasesItsBudgetOnDrop) {
    const auto budget = std::make_shared<Budget>(8 * FRAME_SIZE);
    const auto unreachable = factory(
        std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)
    );
    Forwarder stopped(
        unreachable,
        {.channels = {1}},
        {.budget = budget,
         .drain_timeout = SHORT_DRAIN,
         .breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(stopped.write(frame(static_cast<float>(i))));
    ASSERT_EQ(budget->used(), 3 * FRAME_SIZE);
    ASSERT_NIL(stopped.close());
    ASSERT_EQ(stopped.dropped().stopped, 3);
    ASSERT_EQ(budget->used(), 0);

    const auto failing = factory();
    std::promise<void> gate;
    failing->open_gate = gate.get_future().share();
    failing->open_errors = {x::errors::VALIDATION};
    Forwarder failed(failing, {.channels = {1}}, {.budget = budget});
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(failed.write(frame(static_cast<float>(i))));
    ASSERT_EQ(budget->used(), 3 * FRAME_SIZE);
    gate.set_value();
    ASSERT_OCCURRED_AS(failed.opened(), x::errors::VALIDATION);
    ASSERT_EQ(failed.dropped().failed, 3);
    ASSERT_EQ(budget->used(), 0);
    ASSERT_OCCURRED_AS(failed.close(), x::errors::VALIDATION);
}

/// @brief it should send what it holds to a reachable Core before close returns.
TEST(ForwarderTest, SendsItsQueueOnClose) {
    const auto server = factory();
    std::promise<void> gate;
    server->open_gate = gate.get_future().share();
    Forwarder forwarder(server, {.channels = {1}});
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {200}}));
    gate.set_value();
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(server->writes->size(), 3);
    ASSERT_EQ(server->authority_changes->size(), 1);
}

/// @brief it should stop at once on a license error, so its owner retries it, and
/// return the error from later calls.
TEST(ForwarderTest, StopsOnALicenseError) {
    const auto server = factory({synnax::errors::LICENSE_MISSING});
    Forwarder forwarder(server, {.channels = {1}});
    ASSERT_OCCURRED_AS(forwarder.opened(), synnax::errors::LICENSE_MISSING);
    ASSERT_EQ(server->writer_opens.load(), 1);
    ASSERT_OCCURRED_AS(forwarder.write(frame(1)), synnax::errors::LICENSE_MISSING);
    ASSERT_OCCURRED_AS(
        forwarder.set_authority({.keys = {1}, .authorities = {200}}),
        synnax::errors::LICENSE_MISSING
    );
    ASSERT_OCCURRED_AS(forwarder.close(), synnax::errors::LICENSE_MISSING);
}

/// @brief it should count the frames it holds when a permanent error stops it.
TEST(ForwarderTest, CountsHeldFramesOnAPermanentError) {
    const auto server = factory();
    std::promise<void> gate;
    server->open_gate = gate.get_future().share();
    server->open_errors = {x::errors::VALIDATION};
    Forwarder forwarder(server, {.channels = {1}});
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    gate.set_value();
    ASSERT_OCCURRED_AS(forwarder.opened(), x::errors::VALIDATION);
    ASSERT_EQ(forwarder.dropped().failed, 3);
    ASSERT_OCCURRED_AS(forwarder.close(), x::errors::VALIDATION);
}

/// @brief it should send frames and authority changes in queue order, on one writer
/// and on the writer it reopens after the Core becomes unreachable.
TEST(ForwarderTest, SendsOpsInQueueOrderAcrossAReopen) {
    const auto server = factory();
    std::promise<void> gate;
    server->open_gate = gate.get_future().share();
    server->return_false_ok_on = {1};
    server->close_errors = {freighter::UNREACHABLE};
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {100}}));
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {200}}));
    ASSERT_NIL(forwarder.write(frame(2)));
    gate.set_value();
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 3);
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(server->writer_opens.load(), 2);
    using ::driver::pipeline::testutil::OpType;
    const std::vector expected_ops = {
        OpType::Write,
        OpType::SetAuthority,
        OpType::Write,
        OpType::Write,
        OpType::SetAuthority,
        OpType::Write,
    };
    ASSERT_EQ(server->ops->size(), expected_ops.size());
    for (size_t i = 0; i < expected_ops.size(); i++)
        EXPECT_EQ(server->ops->at(i), expected_ops[i]);
    ASSERT_EQ(server->writes->size(), 3);
    for (size_t i = 0; i < 3; i++)
        EXPECT_EQ(server->writes->at(i).at<float>(1, 0), static_cast<float>(i));
    ASSERT_EQ(server->authority_changes->size(), 2);
    EXPECT_EQ(server->authority_changes->at(0).authorities[0], 100);
    EXPECT_EQ(server->authority_changes->at(1).authorities[0], 200);
    EXPECT_EQ(server->config.authorities, std::vector<x::control::Authority>{100});
}

/// @brief it should call on_close once for each Core writer it opens, a failed one
/// included.
TEST(ForwarderTest, ClosesEachWriterItOpens) {
    const auto server = factory();
    server->return_false_ok_on = {1};
    server->close_errors = {freighter::UNREACHABLE};
    std::atomic<size_t> opens = 0;
    std::atomic<size_t> closes = 0;
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {
            .on_open = [&opens] { opens++; },
            .on_close = [&closes] { closes++; },
            .breaker = {.base_interval = x::telem::MILLISECOND},
        }
    );
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_EVENTUALLY_EQ(opens.load(), 2);
    ASSERT_EQ(closes.load(), 1);
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(closes.load(), 2);
}

/// @brief it should close, back off from, and reopen an idle Core writer whose stream
/// fails, with no write to show the failure.
TEST(ForwarderTest, ReopensAnIdleWriterWhoseStreamFails) {
    const auto server = factory();
    std::atomic<size_t> opens = 0;
    std::atomic<size_t> closes = 0;
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {
            .on_open = [&opens] { opens++; },
            .on_close = [&closes] { closes++; },
            .breaker = {.base_interval = x::telem::MILLISECOND},
        }
    );
    ASSERT_NIL(forwarder.opened());
    ASSERT_EQ(opens.load(), 1);
    server->end_stream(freighter::UNREACHABLE);
    ASSERT_EVENTUALLY_EQ(opens.load(), 2);
    ASSERT_EQ(closes.load(), 1);
    ASSERT_EQ(server->writer_opens.load(), 2);
    ASSERT_EQ(server->ops->size(), 0);
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(closes.load(), 2);
}

/// @brief it should close a Core writer on its own thread, after a write in progress
/// returns, when the writer's stream ends during that write.
TEST(ForwarderTest, ClosesAfterAWriteInProgressWhenTheStreamEnds) {
    const auto server = std::make_shared<RacingFactory>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_EVENTUALLY_TRUE(server->writing.load());
    server->end();
    server->release();
    ASSERT_EVENTUALLY_GE(server->opens.load(), 2);
    ASSERT_FALSE(server->closed_while_writing.load());
    ASSERT_EQ(server->close_thread, server->write_thread);
    ASSERT_NIL(forwarder.close());
}

/// @brief it should stop with the error that ends the stream of an idle Core writer
/// when the Core is reachable.
TEST(ForwarderTest, StopsWhenTheStreamFailsWithAnotherError) {
    const auto server = factory();
    Forwarder forwarder(server, {.channels = {1}});
    ASSERT_NIL(forwarder.opened());
    server->end_stream(x::errors::VALIDATION);
    ASSERT_EVENTUALLY_TRUE(forwarder.write(frame(1)).matches(x::errors::VALIDATION));
    ASSERT_OCCURRED_AS(forwarder.close(), x::errors::VALIDATION);
    ASSERT_EQ(server->writer_opens.load(), 1);
}

/// @brief it should warn once when the queue passes half its cap, stop warning once it
/// drains below a quarter, and warn again when it passes half again.
TEST(ForwarderTest, WarnsWhileFallingBehindTheCore) {
    const auto server = std::make_shared<GatedFactory>();
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.budget = std::make_shared<Budget>(8 * FRAME_SIZE), .warn = record(warnings)}
    );
    // A failed assertion must not leave the forwarder waiting on the Core forever.
    x::defer::defer release([&server] { server->allow(1000); });
    for (int i = 0; i < 5; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    server->allow(1);
    ASSERT_EVENTUALLY_EQ(warnings->size(), 1);
    EXPECT_EQ(warnings->at(0).first, "Falling behind the Core");
    server->allow(2);
    ASSERT_EVENTUALLY_EQ(server->written.load(), 3);
    EXPECT_EQ(warnings->size(), 1);
    server->allow(1);
    ASSERT_EVENTUALLY_EQ(warnings->size(), 2);
    EXPECT_EQ(warnings->at(1).first, "");
    for (int i = 0; i < 5; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    server->allow(1);
    ASSERT_EVENTUALLY_EQ(warnings->size(), 3);
    EXPECT_EQ(warnings->at(2).first, "Falling behind the Core");
    server->allow(100);
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(warnings->size(), 4);
    EXPECT_EQ(warnings->at(3).first, "");
    ASSERT_EQ(forwarder.dropped().total(), 0);
}

/// @brief it should report the count of dropped frames at most once per interval, and
/// keep it after the queue drains, until the forwarder closes.
TEST(ForwarderTest, ReportsDroppedFramesAtMostOncePerInterval) {
    const auto server = std::make_shared<GatedFactory>();
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.budget = std::make_shared<Budget>(4 * FRAME_SIZE),
         .drop_report_interval = x::telem::HOUR,
         .warn = record(warnings)}
    );
    x::defer::defer release([&server] { server->allow(1000); });
    for (int i = 0; i < 5; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_EQ(forwarder.dropped().full, 1);
    server->allow(1);
    const auto last = [&] { return warnings->at(warnings->size() - 1); };
    // The forwarder removed the first frame from its queue once it waits on the second.
    ASSERT_EVENTUALLY_EQ(server->started.load(), 2);
    ASSERT_EVENTUALLY_EQ(last().first, "Dropped frames");
    EXPECT_EQ(
        last().second,
        "Dropped 1 frame because the Driver's queue for the Core was full."
    );
    const auto reports = warnings->size();
    for (int i = 0; i < 2; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_EQ(forwarder.dropped().full, 2);
    server->allow(2);
    ASSERT_EVENTUALLY_EQ(server->written.load(), 3);
    EXPECT_EQ(warnings->size(), reports);
    server->allow(100);
    ASSERT_EVENTUALLY_EQ(server->written.load(), 5);
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_EVENTUALLY_EQ(server->written.load(), 6);
    EXPECT_EQ(last().first, "Dropped frames");
    ASSERT_NIL(forwarder.close());
    EXPECT_EQ(
        last().second,
        "Dropped 2 frames because the Driver's queue for the Core was full."
    );
}

/// @brief it should leave the count of the frames it drops on close for the stop
/// status.
TEST(ForwarderTest, ReportsFramesDroppedOnClose) {
    const auto server = factory(
        std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)
    );
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.drain_timeout = SHORT_DRAIN,
         .warn = record(warnings),
         .breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(warnings->size(), 1);
    EXPECT_EQ(warnings->at(0).first, "Dropped frames");
    EXPECT_EQ(
        warnings->at(0).second,
        "Dropped 3 frames because the task stopped before delivery to the Core."
    );
}

/// @brief it should report the frames it drops when a permanent error stops it.
TEST(ForwarderTest, ReportsFramesDroppedOnAPermanentError) {
    const auto server = factory();
    std::promise<void> gate;
    server->open_gate = gate.get_future().share();
    server->open_errors = {x::errors::VALIDATION};
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(server, {.channels = {1}}, {.warn = record(warnings)});
    ASSERT_NIL(forwarder.write(frame(0)));
    gate.set_value();
    ASSERT_OCCURRED_AS(forwarder.close(), x::errors::VALIDATION);
    ASSERT_EQ(warnings->size(), 1);
    EXPECT_EQ(
        warnings->at(0).second,
        "Dropped 1 frame because a Core error stopped delivery."
    );
}

/// @brief opens Core writers that take each frame after a delay.
class SlowFactory final : public ::driver::pipeline::WriterFactory {
    class Writer final : public ::driver::pipeline::Writer {
        SlowFactory &factory;

    public:
        explicit Writer(SlowFactory &factory): factory(factory) {}

        x::errors::Error write(const x::telem::Frame &) override {
            std::this_thread::sleep_for(this->factory.delay);
            this->factory.written++;
            return x::errors::NIL;
        }

        x::errors::Error close() override { return x::errors::NIL; }
    };

public:
    const std::chrono::milliseconds delay;
    std::atomic<size_t> written = 0;

    explicit SlowFactory(const std::chrono::milliseconds delay): delay(delay) {}

    std::pair<std::unique_ptr<::driver::pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &) override {
        return {std::make_unique<Writer>(*this), x::errors::NIL};
    }
};

/// @brief it should stop sending its queue to a slow Core once the drain timeout
/// passes, and count what is left as dropped by the stop.
TEST(ForwarderTest, StopsDrainingASlowCoreAtTheDrainTimeout) {
    const auto server = std::make_shared<SlowFactory>(std::chrono::milliseconds(20));
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.drain_timeout = 100 * x::telem::MILLISECOND, .warn = record(warnings)}
    );
    constexpr size_t frames = 500;
    for (size_t i = 0; i < frames; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    const auto start = std::chrono::steady_clock::now();
    ASSERT_NIL(forwarder.close());
    ASSERT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(2));
    const auto dropped = forwarder.dropped();
    ASSERT_GT(dropped.stopped, 0);
    ASSERT_EQ(dropped.full, 0);
    ASSERT_EQ(server->written.load() + dropped.stopped, frames);
    EXPECT_EQ(
        warnings->at(warnings->size() - 1).second,
        "Dropped " + std::to_string(dropped.stopped) +
            " frames because the task stopped before delivery to the Core."
    );
}

/// @brief it should report new drops at the report interval while it waits out a long
/// backoff from an unreachable Core.
TEST(ForwarderTest, ReportsDropsDuringALongBackoff) {
    const auto server = factory(
        std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)
    );
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.budget = std::make_shared<Budget>(2 * FRAME_SIZE),
         .drop_report_interval = 10 * x::telem::MILLISECOND,
         .drain_timeout = SHORT_DRAIN,
         .warn = record(warnings),
         .breaker = {.base_interval = x::telem::HOUR}}
    );
    ASSERT_NIL(forwarder.opened());
    for (int i = 0; i < 5; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    const auto last = [&] {
        if (warnings->empty()) return std::pair<std::string, std::string>{};
        return warnings->at(warnings->size() - 1);
    };
    ASSERT_EVENTUALLY_EQ(
        last().second,
        "Dropped 3 frames because the Driver's queue for the Core was full."
    );
    ASSERT_EQ(server->writer_opens.load(), 1);
    ASSERT_NIL(forwarder.close());
    EXPECT_EQ(
        warnings->at(warnings->size() - 1).second,
        "Dropped 3 frames because the Driver's queue for the Core was full. Dropped 2 frames "
        "because the task stopped before delivery to the Core."
    );
}

/// @brief it should hold frames for up to its forward interval, and send them at once
/// when an authority change comes.
TEST(ForwarderTest, HoldsFramesForTheForwardInterval) {
    const auto server = factory();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.forward_interval = x::telem::HOUR}
    );
    ASSERT_NIL(forwarder.opened());
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_NIL(forwarder.write(frame(1)));
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    ASSERT_EQ(server->writes->size(), 0);
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {200}}));
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 2);
    ASSERT_EQ(server->authority_changes->size(), 1);
    ASSERT_NIL(forwarder.close());
}

/// @brief it should send its queue before the forward interval passes once the queue
/// holds forward_size bytes.
TEST(ForwarderTest, SendsAtTheForwardSize) {
    const auto server = factory();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.forward_interval = x::telem::HOUR, .forward_size = 3 * FRAME_SIZE}
    );
    ASSERT_NIL(forwarder.opened());
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_NIL(forwarder.write(frame(1)));
    std::this_thread::sleep_for(std::chrono::milliseconds(50));
    ASSERT_EQ(server->writes->size(), 0);
    ASSERT_NIL(forwarder.write(frame(2)));
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 3);
    ASSERT_NIL(forwarder.close());
}

/// @brief it should send the frames it holds for the forward interval when it closes.
TEST(ForwarderTest, SendsHeldFramesOnClose) {
    const auto server = factory();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.forward_interval = x::telem::HOUR}
    );
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(server->writes->size(), 1);
    ASSERT_EQ(forwarder.dropped().total(), 0);
}

/// @brief it should try the Core again at once when it closes during a long backoff,
/// and send its queue to a Core that is back.
TEST(ForwarderTest, SendsItsQueueWhenTheCoreReturnsDuringAStop) {
    const auto server = factory({freighter::UNREACHABLE});
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.breaker = {.base_interval = x::telem::HOUR}}
    );
    ASSERT_NIL(forwarder.opened());
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(server->writer_opens.load(), 2);
    ASSERT_EQ(server->writes->size(), 3);
    ASSERT_EQ(forwarder.dropped().total(), 0);
}

/// @brief it should stop at once when it closes during an outage with nothing queued.
TEST(ForwarderTest, StopsAtOnceDuringAnOutageWithAnEmptyQueue) {
    const auto server = factory(
        std::vector<x::errors::Error>(1000, freighter::UNREACHABLE)
    );
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.breaker = {.base_interval = x::telem::HOUR}}
    );
    ASSERT_NIL(forwarder.opened());
    const auto start = std::chrono::steady_clock::now();
    ASSERT_NIL(forwarder.close());
    ASSERT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(1));
    ASSERT_EQ(server->writer_opens.load(), 1);
}

/// @brief opens Core writers whose writes block until the writer is cancelled.
class StuckFactory final : public ::driver::pipeline::WriterFactory {
    class Writer final : public ::driver::pipeline::Writer {
        StuckFactory &factory;

    public:
        explicit Writer(StuckFactory &factory): factory(factory) {}

        x::errors::Error write(const x::telem::Frame &) override {
            std::unique_lock lock(this->factory.mu);
            this->factory.cv.wait(lock, [this] { return this->factory.cancelled; });
            return freighter::STREAM_CLOSED;
        }

        x::errors::Error close() override { return x::errors::NIL; }

        void cancel() override {
            {
                std::lock_guard lock(this->factory.mu);
                this->factory.cancelled = true;
            }
            this->factory.cv.notify_all();
        }
    };

public:
    std::mutex mu;
    std::condition_variable cv;
    bool cancelled = false;

    std::pair<std::unique_ptr<::driver::pipeline::Writer>, x::errors::Error>
    open_writer(const synnax::framer::WriterConfig &) override {
        return {std::make_unique<Writer>(*this), x::errors::NIL};
    }
};

/// @brief it should cancel a Core write that blocks past the drain timeout, and count
/// what is left as dropped by the stop.
TEST(ForwarderTest, CancelsABlockedWriteAtTheDrainTimeout) {
    const auto server = std::make_shared<StuckFactory>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.drain_timeout = 100 * x::telem::MILLISECOND}
    );
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_NIL(forwarder.write(frame(1)));
    const auto start = std::chrono::steady_clock::now();
    ASSERT_NIL(forwarder.close());
    ASSERT_LT(std::chrono::steady_clock::now() - start, std::chrono::seconds(2));
    ASSERT_TRUE(server->cancelled);
    ASSERT_EQ(forwarder.dropped().stopped, 2);
}

/// @brief it should name the budget in the unit that fits it.
TEST(ForwarderTest, FormatsSizes) {
    EXPECT_EQ(format_size(512), "512 B");
    EXPECT_EQ(format_size(2048), "2 KB");
    EXPECT_EQ(format_size(size_t{256} * 1024 * 1024), "256 MB");
    EXPECT_EQ(format_size(size_t{3} * 1024 * 1024 * 1024), "3 GB");
}

/// @brief it should end a long backoff from an unreachable Core at once on retry.
TEST(ForwarderTest, RetryEndsTheBackoff) {
    const auto server = factory({freighter::UNREACHABLE});
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.breaker = {.base_interval = x::telem::HOUR}}
    );
    ASSERT_NIL(forwarder.opened());
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_EVENTUALLY_EQ(server->writer_opens.load(), 1);
    forwarder.retry();
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 1);
    ASSERT_EQ(server->writer_opens.load(), 2);
    ASSERT_NIL(forwarder.close());
}

/// @brief it should report the open of a Core writer only once the writer sends the
/// authority changes queued before the open, since it opens without them.
TEST(ForwarderTest, ReportsTheOpenOnceQueuedAuthoritiesAreSent) {
    const auto server = factory();
    std::promise<void> gate;
    server->open_gate = gate.get_future().share();
    std::atomic<size_t> sent_at_open = 1000;
    Forwarder forwarder(server, {.channels = {1}}, {.on_open = [&] {
                            sent_at_open = server->authority_changes->size();
                        }});
    ASSERT_NIL(forwarder.write(frame(0)));
    ASSERT_NIL(forwarder.set_authority({.keys = {1}, .authorities = {100}}));
    ASSERT_NIL(forwarder.write(frame(1)));
    gate.set_value();
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 2);
    ASSERT_EQ(sent_at_open.load(), 1);
    ASSERT_NIL(forwarder.close());
}
}
