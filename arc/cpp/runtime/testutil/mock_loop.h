// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <chrono>
#include <condition_variable>
#include <mutex>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/notify/notify.h"
#include "x/cpp/errors/errors.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::testutil {
/// @brief Mock loop implementation for testing runtime lifecycle.
class MockLoop final : public loop::Loop {
public:
    /// @brief Count of start() invocations.
    std::atomic<int> start_count{0};
    /// @brief Count of wake() invocations.
    std::atomic<int> wake_count{0};
    /// @brief Count of wait() invocations.
    std::atomic<int> wait_count{0};
    /// @brief Count of watch() invocations.
    std::atomic<int> watch_count{0};

    x::errors::Error start() override {
        start_count++;
        std::lock_guard lock(mu);
        should_block = true;
        return x::errors::NIL;
    }

    void wait(x::breaker::Breaker &breaker) override {
        wait_count++;
        std::unique_lock lock(mu);
        cv.wait_for(lock, std::chrono::milliseconds(10), [&] {
            return !should_block || !breaker.running();
        });
    }

    void wake() override {
        wake_count++;
        {
            std::lock_guard lock(mu);
            should_block = false;
        }
        cv.notify_all();
    }

    bool watch(notify::Notifier &notifier) override {
        watch_count++;
        watched_notifiers.push_back(&notifier);
        return true;
    }

    /// @brief List of notifiers that have been watched.
    std::vector<notify::Notifier *> watched_notifiers;

private:
    std::condition_variable cv;
    std::mutex mu;
    bool should_block{true};
};
}
