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
#include <mutex>
#include <thread>
#include <unordered_map>
#include <vector>

#include "glog/logging.h"
#include <windows.h>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

class WindowsLoop final : public Loop {
    static constexpr DWORD MAX_HANDLES = MAXIMUM_WAIT_OBJECTS;

public:
    explicit WindowsLoop(const Config &config): config_(config) {
        if (this->config_.rt_priority > 0 && this->config_.rt_priority > 31) {
            LOG(WARNING) << "[loop] Windows priority range is 0-31, clamping";
        }

        if (this->config_.lock_memory) {
            LOG(WARNING) << "[loop] Memory locking on Windows requires "
                         << "VirtualLock API (not implemented)";
        }
    }

    ~WindowsLoop() override { this->stop(); }

    void notify_data() override {
        if (!this->running_ || this->data_event_ == NULL) return;

        if (!SetEvent(this->data_event_)) {
            LOG(ERROR) << "[loop] Failed to set data event: " << GetLastError();
        }

        this->data_available_.store(true, std::memory_order_release);
    }

    void wait(x::breaker::Breaker &breaker) override {
        if (!this->running_) return;

        switch (this->config_.mode) {
            case ExecutionMode::BUSY_WAIT:
                this->busy_wait(breaker);
                break;
            case ExecutionMode::HIGH_RATE:
                this->high_rate_wait(breaker);
                break;
            case ExecutionMode::RT_EVENT:
            case ExecutionMode::EVENT_DRIVEN:
                this->event_driven_wait(
                    breaker,
                    this->config_.mode == ExecutionMode::EVENT_DRIVEN
                );
                break;
            case ExecutionMode::HYBRID:
                this->hybrid_wait(breaker);
                break;
        }
    }

    x::errors::Error start() override {
        if (this->running_) return x::errors::NIL;

        this->data_event_ = CreateEvent(NULL, TRUE, FALSE, NULL);
        if (this->data_event_ == NULL) {
            return x::errors::Error(
                "Failed to create data event: " + std::to_string(GetLastError())
            );
        }

        if (this->config_.interval.nanoseconds() > 0) {
            this->timer_event_ = CreateWaitableTimer(NULL, FALSE, NULL);
            if (this->timer_event_ == NULL) {
                CloseHandle(this->data_event_);
                return x::errors::Error(
                    "Failed to create waitable timer: " + std::to_string(GetLastError())
                );
            }

            LARGE_INTEGER due_time;
            const int64_t interval_100ns = this->config_.interval.nanoseconds() / 100;
            due_time.QuadPart = -interval_100ns;

            const LONG period_ms = static_cast<LONG>(
                this->config_.interval.nanoseconds() / 1'000'000
            );

            if (!SetWaitableTimer(
                    this->timer_event_,
                    &due_time,
                    period_ms,
                    NULL,
                    NULL,
                    FALSE
                )) {
                CloseHandle(this->timer_event_);
                CloseHandle(this->data_event_);
                return x::errors::Error(
                    "Failed to set waitable timer: " + std::to_string(GetLastError())
                );
            }

            this->timer_enabled_ = true;
        }

        if (this->config_.mode == ExecutionMode::HIGH_RATE ||
            this->config_.mode == ExecutionMode::HYBRID) {
            if (this->config_.interval.nanoseconds() > 0) {
                this->timer_ = std::make_unique<x::loop::Timer>(this->config_.interval);
            }
        }

        if (this->config_.rt_priority > 0) {
            if (auto err = this->set_thread_priority(this->config_.rt_priority); err) {
                LOG(WARNING) << "[loop] Failed to set thread priority: "
                             << err.message();
            }
        }

        if (this->config_.cpu_affinity >= 0) {
            if (auto err = this->set_cpu_affinity(this->config_.cpu_affinity); err) {
                LOG(WARNING) << "[loop] Failed to set CPU affinity: " << err.message();
            }
        }

        this->running_ = true;
        this->data_available_.store(false, std::memory_order_release);

        return x::errors::NIL;
    }

    void stop() override {
        if (!this->running_) return;

        this->running_ = false;
        this->timer_.reset();

        if (this->timer_event_ != NULL) {
            CancelWaitableTimer(this->timer_event_);
            CloseHandle(this->timer_event_);
            this->timer_event_ = NULL;
        }

        if (this->data_event_ != NULL) {
            CloseHandle(this->data_event_);
            this->data_event_ = NULL;
        }

        this->timer_enabled_ = false;
    }

