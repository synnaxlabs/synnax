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

    ~PollingLoop() override { this->stop(); }

    void notify_data() override {
        this->data_available_.store(true, std::memory_order_release);
    }

    void wait(x::breaker::Breaker &breaker) override {
        if (!this->running_) return;

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
                    case ExecutionMode::HIGH_RATE:
                    case ExecutionMode::RT_EVENT:
                    case ExecutionMode::EVENT_DRIVEN:
                    case ExecutionMode::HYBRID:
                    default:
                        this->timer_->wait(breaker);
                        break;
                }

                this->last_tick_ = std::chrono::steady_clock::now();
            } else {
                this->last_tick_ = now;
            }
        } else {
            if (this->config_.mode == ExecutionMode::BUSY_WAIT) {
                std::this_thread::sleep_for(std::chrono::microseconds(1));
            } else {
                std::this_thread::sleep_for(timing::HIGH_RATE_POLL_INTERVAL.chrono());
            }
        }

        this->data_available_.store(false, std::memory_order_release);
    }

    x::errors::Error start() override {
        if (this->running_) return x::errors::NIL;

        if (this->config_.interval.nanoseconds() > 0) {
            this->timer_ = std::make_unique<x::loop::Timer>(this->config_.interval);
        }

        this->last_tick_ = std::chrono::steady_clock::now();
        this->running_ = true;
        this->data_available_.store(false, std::memory_order_release);

        return x::errors::NIL;
    }

    void stop() override {
        this->running_ = false;
        this->timer_.reset();
    }

    bool watch(x::notify::Notifier &notifier) override {
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
    void busy_wait(uint64_t duration_ns, x::breaker::Breaker &breaker) {
        const auto start = std::chrono::steady_clock::now();
        while (!!breaker.running()) {
            const auto now = std::chrono::steady_clock::now();
            const auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(
                                     now - start
            )
                                     .count();
            if (elapsed >= static_cast<int64_t>(duration_ns)) break;
        }
    }

    Config config_;
    std::unique_ptr<::x::loop::Timer> timer_;
    std::chrono::steady_clock::time_point last_tick_;
    std::atomic<bool> data_available_{false};
    std::atomic<bool> running_{false};
};

std::pair<std::unique_ptr<Loop>, x::errors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<PollingLoop>(cfg);
    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), x::errors::NIL};
}

}
