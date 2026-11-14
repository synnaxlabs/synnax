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
#include <optional>
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
/// @brief a status set transport that holds each write that carries one status owner
/// until released, and echoes every write. The first passes such writes are not held.
class HoldingSetClient final : public synnax::status::SetClient {
    const synnax::resource::ID held_owner;
    size_t passes;
    std::mutex mu;
    std::condition_variable cv;
    bool released = false;
    x::errors::Error held_err = x::errors::NIL;
    std::vector<std::vector<synnax::task::Status>> requests;

public:
    std::atomic<bool> holding = false;

    explicit HoldingSetClient(synnax::resource::ID held_owner, const size_t passes = 0):
        held_owner(std::move(held_owner)), passes(passes) {}

    void use(std::shared_ptr<freighter::Middleware>) override {}

    std::pair<grpc::status::SetResponse, x::errors::Error>
    send(const std::string &, grpc::status::SetRequest &req) override {
        std::unique_lock lock{this->mu};
        auto &statuses = this->requests.emplace_back();
        bool held = false;
        for (const auto &pb: req.statuses()) {
            auto [status, err] = synnax::task::Status::from_proto(pb);
            if (err) return {{}, err};
            held = held || status.owner == this->held_owner;
            statuses.push_back(std::move(status));
        }
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

    /// @brief returns the statuses of each request, in order.
    std::vector<std::vector<synnax::task::Status>> sent() {
        std::lock_guard lock{this->mu};
        return this->requests;
    }

    /// @brief returns the number of requests that carry exactly one status, owned by
    /// owner.
    size_t count(const synnax::resource::ID &owner) {
        std::lock_guard lock{this->mu};
        return std::ranges::count_if(this->requests, [&](const auto &statuses) {
            return statuses.size() == 1 && statuses[0].owner == owner;
        });
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
        synnax::task::resource_id(held.key)
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status held_status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .owner = synnax::task::resource_id(held.key),
    };
    synnax::task::Status free_status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .owner = synnax::task::resource_id(free.key),
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
        synnax::task::resource_id(held.key)
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status free_status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .owner = synnax::task::resource_id(free.key),
    };
    ASSERT_NIL(cache.set(free_status));
    synnax::task::Status held_status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .owner = synnax::task::resource_id(held.key),
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
    EXPECT_EQ(transport->count(synnax::task::resource_id(held.key)), 2);
}