    bool watch(x::notify::Notifier &notifier) override {
        static bool warned = false;
        if (!warned) {
            LOG(WARNING) << "[loop] watch() not supported on Windows; "
                         << "external notifiers will not wake wait()";
            warned = true;
        }
        (void) notifier;
        return false;
    }

private:
    void busy_wait(x::breaker::Breaker &breaker) {
        HANDLE handles[2];
        DWORD count = 1;
        handles[0] = this->data_event_;

        if (this->timer_enabled_) {
            handles[1] = this->timer_event_;
            count = 2;
        }

        while (!!breaker.running()) {
            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);

            if (result < WAIT_OBJECT_0 + count) {
                ResetEvent(this->data_event_);
                this->data_available_.store(false, std::memory_order_release);
                return;
            }

            if (this->data_available_.load(std::memory_order_acquire)) {
                ResetEvent(this->data_event_);
                this->data_available_.store(false, std::memory_order_release);
                return;
            }

            if (result == WAIT_FAILED) {
                LOG(ERROR) << "[loop] WaitForMultipleObjects failed: "
                           << GetLastError();
                return;
            }
        }
    }

    void high_rate_wait(x::breaker::Breaker &breaker) {
        if (this->timer_) {
            this->timer_->wait(breaker);
        } else {
            std::this_thread::sleep_for(timing::HIGH_RATE_POLL_INTERVAL.chrono());
        }

        ResetEvent(this->data_event_);
        this->data_available_.store(false, std::memory_order_release);
    }

    void event_driven_wait(x::breaker::Breaker &breaker, bool blocking) {
        HANDLE handles[2];
        DWORD count = 1;
        handles[0] = this->data_event_;

        if (this->timer_enabled_) {
            handles[1] = this->timer_event_;
            count = 2;
        }

        const DWORD timeout_ms = blocking
                                   ? INFINITE
                                   : static_cast<DWORD>(
                                         timing::HYBRID_BLOCK_TIMEOUT.milliseconds()
                                     );

        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout_ms);

        if (result < WAIT_OBJECT_0 + count) {
            ResetEvent(this->data_event_);
        } else if (result == WAIT_FAILED) {
            LOG(ERROR) << "[loop] WaitForMultipleObjects failed: " << GetLastError();
        }

        this->data_available_.store(false, std::memory_order_release);
    }

    void hybrid_wait(x::breaker::Breaker &breaker) {
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = std::chrono::nanoseconds(
            this->config_.spin_duration.nanoseconds()
        );

        HANDLE handles[2];
        DWORD count = 1;
        handles[0] = this->data_event_;

        if (this->timer_enabled_) {
            handles[1] = this->timer_event_;
            count = 2;
        }

        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;

            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);

            if (result < WAIT_OBJECT_0 + count) {
                ResetEvent(this->data_event_);
                this->data_available_.store(false, std::memory_order_release);
                return;
            }

            if (this->data_available_.load(std::memory_order_acquire)) {
                ResetEvent(this->data_event_);
                this->data_available_.store(false, std::memory_order_release);
                return;
            }
        }

        const DWORD timeout_ms = static_cast<DWORD>(
            timing::HYBRID_BLOCK_TIMEOUT.milliseconds()
        );
        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout_ms);
        if (result < WAIT_OBJECT_0 + count) { ResetEvent(this->data_event_); }

        this->data_available_.store(false, std::memory_order_release);
    }

    x::errors::Error set_thread_priority(int priority) {
        int win_priority;
        if (priority >= 90) {
            win_priority = THREAD_PRIORITY_TIME_CRITICAL;
        } else if (priority >= 70) {
            win_priority = THREAD_PRIORITY_HIGHEST;
        } else if (priority >= 50) {
            win_priority = THREAD_PRIORITY_ABOVE_NORMAL;
        } else {
            win_priority = THREAD_PRIORITY_NORMAL;
        }

        if (!SetThreadPriority(GetCurrentThread(), win_priority)) {
            return x::errors::Error(
                "Failed to set thread priority: " + std::to_string(GetLastError())
            );
        }

        return x::errors::NIL;
    }

    x::errors::Error set_cpu_affinity(int cpu) {
        const DWORD_PTR mask = static_cast<DWORD_PTR>(1) << cpu;

        if (!SetThreadAffinityMask(GetCurrentThread(), mask)) {
            return x::errors::Error(
                "Failed to set thread affinity: " + std::to_string(GetLastError())
            );
        }

        return x::errors::NIL;
    }

    Config config_;
    HANDLE data_event_ = NULL;
    HANDLE timer_event_ = NULL;
    bool timer_enabled_ = false;
    std::unique_ptr<::x::loop::Timer> timer_;
    std::atomic<bool> data_available_{false};
    std::atomic<bool> running_{false};
};

std::pair<std::unique_ptr<Loop>, x::errors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<WindowsLoop>(cfg);
    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), x::errors::NIL};
}

}
