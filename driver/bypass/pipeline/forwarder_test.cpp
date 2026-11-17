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

using Warnings = x::sync::List<std::pair<std::string, std::string>>;

Warn record(const std::shared_ptr<Warnings> &warnings) {
    return [warnings](const std::string &message, const std::string &description) {
        warnings->push_back({message, description});
    };
}

constexpr size_t FRAME_SIZE = FRAME_OVERHEAD + SERIES_OVERHEAD + sizeof(float);

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
        {.max_queue_bytes = 10 * frame_size,
         .breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    for (int i = 0; i < 15; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_EQ(forwarder.dropped(), 5);
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(forwarder.dropped(), 15);
    ASSERT_EQ(server->writes->size(), 0);
}

/// @brief it should accept a frame larger than its cap when the queue is empty.
TEST(ForwarderTest, AcceptsALargeFrameIntoAnEmptyQueue) {
    const auto server = factory();
    Forwarder forwarder(server, {.channels = {1}}, {.max_queue_bytes = 2});
    ASSERT_NIL(forwarder.write(frame(1)));
    ASSERT_EVENTUALLY_EQ(server->writes->size(), 1);
    ASSERT_EQ(forwarder.dropped(), 0);
    ASSERT_NIL(forwarder.close());
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
    ASSERT_EQ(forwarder.dropped(), 3);
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

/// @brief it should warn once when the queue passes half its cap, stop warning once it
/// drains below a quarter, and warn again when it passes half again.
TEST(ForwarderTest, WarnsWhileFallingBehindTheCore) {
    const auto server = std::make_shared<GatedFactory>();
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.max_queue_bytes = 8 * FRAME_SIZE, .warn = record(warnings)}
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
    ASSERT_EQ(forwarder.dropped(), 0);
}

/// @brief it should report the count of dropped frames at most once per interval, and
/// keep it after the queue drains, until the forwarder closes.
TEST(ForwarderTest, ReportsDroppedFramesAtMostOncePerInterval) {
    const auto server = std::make_shared<GatedFactory>();
    const auto warnings = std::make_shared<Warnings>();
    Forwarder forwarder(
        server,
        {.channels = {1}},
        {.max_queue_bytes = 4 * FRAME_SIZE,
         .drop_report_interval = x::telem::HOUR,
         .warn = record(warnings)}
    );
    x::defer::defer release([&server] { server->allow(1000); });
    for (int i = 0; i < 5; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_EQ(forwarder.dropped(), 1);
    server->allow(1);
    const auto last = [&] { return warnings->at(warnings->size() - 1); };
    // The forwarder removed the first frame from its queue once it waits on the second.
    ASSERT_EVENTUALLY_EQ(server->started.load(), 2);
    ASSERT_EVENTUALLY_EQ(last().first, "Dropped frames");
    EXPECT_EQ(
        last().second,
        "Dropped 1 frame because the Core could not take them fast enough."
    );
    const auto reports = warnings->size();
    for (int i = 0; i < 2; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_EQ(forwarder.dropped(), 2);
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
        "Dropped 2 frames because the Core could not take them fast enough."
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
        {.warn = record(warnings), .breaker = {.base_interval = x::telem::MILLISECOND}}
    );
    for (int i = 0; i < 3; i++)
        ASSERT_NIL(forwarder.write(frame(static_cast<float>(i))));
    ASSERT_NIL(forwarder.close());
    ASSERT_EQ(warnings->size(), 1);
    EXPECT_EQ(warnings->at(0).first, "Dropped frames");
    EXPECT_EQ(
        warnings->at(0).second,
        "Dropped 3 frames because the Core could not take them fast enough."
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
        "Dropped 1 frame because the Core could not take them fast enough."
    );
}
}