/// @brief it should report a failed write as stale until a re-send starts.
TEST(StatusCacheTest, FailedWriteIsStaleUntilResend) {
    const auto task = synnax::task::Task{.key = x::uuid::create(), .name = "lost"};
    const auto transport = std::make_shared<HoldingSetClient>(
        synnax::task::resource_id(task.key)
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status status{
        .variant = synnax::status::VARIANT_WARNING,
        .owner = synnax::task::resource_id(task.key),
    };
    EXPECT_FALSE(cache.stale());
    transport->release(freighter::UNREACHABLE);
    ASSERT_OCCURRED_AS(cache.set(status), freighter::UNREACHABLE);
    EXPECT_TRUE(cache.stale());
    ASSERT_NIL(cache.resend());
    EXPECT_FALSE(cache.stale());
    EXPECT_EQ(transport->sent().size(), 2);
}

/// @brief it should stay stale after a failed re-send, until a re-send succeeds.
TEST(StatusCacheTest, FailedResendStaysStale) {
    const auto task = synnax::task::Task{.key = x::uuid::create(), .name = "lost"};
    const auto transport = std::make_shared<HoldingSetClient>(
        synnax::task::resource_id(task.key)
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status status{
        .variant = synnax::status::VARIANT_WARNING,
        .owner = synnax::task::resource_id(task.key),
    };
    transport->release(freighter::UNREACHABLE);
    ASSERT_OCCURRED_AS(cache.set(status), freighter::UNREACHABLE);
    transport->release(freighter::UNREACHABLE);
    ASSERT_OCCURRED_AS(cache.resend(), freighter::UNREACHABLE);
    EXPECT_TRUE(cache.stale());
    ASSERT_NIL(cache.resend());
    EXPECT_FALSE(cache.stale());
    EXPECT_EQ(transport->sent().size(), 3);
}

/// @brief it should write the status of a task while the re-send of another task's
/// status hangs.
TEST(StatusCacheTest, WriteDoesNotWaitOnResendOfAnotherTask) {
    const auto held = synnax::task::Task{.key = x::uuid::create(), .name = "held"};
    const auto free = synnax::task::Task{.key = x::uuid::create(), .name = "free"};
    const auto transport = std::make_shared<HoldingSetClient>(
        synnax::task::resource_id(held.key),
        1
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status held_status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .owner = synnax::task::resource_id(held.key),
    };
    ASSERT_NIL(cache.set(held_status));
    synnax::task::Status free_status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .owner = synnax::task::resource_id(free.key),
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
    const auto transport = std::make_shared<HoldingSetClient>(synnax::resource::ID{});
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status late{
        .variant = synnax::status::VARIANT_ERROR,
        .message = "Failed to complete task operation",
        .owner = synnax::task::resource_id(task.key),
    };
    cache.drop(task.key);
    ASSERT_NIL(cache.set(late));
    ASSERT_NIL(cache.resend());
    EXPECT_TRUE(transport->sent().empty());
}

/// @brief it should write and keep the statuses of a task claimed again after a drop.
TEST(StatusCacheTest, ClaimWritesStatusesAgain) {
    const auto task = synnax::task::Task{.key = x::uuid::create(), .name = "returned"};
    const auto transport = std::make_shared<HoldingSetClient>(synnax::resource::ID{});
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status configured{
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Task configured",
        .owner = synnax::task::resource_id(task.key),
    };
    cache.drop(task.key);
    cache.claim(task.key);
    ASSERT_NIL(cache.set(configured));
    ASSERT_NIL(cache.resend());
    EXPECT_EQ(transport->sent().size(), 2);
    EXPECT_EQ(transport->count(synnax::task::resource_id(task.key)), 2);
}

/// @brief it should keep one status per owner and send the latest one again.
TEST(StatusCacheTest, ResendSendsLatestStatusOfOwner) {
    const auto task = synnax::task::Task{.key = x::uuid::create(), .name = "task"};
    const auto transport = std::make_shared<HoldingSetClient>(synnax::resource::ID{});
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status first{
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Task configured",
        .owner = synnax::task::resource_id(task.key),
    };
    synnax::task::Status second{
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Started task",
        .owner = synnax::task::resource_id(task.key),
    };
    ASSERT_NIL(cache.set(first));
    ASSERT_NIL(cache.set(second));
    ASSERT_NIL(cache.resend());
    const auto sent = transport->sent();
    ASSERT_EQ(sent.size(), 3);
    ASSERT_EQ(sent[2].size(), 1);
    EXPECT_EQ(sent[2][0].message, "Started task");
    EXPECT_TRUE(sent[2][0].key.empty());
}

/// @brief it should refuse a status that names no owner.
TEST(StatusCacheTest, SetRequiresOwner) {
    const auto transport = std::make_shared<HoldingSetClient>(synnax::resource::ID{});
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status ownerless{.variant = synnax::status::VARIANT_SUCCESS};
    EXPECT_THROW((void) cache.set(ownerless), std::bad_optional_access);
    EXPECT_TRUE(transport->sent().empty());
}

/// @brief it should return the latest status of a task without its command key.
TEST(StatusCacheTest, LatestReturnsTheLatestStatusWithNoCommandKey) {
    const auto task = synnax::task::Task{.key = x::uuid::create(), .name = "failed"};
    const auto transport = std::make_shared<HoldingSetClient>(synnax::resource::ID{});
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status configured{
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Task configured",
        .owner = synnax::task::resource_id(task.key),
    };
    synnax::task::Status failed{
        .variant = synnax::status::VARIANT_ERROR,
        .message = "Failed to configure task",
        .details = {.cmd = "start-cmd"},
        .owner = synnax::task::resource_id(task.key),
    };
    ASSERT_NIL(cache.set(configured));
    ASSERT_NIL(cache.set(failed));
    const auto latest = cache.latest(task.key);
    ASSERT_TRUE(latest.has_value());
    EXPECT_EQ(latest.value().message, "Failed to configure task");
    EXPECT_TRUE(latest.value().details.cmd.empty());
}

/// @brief it should return no status for a task that left this Driver or never had one.
TEST(StatusCacheTest, LatestReturnsNothingForADroppedOrUnknownTask) {
    const auto task = synnax::task::Task{.key = x::uuid::create(), .name = "released"};
    const auto transport = std::make_shared<HoldingSetClient>(synnax::resource::ID{});
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status configured{
        .variant = synnax::status::VARIANT_SUCCESS,
        .message = "Task configured",
        .owner = synnax::task::resource_id(task.key),
    };
    ASSERT_NIL(cache.set(configured));
    cache.drop(task.key);
    EXPECT_FALSE(cache.latest(task.key).has_value());
    EXPECT_FALSE(cache.latest(x::uuid::create()).has_value());
}

/// @brief it should re-send no status after clear, and keep a dropped task dropped.
TEST(StatusCacheTest, ClearForgetsStatusesAndKeepsDrops) {
    const auto kept = synnax::task::Task{.key = x::uuid::create(), .name = "kept"};
    const auto gone = synnax::task::Task{.key = x::uuid::create(), .name = "gone"};
    const auto transport = std::make_shared<HoldingSetClient>(
        synnax::task::resource_id(x::uuid::create())
    );
    StatusCache cache(synnax::status::Client(nullptr, transport, nullptr, nullptr));
    synnax::task::Status status{
        .variant = synnax::status::VARIANT_SUCCESS,
        .owner = synnax::task::resource_id(kept.key),
    };
    ASSERT_NIL(cache.set(status));
    cache.drop(gone.key);
    cache.clear();
    EXPECT_FALSE(cache.latest(kept.key).has_value());
    ASSERT_NIL(cache.resend());
    EXPECT_EQ(transport->sent().size(), 1);
    synnax::task::Status late{
        .variant = synnax::status::VARIANT_SUCCESS,
        .owner = synnax::task::resource_id(gone.key),
    };
    ASSERT_NIL(cache.set(late));
    EXPECT_EQ(transport->sent().size(), 1);
}
}
