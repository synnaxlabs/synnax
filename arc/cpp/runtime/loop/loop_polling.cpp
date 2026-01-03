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
#include <thread>

#include "glog/logging.h"

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {
/// @brief Polling-based fallback implementation of Loop.
///
/// This implementation doesn't use any platform-specific event primitives.
/// It uses simple polling with optional sleep intervals. Suitable for platforms
/// without epoll/kqueue/IOCP support or as a simple fallback.
class PollingLoop final : public Loop {
public:
    explicit PollingLoop(const Config &config): config_(config) {
        // Validate configuration
        if (config_.rt_priority > 0) {
            LOG(WARNING) << "[loop] RT priority not supported in polling mode";
        }
        if (config_.cpu_affinity >= 0) {
            LOG(WARNING) << "[loop] CPU affinity not supported in polling mode";
        }
        if (config_.lock_memory) {
            LOG(WARNING) << "[loop] Memory locking not supported in polling mode";
        }

        // All modes fall back to polling in this implementation
        if (config_.mode == ExecutionMode::RT_EVENT ||
            config_.mode == ExecutionMode::EVENT_DRIVEN ||
            config_.mode == ExecutionMode::HYBRID) {
            LOG(INFO) << "[loop] Falling back to HIGH_RATE mode for "
                      << "unsupported execution mode in polling implementation";
        }
    }

    ~PollingLoop() override { stop(); }

    void notify_data() override {
        // In polling mode, we don't have event-driven notifications.
        // The wait() loop will pick up data on the next poll cycle.
        // Set a flag to minimize latency in HYBRID mode if needed.
        data_available_.store(true, std::memory_order_release);
    }

    void wait(breaker::Breaker &breaker) override {
        if (!running_) return;

        // Check if we need to wait for timer interval
        if (config_.interval.nanoseconds() > 0 && timer_) {
            const auto now = std::chrono::steady_clock::now();
            const auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(
                                     now - last_tick_
            )
                                     .count();

            if (elapsed < config_.interval.nanoseconds()) {
                // Timer hasn't expired yet
                const int64_t remaining_ns = config_.interval.nanoseconds() - elapsed;

                // Choose wait strategy based on mode
                switch (config_.mode) {
                    case ExecutionMode::BUSY_WAIT:
                        // Busy wait - no sleep
                        busy_wait(remaining_ns, breaker);
                        break;

                    case ExecutionMode::HIGH_RATE:
                    case ExecutionMode::RT_EVENT:
                    case ExecutionMode::EVENT_DRIVEN:
                    case ExecutionMode::HYBRID:
                    default:
                        // Use precise timer for sleeping
                        timer_->wait(breaker);
                        break;
                }

                last_tick_ = std::chrono::steady_clock::now();
            } else {
                // Timer has already expired
                last_tick_ = now;
            }
        } else {
            // No timer configured - use short sleep to avoid busy loop
            const uint64_t poll_interval_us = config_.mode == ExecutionMode::BUSY_WAIT
                                                ? 1
                                                : 100;
            std::this_thread::sleep_for(std::chrono::microseconds(poll_interval_us));
        }

        // Clear the data available flag after waiting
        data_available_.store(false, std::memory_order_release);
    }

    xerrors::Error start() override {
        if (running_) {
            return xerrors::NIL; // Already started
        }

        // Initialize timer if interval is configured
        if (config_.interval.nanoseconds() > 0) {
            timer_ = std::make_unique<::loop::Timer>(config_.interval);
        }

        last_tick_ = std::chrono::steady_clock::now();
        running_ = true;
        data_available_.store(false, std::memory_order_release);

        return xerrors::NIL;
    }

    void stop() override {
        running_ = false;
        timer_.reset();
    }

private:
    /// @brief Busy wait for the specified duration.
    void busy_wait(uint64_t duration_ns, breaker::Breaker &breaker) {
        const auto start = std::chrono::steady_clock::now();
        while (!!breaker.running()) {
            const auto now = std::chrono::steady_clock::now();
            const auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(
                                     now - start
            )
                                     .count();
            if (elapsed >= static_cast<int64_t>(duration_ns)) { break; }
            // Optionally yield to prevent complete CPU starvation
            // std::this_thread::yield();
        }
    }

    Config config_;
    std::unique_ptr<::loop::Timer> timer_;
    std::chrono::steady_clock::time_point last_tick_;
    std::atomic<bool> data_available_{false};
    bool running_ = false;
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<PollingLoop>(cfg);
    if (auto err = loop->start(); err) { return {nullptr, err}; }
    return {std::move(loop), xerrors::NIL};
}
}
