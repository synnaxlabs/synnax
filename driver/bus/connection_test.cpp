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
#include <memory>
#include <mutex>
#include <thread>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/test/test.h"

#include "driver/bus/connection.h"
#include "driver/bus/testutil/testutil.h"

namespace driver::bus {
using namespace testutil;

namespace {
int opens(Wire &wire) {
    std::lock_guard lock(wire.mu);
    return wire.opens;
}

std::pair<std::shared_ptr<Connection>, x::errors::Error> acquire(
    Connections &connections,
    const std::string &key,
    const x::json::json &settings,
    const std::shared_ptr<Wire> &wire
) {
    return connections.acquire(key, settings, [&] {
        return std::make_shared<Connection>(Connection::Config{.open = opener(wire)});
    });
}

int closes(Wire &wire) {
    std::lock_guard lock(wire.mu);
    return wire.closes;
}

int attempts(Wire &wire) {
    std::lock_guard lock(wire.mu);
    return wire.attempts;
}

const auto UNREACHABLE = x::errors::Error(transport::UNREACHABLE_ERROR, "gone");
}

TEST(Connections, SharesOneTransportAmongTheTasksOfADevice) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    const auto settings = x::json::json{{"port", "/dev/ttyUSB0"}};
    const auto a = ASSERT_NIL_P(acquire(connections, "dev", settings, wire));
    const auto b = ASSERT_NIL_P(acquire(connections, "dev", settings, wire));
    EXPECT_EQ(a, b);
    ASSERT_NIL(a->lock().transport().second);
    ASSERT_NIL(b->lock().transport().second);
    EXPECT_EQ(opens(*wire), 1);
}

TEST(Connections, ClosesTheTransportWhenTheLastTaskReleasesIt) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    auto a = ASSERT_NIL_P(acquire(connections, "dev", {}, wire));
    auto b = ASSERT_NIL_P(acquire(connections, "dev", {}, wire));
    ASSERT_NIL(a->lock().transport().second);
    a.reset();
    EXPECT_EQ(closes(*wire), 0);
    b.reset();
    EXPECT_EQ(closes(*wire), 1);
    const auto c = ASSERT_NIL_P(acquire(connections, "dev", {}, wire));
    ASSERT_NIL(c->lock().transport().second);
    EXPECT_EQ(opens(*wire), 2);
}

TEST(Connections, ForgetsADeviceWhenTheLastTaskReleasesIt) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    auto a = ASSERT_NIL_P(acquire(connections, "dev", {}, wire));
    ASSERT_NIL(a->lock().transport().second);
    EXPECT_EQ(connections.find("dev"), a);
    a.reset();
    EXPECT_EQ(connections.find("dev"), nullptr);
    EXPECT_EQ(closes(*wire), 1);
}

TEST(Connections, ClosesAConnectionReleasedAfterTheRegistry) {
    const auto wire = std::make_shared<Wire>();
    auto connections = std::make_unique<Connections>();
    auto a = ASSERT_NIL_P(acquire(*connections, "dev", {}, wire));
    ASSERT_NIL(a->lock().transport().second);
    connections.reset();
    EXPECT_EQ(closes(*wire), 0);
    a.reset();
    EXPECT_EQ(closes(*wire), 1);
}

TEST(Connections, RejectsOtherSettingsForAnOpenDevice) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    auto a = ASSERT_NIL_P(acquire(connections, "dev", {{"baud_rate", 9600}}, wire));
    ASSERT_OCCURRED_AS_P(
        acquire(connections, "dev", {{"baud_rate", 115200}}, wire),
        transport::CONFIG_ERROR
    );
    a.reset();
    ASSERT_NIL_P(acquire(connections, "dev", {{"baud_rate", 115200}}, wire));
}

TEST(Connections, OpensADeviceConnectionPerDevice) {
    const auto wire = std::make_shared<Wire>();
    Connections connections;
    const auto a = ASSERT_NIL_P(acquire(connections, "a", {}, wire));
    const auto b = ASSERT_NIL_P(acquire(connections, "b", {}, wire));
    EXPECT_NE(a, b);
}

TEST(Connection, ReopensTheTransportAfterTheBackoffWhenItFails) {
    const auto wire = std::make_shared<Wire>();
    Clock clock;
    Connection conn({.open = opener(wire), .now = clock.func()});
    {
        auto guard = conn.lock();
        ASSERT_NIL(guard.transport().second);
        EXPECT_EQ(guard.opens(), 1);
        guard.fail(UNREACHABLE);
        ASSERT_OCCURRED_AS_P(guard.transport(), transport::UNREACHABLE_ERROR);
    }
    EXPECT_EQ(closes(*wire), 1);
    {
        auto guard = conn.lock();
        const auto err = guard.transport().second;
        ASSERT_MATCHES(err, transport::UNREACHABLE_ERROR);
        EXPECT_EQ(err.data, "gone");
    }
    EXPECT_EQ(attempts(*wire), 1);
    clock.advance(250 * x::telem::MILLISECOND);
    auto guard = conn.lock();
    ASSERT_NIL(guard.transport().second);
    EXPECT_EQ(guard.opens(), 2);
}

