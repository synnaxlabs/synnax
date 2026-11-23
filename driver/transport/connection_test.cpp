// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <cstdint>
#include <deque>
#include <memory>
#include <mutex>
#include <span>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/testutil/testutil.h"

#include "driver/transport/connection.h"
#include "driver/transport/errors.h"
#include "driver/transport/transport.h"

namespace driver::transport {
namespace {
const auto GONE = x::errors::Error(UNREACHABLE_ERROR, "gone");

/// @brief Device counts the opens and closes of the transports to a device.
struct Device {
    std::mutex mu;
    int attempts = 0;
    int opens = 0;
    int closes = 0;
    /// @brief failures holds the errors of the next opens, in order.
    std::deque<x::errors::Error> failures;

    int count(const int Device::*field) {
        std::lock_guard lock(this->mu);
        return this->*field;
    }
};

class Port final : public Transport {
    std::shared_ptr<Device> device;

public:
    explicit Port(std::shared_ptr<Device> device): device(std::move(device)) {}

    ~Port() override {
        std::lock_guard lock(this->device->mu);
        this->device->closes++;
    }

    std::pair<Chunk, x::errors::Error> read(x::telem::TimeSpan) override {
        return {{}, x::errors::NIL};
    }

    x::errors::Error write(std::span<const std::uint8_t>, x::telem::TimeSpan) override {
        return x::errors::NIL;
    }
};

Opener opener(const std::shared_ptr<Device> &device) {
    return [device]() -> std::pair<std::unique_ptr<Transport>, x::errors::Error> {
        std::lock_guard lock(device->mu);
        device->attempts++;
        if (!device->failures.empty()) {
            auto err = device->failures.front();
            device->failures.pop_front();
            return {nullptr, err};
        }
        device->opens++;
        return {std::make_unique<Port>(device), x::errors::NIL};
    };
}

/// @brief ManualClock is a steady clock that moves only when a spec advances it.
struct ManualClock {
    std::chrono::steady_clock::time_point time;

    Clock func() {
        return [this] { return this->time; };
    }

