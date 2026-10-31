// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <atomic>
#include <chrono>
#include <condition_variable>
#include <future>
#include <memory>
#include <mutex>
#include <string>
#include <thread>
#include <utility>
#include <vector>

#include "gtest/gtest.h"

#include "client/cpp/status/status.h"
#include "freighter/cpp/freighter.h"
#include "x/cpp/defer/defer.h"
#include "x/cpp/testutil/testutil.h"
#include "x/cpp/uuid/uuid.h"

#include "driver/task/status_cache.h"

namespace driver::task {
/// @brief a status set transport that holds each write that carries one status key
/// until released, and echoes every write. The first passes such writes are not held.
class HoldingSetClient final : public synnax::status::SetClient {
    const std::string held_key;
    size_t passes;
    std::mutex mu;
    std::condition_variable cv;
    bool released = false;
    x::errors::Error held_err = x::errors::NIL;
    std::vector<std::vector<std::string>> requests;

public:
    std::atomic<bool> holding = false;

    explicit HoldingSetClient(std::string held_key, const size_t passes = 0):
        held_key(std::move(held_key)), passes(passes) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::status::SetResponse, x::errors::Error>
    send(const std::string &, grpc::status::SetRequest &req) override {
        std::unique_lock lock{this->mu};
        auto &keys = this->requests.emplace_back();
        for (const auto &status: req.statuses())
            keys.push_back(status.key());
        const bool held = std::ranges::find(keys, this->held_key) != keys.end();
        if (held && this->passes > 0)
            this->passes--;
        else if (held) {
            this->holding = true;
            this->cv.wait(lock, [this] { return this->released; });
            if (const auto err = std::exchange(this->held_err, x::errors::NIL); err)
                return {{}, err};
        }
        grpc::status::SetResponse res;
        *res.mutable_statuses() = req.statuses();
        return {res, x::errors::NIL};
    }

    /// @brief returns the status keys of each request, in order.
    std::vector<std::vector<std::string>> sent() {
        std::lock_guard lock{this->mu};
        return this->requests;
    }

