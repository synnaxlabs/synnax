// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <thread>

#include "absl/log/log.h"

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

bool has_rt_scheduling() {
    return false;
}

class PollingLoop final : public Loop {
public:
    explicit PollingLoop(const Config &config): config(config) {
        if (this->config.rt_priority > 0) {
            LOG(WARNING) << "[arc.loop] RT priority not supported in polling mode";
        }
        if (this->config.cpu_affinity >= 0) {
            LOG(WARNING) << "[arc.loop] CPU affinity not supported in polling mode";
        }
        if (this->config.memory_locked) {
            LOG(WARNING) << "[arc.loop] Memory locking not supported in polling mode";
        }

        if (this->config.mode == ExecutionMode::RT_EVENT ||
            this->config.mode == ExecutionMode::EVENT_DRIVEN ||
            this->config.mode == ExecutionMode::HYBRID) {
            LOG(INFO) << "[arc.loop] Falling back to HIGH_RATE mode for "
                      << "unsupported execution mode in polling implementation";
        }
    }

    ~PollingLoop() override { this->timer.reset(); }

    WakeReason wait(
        x::breaker::Breaker &breaker,
        x::telem::TimeSpan max_timeout = x::telem::TimeSpan(0),
        x::telem::TimeSpan span = x::telem::TimeSpan::max()
    ) override {
        if (!this->started) return WakeReason::Shutdown;

        if (this->config.interval.nanoseconds() > 0 && this->timer) {
            const auto now = std::chrono::steady_clock::now();
            const auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(
                                     now - this->last_tick
            )
                                     .count();

            auto interval_ns = this->config.interval.nanoseconds();
            if (max_timeout.nanoseconds() > 0)
                interval_ns = std::min(interval_ns, max_timeout.nanoseconds());

            if (elapsed < interval_ns) {
                const int64_t remaining_ns = interval_ns - elapsed;

                switch (this->config.mode) {
                    case ExecutionMode::BUSY_WAIT:
                        this->busy_wait(remaining_ns, breaker);
                        break;
                    case ExecutionMode::AUTO:
                    case ExecutionMode::HIGH_RATE:
                    case ExecutionMode::RT_EVENT:
                    case ExecutionMode::EVENT_DRIVEN:
                    case ExecutionMode::HYBRID:
                        this->timer->wait(breaker);
                        break;
                }

                this->last_tick = std::chrono::steady_clock::now();
            } else {
                this->last_tick = now;
            }
        } else {
            if (max_timeout.nanoseconds() > 0) {
                std::this_thread::sleep_for(max_timeout.chrono());
            } else if (this->config.mode == ExecutionMode::BUSY_WAIT) {
                std::this_thread::sleep_for(x::telem::MICROSECOND.chrono());
            } else {
                std::this_thread::sleep_for(timing::HIGH_RATE_POLL_INTERVAL.chrono());
            }
        }
        return WakeReason::Timer;
    }

    x::errors::Error start() override {
        if (this->started) return x::errors::NIL;

        if (this->config.interval.nanoseconds() > 0) {
            this->timer = std::make_unique<::x::loop::Timer>(this->config.interval);
        }

        this->last_tick = std::chrono::steady_clock::now();
        this->started = true;

        return x::errors::NIL;
    }

    void wake() override {
        // Polling loop doesn't block on OS primitives, so wake() is a no-op.
        // The breaker.running() check in the caller will handle shutdown.
    }

    bool watch(x::notify::Notifier &notifier) override {
        static bool warned = false;
        if (!warned) {
            LOG(WARNING) << "[arc.loop] watch() not supported in polling mode; "
                         << "external notifiers will not wake wait()";
            warned = true;
        }
        (void) notifier;
        return false;
    }

private:
    void busy_wait(uint64_t duration_ns, x::breaker::Breaker &breaker) {
        const auto start = std::chrono::steady_clock::now();
        while (breaker.running()) {
            const auto now = std::chrono::steady_clock::now();
            const auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(
                                     now - start
            )
                                     .count();
            if (elapsed >= static_cast<int64_t>(duration_ns)) break;
        }
    }

    Config config;
    std::unique_ptr<::x::loop::Timer> timer;
    std::chrono::steady_clock::time_point last_tick;
    bool started = false;
};

std::unique_ptr<Loop>
create(const Config &cfg, std::shared_ptr<x::thread::rt::Handle> rt_handle) {
    return std::make_unique<PollingLoop>(cfg);
}
}