    void advance(const x::telem::TimeSpan span) { this->time += span.chrono(); }
};

std::pair<std::shared_ptr<Connection>, x::errors::Error> acquire(
    Connections &connections,
    const std::string &key,
    const x::json::json &settings,
    const std::shared_ptr<Device> &device
) {
    return connections.acquire(key, settings, {.open = opener(device)});
}
}

/// @brief it should share one transport among the tasks of a device.
TEST(Connections, ShareTransportAmongTasks) {
    const auto device = std::make_shared<Device>();
    Connections connections;
    const auto settings = x::json::json{{"port", "/dev/ttyUSB0"}};
    const auto a = ASSERT_NIL_P(acquire(connections, "dev", settings, device));
    const auto b = ASSERT_NIL_P(acquire(connections, "dev", settings, device));
    EXPECT_EQ(a, b);
    ASSERT_NIL_P(a->lock().transport());
    ASSERT_NIL_P(b->lock().transport());
    EXPECT_EQ(device->count(&Device::opens), 1);
}

/// @brief it should give each device its own connection.
TEST(Connections, KeepDevicesApart) {
    const auto device = std::make_shared<Device>();
    Connections connections;
    const auto a = ASSERT_NIL_P(acquire(connections, "a", {}, device));
    const auto b = ASSERT_NIL_P(acquire(connections, "b", {}, device));
    EXPECT_NE(a, b);
}

/// @brief it should close the transport when the last task releases it, and open a new
/// one for the next task.
TEST(Connections, CloseWhenLastTaskReleases) {
    const auto device = std::make_shared<Device>();
    Connections connections;
    auto a = ASSERT_NIL_P(acquire(connections, "dev", {}, device));
    auto b = ASSERT_NIL_P(acquire(connections, "dev", {}, device));
    ASSERT_NIL_P(a->lock().transport());
    a.reset();
    EXPECT_EQ(device->count(&Device::closes), 0);
    b.reset();
    EXPECT_EQ(device->count(&Device::closes), 1);
    const auto c = ASSERT_NIL_P(acquire(connections, "dev", {}, device));
    ASSERT_NIL_P(c->lock().transport());
    EXPECT_EQ(device->count(&Device::opens), 2);
}

/// @brief it should refuse other settings for a device that a task holds open.
TEST(Connections, RefuseOtherSettings) {
    const auto device = std::make_shared<Device>();
    Connections connections;
    auto a = ASSERT_NIL_P(acquire(connections, "dev", {{"baud_rate", 9600}}, device));
    const auto err = acquire(connections, "dev", {{"baud_rate", 115200}}, device)
                         .second;
    ASSERT_OCCURRED_AS(err, CONFIG_ERROR);
    EXPECT_EQ(err.data, "another task has device dev open with other settings");
    a.reset();
    ASSERT_NIL_P(acquire(connections, "dev", {{"baud_rate", 115200}}, device));
}

/// @brief it should close a failed transport, report its error, and open it again after
/// the backoff.
TEST(Connection, ReopenAfterBackoff) {
    const auto device = std::make_shared<Device>();
    ManualClock clock;
    Connection conn({.open = opener(device), .now = clock.func()});
    {
        auto guard = conn.lock();
        ASSERT_NIL_P(guard.transport());
        EXPECT_EQ(guard.opens(), 1);
        guard.fail(GONE);
        ASSERT_OCCURRED_AS_P(guard.transport(), UNREACHABLE_ERROR);
    }
    EXPECT_EQ(device->count(&Device::closes), 1);
    clock.advance(249 * x::telem::MILLISECOND);
    {
        auto guard = conn.lock();
        const auto err = guard.transport().second;
        ASSERT_OCCURRED_AS(err, UNREACHABLE_ERROR);
        EXPECT_EQ(err.data, "gone");
    }
    EXPECT_EQ(device->count(&Device::attempts), 1);
    clock.advance(x::telem::MILLISECOND);
    auto guard = conn.lock();
    ASSERT_NIL_P(guard.transport());
    EXPECT_EQ(guard.opens(), 2);
}

/// @brief it should double the backoff after each failed open, up to the max.
TEST(Connection, DoubleBackoffUpToMax) {
    const auto device = std::make_shared<Device>();
    for (int i = 0; i < 8; i++)
        device->failures.push_back(GONE);
    ManualClock clock;
    Connection conn({
        .open = opener(device),
        .min_backoff = 100 * x::telem::MILLISECOND,
        .max_backoff = x::telem::SECOND,
        .now = clock.func(),
    });
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), UNREACHABLE_ERROR);
    EXPECT_EQ(device->count(&Device::attempts), 1);
    for (const auto wait: {100, 200, 400, 800, 1000, 1000}) {
        clock.advance((wait - 1) * x::telem::MILLISECOND);
        ASSERT_OCCURRED_AS_P(conn.lock().transport(), UNREACHABLE_ERROR);
        const auto before = device->count(&Device::attempts);
        clock.advance(x::telem::MILLISECOND);
        ASSERT_OCCURRED_AS_P(conn.lock().transport(), UNREACHABLE_ERROR);
        EXPECT_EQ(device->count(&Device::attempts), before + 1) << wait << "ms";
    }
}

/// @brief it should keep doubling the backoff while each open fails at once, and reset
/// it after a turn that uses the transport without failing.
TEST(Connection, ResetBackoffAfterHealthyTurn) {
    const auto device = std::make_shared<Device>();
    ManualClock clock;
    Connection conn({
        .open = opener(device),
        .min_backoff = 100 * x::telem::MILLISECOND,
        .now = clock.func(),
    });
    conn.lock().fail(GONE);
    clock.advance(100 * x::telem::MILLISECOND);
    conn.lock().fail(GONE);
    clock.advance(199 * x::telem::MILLISECOND);
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), UNREACHABLE_ERROR);
    clock.advance(x::telem::MILLISECOND);
    ASSERT_NIL_P(conn.lock().transport());
    conn.lock().fail(GONE);
    clock.advance(100 * x::telem::MILLISECOND);
    ASSERT_NIL_P(conn.lock().transport());
    EXPECT_EQ(device->count(&Device::attempts), 4);
}

