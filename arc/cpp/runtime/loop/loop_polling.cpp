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

#include "glog/logging.h"

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

bool has_rt_scheduling() {
    return false;
}

class PollingLoop final : public Loop {
public:
    explicit PollingLoop(const Config &config): config_(config) {
        if (this->config_.rt_priority > 0) {
            LOG(WARNING) << "[loop] RT priority not supported in polling mode";
        }
        if (this->config_.cpu_affinity >= 0) {
            LOG(WARNING) << "[loop] CPU affinity not supported in polling mode";
        }
        if (this->config_.lock_memory) {
            LOG(WARNING) << "[loop] Memory locking not supported in polling mode";
        }

        if (this->config_.mode == ExecutionMode::RT_EVENT ||
            this->config_.mode == ExecutionMode::EVENT_DRIVEN ||
            this->config_.mode == ExecutionMode::HYBRID) {
            LOG(INFO) << "[loop] Falling back to HIGH_RATE mode for "
                      << "unsupported execution mode in polling implementation";
        }
    }

    ~PollingLoop() override { this->timer_.reset(); }

    WakeReason wait(breaker::Breaker &breaker) override {
        if (!this->started_) return WakeReason::Shutdown;

        if (this->config_.interval.nanoseconds() > 0 && this->timer_) {
            const auto now = std::chrono::steady_clock::now();
            const auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(
                                     now - this->last_tick_
            )
                                     .count();

            if (elapsed < this->config_.interval.nanoseconds()) {
                const int64_t remaining_ns = this->config_.interval.nanoseconds() -
                                             elapsed;

                switch (this->config_.mode) {
                    case ExecutionMode::BUSY_WAIT:
                        this->busy_wait(remaining_ns, breaker);
                        break;
                    case ExecutionMode::AUTO:
                    case ExecutionMode::HIGH_RATE:
                    case ExecutionMode::RT_EVENT:
                    case ExecutionMode::EVENT_DRIVEN:
                    case ExecutionMode::HYBRID:
                        this->timer_->wait(breaker);
                        break;
                }

                this->last_tick_ = std::chrono::steady_clock::now();
            } else {
                this->last_tick_ = now;
            }
        } else {
            if (this->config_.mode == ExecutionMode::BUSY_WAIT) {
                std::this_thread::sleep_for(telem::MICROSECOND.chrono());
            } else {
                std::this_thread::sleep_for(timing::HIGH_RATE_POLL_INTERVAL.chrono());
            }
        }
        return WakeReason::Timer;
    }

    xerrors::Error start() override {
        if (this->started_) return xerrors::NIL;

        if (this->config_.interval.nanoseconds() > 0) {
            this->timer_ = std::make_unique<::loop::Timer>(this->config_.interval);
        }

        this->last_tick_ = std::chrono::steady_clock::now();
        this->started_ = true;

        return xerrors::NIL;
    }

    void wake() override {
        // Polling loop doesn't block on OS primitives, so wake() is a no-op.
        // The breaker.running() check in the caller will handle shutdown.
    }

    bool watch(notify::Notifier &notifier) override {
        static bool warned = false;
        if (!warned) {
            LOG(WARNING) << "[loop] watch() not supported in polling mode; "
                         << "external notifiers will not wake wait()";
            warned = true;
        }
        (void) notifier;
        return false;
    }

private:
    void busy_wait(uint64_t duration_ns, breaker::Breaker &breaker) {
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

    Config config_;
    std::unique_ptr<::loop::Timer> timer_;
    std::chrono::steady_clock::time_point last_tick_;
    bool started_ = false;
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<PollingLoop>(cfg);
    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), xerrors::NIL};
}

}
