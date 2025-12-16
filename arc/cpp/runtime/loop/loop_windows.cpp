// Copyright 2025 Synnax Labs, Inc.
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
#include <windows.h>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {
/// @brief Windows WaitForMultipleObjects-based implementation of Loop.
///
/// Uses Windows-specific primitives for event handling:
/// - CreateWaitableTimer: Periodic timer events
/// - CreateEvent: Manual-reset event for data notifications
/// - WaitForMultipleObjects: Event multiplexing
/// - SetThreadPriority: Thread priority configuration
/// - SetThreadAffinityMask: CPU affinity
///
/// Windows has RT support through high-priority threads and timer resolution.
class WindowsLoop final : public Loop {
public:
    WindowsLoop() = default;

    ~WindowsLoop() override { stop(); }

    xerrors::Error configure(const Config &config) override {
        config_ = config;

        // Windows RT support is available via high-priority threads
        if (config_.rt_priority > 0 && config_.rt_priority > 31) {
            LOG(WARNING) << "[loop] Windows priority range is 0-31, clamping";
        }

        if (config_.lock_memory) {
            LOG(WARNING) << "[loop] Memory locking on Windows requires "
                         << "VirtualLock API (not implemented)";
        }

        return xerrors::NIL;
    }

    void notify_data() override {
        if (!running_ || data_event_ == NULL) return;

        // Set the manual-reset event to wake up WaitForMultipleObjects
        if (!SetEvent(data_event_)) {
            LOG(ERROR) << "[loop] Failed to set data event: " << GetLastError();
        }

        data_available_.store(true, std::memory_order_release);
    }

    void wait(breaker::Breaker &breaker) override {
        if (!running_) return;

        switch (config_.mode) {
            case ExecutionMode::BUSY_WAIT:
                busy_wait(breaker);
                break;

            case ExecutionMode::HIGH_RATE:
                high_rate_wait(breaker);
                break;

            case ExecutionMode::RT_EVENT:
            case ExecutionMode::EVENT_DRIVEN:
                event_driven_wait(breaker, config_.mode == ExecutionMode::EVENT_DRIVEN);
                break;

            case ExecutionMode::HYBRID:
                hybrid_wait(breaker);
                break;
        }
    }

    xerrors::Error start() override {
        if (running_) {
            return xerrors::NIL; // Already started
        }

        // Create manual-reset event for data notifications
        data_event_ = CreateEvent(
            NULL, // Default security
            TRUE, // Manual reset
            FALSE, // Initial state: nonsignaled
            NULL // Unnamed
        );
        if (data_event_ == NULL) {
            return xerrors::Error(
                "Failed to create data event: " + std::to_string(GetLastError())
            );
        }

        // Create waitable timer if interval is configured
        if (config_.interval > 0) {
            timer_event_ = CreateWaitableTimer(
                NULL, // Default security
                FALSE, // Auto-reset
                NULL // Unnamed
            );
            if (timer_event_ == NULL) {
                CloseHandle(data_event_);
                return xerrors::Error(
                    "Failed to create waitable timer: " + std::to_string(GetLastError())
                );
            }

            // Set timer interval
            // LARGE_INTEGER is in 100-nanosecond units, negative = relative time
            LARGE_INTEGER due_time;
            const int64_t interval_100ns = static_cast<int64_t>(config_.interval) / 100;
            due_time.QuadPart = -interval_100ns; // Negative = relative

            const LONG period_ms = static_cast<LONG>(config_.interval / 1'000'000);

            if (!SetWaitableTimer(
                    timer_event_,
                    &due_time,
                    period_ms,
                    NULL,
                    NULL,
                    FALSE
                )) {
                CloseHandle(timer_event_);
                CloseHandle(data_event_);
                return xerrors::Error(
                    "Failed to set waitable timer: " + std::to_string(GetLastError())
                );
            }

            timer_enabled_ = true;
        }

        // Initialize high-rate timer if needed
        if (config_.mode == ExecutionMode::HIGH_RATE ||
            config_.mode == ExecutionMode::HYBRID) {
            if (config_.interval > 0) {
                const auto interval = telem::TimeSpan(
                    static_cast<int64_t>(config_.interval)
                );
                timer_ = std::make_unique<::loop::Timer>(interval);
            }
        }

        // Apply thread priority and affinity
        if (config_.rt_priority > 0) {
            if (auto err = set_thread_priority(config_.rt_priority); err) {
                LOG(WARNING) << "[loop] Failed to set thread priority: " << err.what();
            }
        }

        if (config_.cpu_affinity >= 0) {
            if (auto err = set_cpu_affinity(config_.cpu_affinity); err) {
                LOG(WARNING) << "[loop] Failed to set CPU affinity: " << err.what();
            }
        }

        running_ = true;
        data_available_.store(false, std::memory_order_release);

        return xerrors::NIL;
    }