TEST(Connection, DoublesTheBackoffAfterEachFailedOpenUpToTheMaximum) {
    const auto wire = std::make_shared<Wire>();
    for (int i = 0; i < 8; i++)
        wire->open_errs.push_back(UNREACHABLE);
    Clock clock;
    Connection conn({
        .open = opener(wire),
        .min_backoff = 100 * x::telem::MILLISECOND,
        .max_backoff = 1 * x::telem::SECOND,
        .now = clock.func(),
    });
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), transport::UNREACHABLE_ERROR);
    EXPECT_EQ(attempts(*wire), 1);
    for (const auto wait: {100, 200, 400, 800, 1000, 1000}) {
        clock.advance((wait - 1) * x::telem::MILLISECOND);
        ASSERT_OCCURRED_AS_P(conn.lock().transport(), transport::UNREACHABLE_ERROR);
        const auto before = attempts(*wire);
        clock.advance(x::telem::MILLISECOND);
        ASSERT_OCCURRED_AS_P(conn.lock().transport(), transport::UNREACHABLE_ERROR);
        EXPECT_EQ(attempts(*wire), before + 1) << "after a wait of " << wait << "ms";
    }
}

TEST(Connection, ResetsTheBackoffAfterASuccessfulOpen) {
    const auto wire = std::make_shared<Wire>();
    wire->open_errs = {UNREACHABLE, UNREACHABLE};
    Clock clock;
    Connection conn({
        .open = opener(wire),
        .min_backoff = 100 * x::telem::MILLISECOND,
        .now = clock.func(),
    });
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), transport::UNREACHABLE_ERROR);
    clock.advance(100 * x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), transport::UNREACHABLE_ERROR);
    clock.advance(200 * x::telem::MILLISECOND);
    conn.lock().fail(UNREACHABLE);
    clock.advance(100 * x::telem::MILLISECOND);
    ASSERT_NIL(conn.lock().transport().second);
    EXPECT_EQ(attempts(*wire), 4);
}

TEST(Connection, SharesTheBackoffAmongTheTasksOfADevice) {
    const auto wire = std::make_shared<Wire>();
    wire->open_errs = {UNREACHABLE};
    Clock clock;
    const auto conn = std::make_shared<Connection>(
        Connection::Config{.open = opener(wire), .now = clock.func()}
    );
    std::vector<std::thread> tasks;
    for (int i = 0; i < 4; i++)
        tasks.emplace_back([&] {
            for (int j = 0; j < 10; j++)
                ASSERT_OCCURRED_AS_P(
                    conn->lock().transport(),
                    transport::UNREACHABLE_ERROR
                );
        });
    for (auto &t: tasks)
        t.join();
    EXPECT_EQ(attempts(*wire), 1);
}

TEST(Connection, OpensWithoutBlockingTheTaskThatHoldsTheLock) {
    const auto wire = std::make_shared<Wire>();
    std::mutex mu;
    std::condition_variable cv;
    int calls = 0;
    bool opening = false;
    bool release = false;
    const auto open = opener(wire);
    Clock clock;
    Connection conn({
        .open =
            [&] {
                std::unique_lock lock(mu);
                if (calls++ > 0) {
                    opening = true;
                    cv.notify_all();
                    cv.wait(lock, [&] { return release; });
                }
                lock.unlock();
                return open();
            },
        .min_backoff = x::telem::TimeSpan::ZERO(),
        .now = clock.func(),
    });
    std::thread opener_task;
    {
        auto guard = conn.lock();
        ASSERT_NIL(guard.transport().second);
        guard.fail(UNREACHABLE);
        opener_task = std::thread([&] { ASSERT_NIL(conn.lock().transport().second); });
        std::unique_lock lock(mu);
        cv.wait(lock, [&] { return opening; });
        lock.unlock();
        ASSERT_OCCURRED_AS_P(guard.transport(), transport::UNREACHABLE_ERROR);
    }
    {
        std::lock_guard lock(mu);
        release = true;
    }
    cv.notify_all();
    opener_task.join();
    EXPECT_EQ(opens(*wire), 2);
}

TEST(Connection, ReportsTheErrorOfTheLastOpen) {
    const auto wire = std::make_shared<Wire>();
    wire->open_errs.push_back(UNREACHABLE);
    Clock clock;
    Connection conn({.open = opener(wire), .now = clock.func()});
    ASSERT_NIL(conn.error());
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), transport::UNREACHABLE_ERROR);
    const auto err = conn.error();
    ASSERT_MATCHES(err, transport::UNREACHABLE_ERROR);
    EXPECT_EQ(err.data, "gone");
    EXPECT_EQ(conn.retry_at(), clock.now() + 250 * x::telem::MILLISECOND);
    clock.advance(250 * x::telem::MILLISECOND);
    ASSERT_NIL(conn.lock().transport().second);
    ASSERT_NIL(conn.error());
}

TEST(Connection, GrantsTheLockInTheOrderItWasAsked) {
    Connection conn({.open = opener(std::make_shared<Wire>())});
    std::atomic running = true;
    std::thread spinner([&] {
        while (running)
            auto guard = conn.lock();
    });
    const x::telem::Stopwatch sw;
    for (int i = 0; i < 1000; i++)
        auto guard = conn.lock();
    running = false;
    spinner.join();
    EXPECT_LT(sw.elapsed(), x::telem::SECOND);
}
}