/// @brief it should return the time at which the next open may run.
TEST(Connection, ReportRetryTime) {
    const auto device = std::make_shared<Device>();
    device->failures = {GONE};
    ManualClock clock;
    Connection conn({.open = opener(device), .now = clock.func()});
    ASSERT_OCCURRED_AS_P(conn.lock().transport(), UNREACHABLE_ERROR);
    EXPECT_EQ(conn.retry_at(), clock.time + (250 * x::telem::MILLISECOND).chrono());
}

/// @brief it should make every task on a device wait out one shared backoff.
TEST(Connection, ShareBackoffAmongTasks) {
    const auto device = std::make_shared<Device>();
    device->failures = {GONE};
    ManualClock clock;
    Connection conn({.open = opener(device), .now = clock.func()});
    std::vector<std::thread> tasks;
    tasks.reserve(4);
    for (int i = 0; i < 4; i++)
        tasks.emplace_back([&] {
            for (int j = 0; j < 10; j++)
                ASSERT_OCCURRED_AS_P(conn.lock().transport(), UNREACHABLE_ERROR);
        });
    for (auto &t: tasks)
        t.join();
    EXPECT_EQ(device->count(&Device::attempts), 1);
}

/// @brief it should open the transport without blocking the task whose turn it is.
TEST(Connection, OpenWithoutBlockingTurn) {
    const auto device = std::make_shared<Device>();
    std::mutex mu;
    std::condition_variable cv;
    int calls = 0;
    bool opening = false;
    bool released = false;
    const auto open = opener(device);
    Connection conn({
        .open =
            [&] {
                std::unique_lock lock(mu);
                if (calls++ > 0) {
                    opening = true;
                    cv.notify_all();
                    cv.wait(lock, [&] { return released; });
                }
                lock.unlock();
                return open();
            },
        .min_backoff = x::telem::TimeSpan::ZERO(),
    });
    std::thread task;
    {
        auto guard = conn.lock();
        ASSERT_NIL_P(guard.transport());
        guard.fail(GONE);
        task = std::thread([&] { ASSERT_NIL_P(conn.lock().transport()); });
        std::unique_lock lock(mu);
        cv.wait(lock, [&] { return opening; });
        lock.unlock();
        ASSERT_OCCURRED_AS_P(guard.transport(), UNREACHABLE_ERROR);
    }
    {
        std::lock_guard lock(mu);
        released = true;
    }
    cv.notify_all();
    task.join();
    EXPECT_EQ(device->count(&Device::opens), 2);
}

/// @brief it should give one task a turn at a time.
TEST(Connection, GiveOneTurnAtATime) {
    Connection conn({.open = opener(std::make_shared<Device>())});
    std::atomic taken = false;
    std::thread task;
    {
        auto guard = conn.lock();
        task = std::thread([&] {
            auto other = conn.lock();
            taken = true;
        });
        std::this_thread::sleep_for(std::chrono::milliseconds(50));
        EXPECT_FALSE(taken);
    }
    task.join();
    EXPECT_TRUE(taken);
}

/// @brief it should give turns in the order that tasks ask, so a task that locks again
/// at once waits behind a task that asked first.
TEST(Connection, GiveTurnsInOrder) {
    const auto device = std::make_shared<Device>();
    std::mutex mu;
    std::condition_variable cv;
    int calls = 0;
    bool asked = false;
    const auto open = opener(device);
    Connection conn({
        .open =
            [&] {
                std::lock_guard lock(mu);
                if (calls++ == 1) {
                    asked = true;
                    cv.notify_all();
                }
                return open();
            },
        .min_backoff = x::telem::TimeSpan::ZERO(),
    });
    std::vector<std::string> order;
    std::thread task;
    {
        auto guard = conn.lock();
        guard.fail(GONE);
        task = std::thread([&] {
            auto other = conn.lock();
            order.emplace_back("task");
        });
        std::unique_lock lock(mu);
        cv.wait(lock, [&] { return asked; });
    }
    {
        auto again = conn.lock();
        order.emplace_back("again");
    }
    task.join();
    EXPECT_EQ(order, (std::vector<std::string>{"task", "again"}));
}
}