    void stop() override {
        if (!running_) return;

        running_ = false;
        timer_.reset();

        if (timer_event_ != NULL) {
            CancelWaitableTimer(timer_event_);
            CloseHandle(timer_event_);
            timer_event_ = NULL;
        }

        if (data_event_ != NULL) {
            CloseHandle(data_event_);
            data_event_ = NULL;
        }

        timer_enabled_ = false;
    }

private:
    /// @brief Busy-wait mode - continuously check events with zero timeout.
    void busy_wait(breaker::Breaker &breaker) {
        HANDLE handles[2];
        DWORD count = 1;
        handles[0] = data_event_;

        if (timer_enabled_) {
            handles[1] = timer_event_;
            count = 2;
        }

        while (!!breaker.running()) {
            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);

            if (result >= WAIT_OBJECT_0 && result < WAIT_OBJECT_0 + count) {
                // Event signaled
                ResetEvent(data_event_); // Manual-reset event
                data_available_.store(false, std::memory_order_release);
                return;
            }

            if (data_available_.load(std::memory_order_acquire)) {
                ResetEvent(data_event_);
                data_available_.store(false, std::memory_order_release);
                return;
            }

            if (result == WAIT_FAILED) {
                LOG(ERROR) << "[loop] WaitForMultipleObjects failed: "
                           << GetLastError();
                return;
            }
        }
    }

    /// @brief High-rate polling with precise sleep.
    void high_rate_wait(breaker::Breaker &breaker) {
        if (timer_) {
            timer_->wait(breaker);
        } else {
            // No timer configured, use short sleep
            std::this_thread::sleep_for(std::chrono::microseconds(100));
        }

        ResetEvent(data_event_);
        data_available_.store(false, std::memory_order_release);
    }

    /// @brief Event-driven wait using WaitForMultipleObjects.
    void event_driven_wait(breaker::Breaker &breaker, bool blocking) {
        HANDLE handles[2];
        DWORD count = 1;
        handles[0] = data_event_;

        if (timer_enabled_) {
            handles[1] = timer_event_;
            count = 2;
        }

        // Blocking: INFINITE, Non-blocking (RT): short timeout
        const DWORD timeout_ms = blocking ? INFINITE : 10;

        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout_ms);

        if (result >= WAIT_OBJECT_0 && result < WAIT_OBJECT_0 + count) {
            // Event signaled
            ResetEvent(data_event_);
        } else if (result == WAIT_FAILED) {
            LOG(ERROR) << "[loop] WaitForMultipleObjects failed: " << GetLastError();
        }

        data_available_.store(false, std::memory_order_release);
    }

    /// @brief Hybrid mode - spin briefly, then block on events.
    void hybrid_wait(breaker::Breaker &breaker) {
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = std::chrono::microseconds(config_.spin_duration_us);

        HANDLE handles[2];
        DWORD count = 1;
        handles[0] = data_event_;

        if (timer_enabled_) {
            handles[1] = timer_event_;
            count = 2;
        }

        // Spin phase - non-blocking checks
        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;

            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);

            if (result >= WAIT_OBJECT_0 && result < WAIT_OBJECT_0 + count) {
                ResetEvent(data_event_);
                data_available_.store(false, std::memory_order_release);
                return;
            }

            if (data_available_.load(std::memory_order_acquire)) {
                ResetEvent(data_event_);
                data_available_.store(false, std::memory_order_release);
                return;
            }
        }

        // Block phase - wait with timeout
        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 10); // 10ms
        if (result >= WAIT_OBJECT_0 && result < WAIT_OBJECT_0 + count) {
            ResetEvent(data_event_);
        }

        data_available_.store(false, std::memory_order_release);
    }

    /// @brief Set thread priority (Windows-specific).
    xerrors::Error set_thread_priority(int priority) {
        // Map priority (1-99) to Windows priority levels
        // Windows: THREAD_PRIORITY_TIME_CRITICAL = 15 (highest)
        //          THREAD_PRIORITY_HIGHEST = 2
        //          THREAD_PRIORITY_ABOVE_NORMAL = 1
        //          THREAD_PRIORITY_NORMAL = 0
        //          ...

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
            return xerrors::Error(
                "Failed to set thread priority: " + std::to_string(GetLastError())
            );
        }

        return xerrors::NIL;
    }

    /// @brief Set CPU affinity (Windows-specific).
    xerrors::Error set_cpu_affinity(int cpu) {
        const DWORD_PTR mask = static_cast<DWORD_PTR>(1) << cpu;

        if (!SetThreadAffinityMask(GetCurrentThread(), mask)) {
            return xerrors::Error(
                "Failed to set thread affinity: " + std::to_string(GetLastError())
            );
        }

        return xerrors::NIL;
    }

    Config config_;
    HANDLE data_event_ = NULL;
    HANDLE timer_event_ = NULL;
    bool timer_enabled_ = false;
    std::unique_ptr<::loop::Timer> timer_;
    std::atomic<bool> data_available_{false};
    bool running_ = false;
};

std::unique_ptr<Loop> create() {
    return std::make_unique<WindowsLoop>();
}
}