    /// @brief lets held writes go on. The first one fails with err.
    void release(const x::errors::Error &err = x::errors::NIL) {
        {
            std::lock_guard lock{this->mu};
            this->released = true;
            this->held_err = err;
        }
        this->cv.notify_all();
    }
};

/// @brief it should write the status of a task while the write of another task's status
/// hangs.
TEST(StatusCacheTest, WriteDoesNotWaitOnAnotherTask) {
    const auto held = synnax::task::Task{.key = x::uuid::create(), .name = "held"};
    const auto free = synnax::task::Task{.key = x::uuid::create(), .name = "free"};
    const auto transport = std::make_shared<HoldingSetClient>(
        synnax::task::status_key(held)
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status held_status{
        .key = synnax::task::status_key(held),
        .variant = synnax::status::VARIANT_SUCCESS,
        .details = {.task = held.key},
    };
    synnax::task::Status free_status{
        .key = synnax::task::status_key(free),
        .variant = synnax::status::VARIANT_SUCCESS,
        .details = {.task = free.key},
    };
    // Declared before the guard, so the guard releases the hung write before the future
    // waits for its task.
    std::future<void> written;
    std::thread hung([&] { ASSERT_NIL(cache.set(held_status)); });
    x::defer::defer release([&] {
        transport->release();
        hung.join();
    });
    ASSERT_EVENTUALLY_TRUE(transport->holding.load());
    written = std::async(std::launch::async, [&] {
        ASSERT_NIL(cache.set(free_status));
    });
    ASSERT_EQ(written.wait_for(std::chrono::seconds(2)), std::future_status::ready);
}

/// @brief it should wait for a write of a task's own status in progress, then send the
/// status again, so a write that failed is repaired.
TEST(StatusCacheTest, ResendRepairsAWriteInProgress) {
    const auto held = synnax::task::Task{.key = x::uuid::create(), .name = "held"};
    const auto free = synnax::task::Task{.key = x::uuid::create(), .name = "free"};
    const auto transport = std::make_shared<HoldingSetClient>(
        synnax::task::status_key(held)
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status free_status{
        .key = synnax::task::status_key(free),
        .variant = synnax::status::VARIANT_SUCCESS,
        .details = {.task = free.key},
    };
    ASSERT_NIL(cache.set(free_status));
    synnax::task::Status held_status{
        .key = synnax::task::status_key(held),
        .variant = synnax::status::VARIANT_SUCCESS,
        .details = {.task = held.key},
    };
    // Declared before the guard, so the guard releases the hung write before the future
    // waits for the re-send.
    std::future<x::errors::Error> resent;
    std::thread hung([&] {
        ASSERT_OCCURRED_AS(cache.set(held_status), freighter::UNREACHABLE);
    });
    x::defer::defer release([&] {
        transport->release();
        hung.join();
    });
    ASSERT_EVENTUALLY_TRUE(transport->holding.load());
    resent = std::async(std::launch::async, [&] { return cache.resend(); });
    ASSERT_EQ(
        resent.wait_for(std::chrono::milliseconds(200)),
        std::future_status::timeout
    );
    transport->release(freighter::UNREACHABLE);
    ASSERT_EQ(resent.wait_for(std::chrono::seconds(2)), std::future_status::ready);
    ASSERT_NIL(resent.get());
    EXPECT_EQ(
        std::ranges::count(
            transport->sent(),
            std::vector{synnax::task::status_key(held)}
        ),
        2
    );
}

/// @brief it should write the status of a task while the re-send of another task's
/// status hangs.
TEST(StatusCacheTest, WriteDoesNotWaitOnResendOfAnotherTask) {
    const auto held = synnax::task::Task{.key = x::uuid::create(), .name = "held"};
    const auto free = synnax::task::Task{.key = x::uuid::create(), .name = "free"};
    const auto transport = std::make_shared<HoldingSetClient>(
        synnax::task::status_key(held),
        1
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status held_status{
        .key = synnax::task::status_key(held),
        .variant = synnax::status::VARIANT_SUCCESS,
        .details = {.task = held.key},
    };
    ASSERT_NIL(cache.set(held_status));
    synnax::task::Status free_status{
        .key = synnax::task::status_key(free),
        .variant = synnax::status::VARIANT_SUCCESS,
        .details = {.task = free.key},
    };
    ASSERT_NIL(cache.set(free_status));
    // Declared before the guard, so the guard releases the hung re-send before the
    // futures wait for their tasks.
    std::future<x::errors::Error> resent;
    std::future<void> written;
    x::defer::defer release([&] { transport->release(); });
    resent = std::async(std::launch::async, [&] { return cache.resend(); });
    ASSERT_EVENTUALLY_TRUE(transport->holding.load());
    written = std::async(std::launch::async, [&] {
        ASSERT_NIL(cache.set(free_status));
    });
    ASSERT_EQ(written.wait_for(std::chrono::seconds(2)), std::future_status::ready);
    transport->release();
    ASSERT_NIL(resent.get());
}

/// @brief it should neither write nor keep a status of a task that left this Driver.
TEST(StatusCacheTest, DropDiscardsLaterStatuses) {
    const auto task = synnax::task::Task{.key = x::uuid::create(), .name = "released"};
    const auto transport = std::make_shared<HoldingSetClient>("");
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status late{
        .key = synnax::task::status_key(task),
        .variant = synnax::status::VARIANT_ERROR,
        .message = "Failed to complete task operation",
        .details = {.task = task.key},
    };
    cache.drop(task.key);
    ASSERT_NIL(cache.set(late));
    ASSERT_NIL(cache.resend());
    EXPECT_TRUE(transport->sent().empty());
}

/// @brief it should write and keep the statuses of a task claimed again after a drop.
TEST(StatusCacheTest, ClaimWritesStatusesAgain) {
    const auto task = synnax::task::Task{.key = x::uuid::create(), .name = "returned"};
    const auto transport = std::make_shared<HoldingSetClient>("");
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status configured{
        .key = synnax::task::status_key(task),
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Task configured",
        .details = {.task = task.key},
    };
    cache.drop(task.key);
    cache.claim(task.key);
    ASSERT_NIL(cache.set(configured));
    ASSERT_NIL(cache.resend());
    const auto key = synnax::task::status_key(task);
    EXPECT_EQ(transport->sent(), (std::vector<std::vector<std::string>>{{key}, {key}}));
}
}
