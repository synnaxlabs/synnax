// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>

#include "absl/log/log.h"
#include <windows.h>

// timeBeginPeriod/timeEndPeriod from winmm.lib. We declare them manually instead of
// including <timeapi.h> because WIN32_LEAN_AND_MEAN (set by the build) excludes
// multimedia headers and their transitive type dependencies.
extern "C" {
__declspec(dllimport) UINT WINAPI timeBeginPeriod(UINT uPeriod);
__declspec(dllimport) UINT WINAPI timeEndPeriod(UINT uPeriod);
}

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/thread/rt/rt.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {
#ifndef CREATE_WAITABLE_TIMER_HIGH_RESOLUTION
#define CREATE_WAITABLE_TIMER_HIGH_RESOLUTION 0x00000002
#endif

class WindowsLoop final : public Loop {
    static constexpr DWORD MAX_HANDLES = MAXIMUM_WAIT_OBJECTS;

public:
    explicit WindowsLoop(
        const Config &config,
        std::shared_ptr<x::thread::rt::Handle> rt_handle = nullptr
    ):
        config_(config), rt_handle_(std::move(rt_handle)) {
        if (this->config_.memory_locked) {
            LOG(WARNING) << "[arc.loop] Memory locking on Windows requires "
                         << "VirtualLock API (not implemented)";
        }
    }

    ~WindowsLoop() override { this->close_handles(); }

    WakeReason wait(
        x::breaker::Breaker &breaker,
        x::telem::TimeSpan max_timeout = x::telem::TimeSpan(0)
    ) override {
        if (this->wake_event_ == NULL) return WakeReason::Shutdown;

        switch (this->config_.mode) {
            case ExecutionMode::BUSY_WAIT:
                return this->busy_wait(breaker);
            case ExecutionMode::HIGH_RATE:
                return this->high_rate_wait(breaker);
            case ExecutionMode::RT_EVENT:
                return this->event_driven_wait(false, max_timeout);
            case ExecutionMode::HYBRID:
                return this->hybrid_wait(breaker, max_timeout);
            case ExecutionMode::AUTO:
            case ExecutionMode::EVENT_DRIVEN:
                return this->event_driven_wait(true, max_timeout);
        }
        return WakeReason::Shutdown;
    }

    x::errors::Error start() override {
        if (this->wake_event_ != NULL) return x::errors::NIL;

        this->wake_event_ = CreateEvent(NULL, FALSE, FALSE, NULL);
        if (this->wake_event_ == NULL) {
            return x::errors::Error(
                "Failed to create wake event: " + std::to_string(GetLastError())
            );
        }

        if (this->config_.mode == ExecutionMode::HIGH_RATE) {
            if (this->config_.interval.nanoseconds() > 0)
                this->timer_ = std::make_unique<::x::loop::Timer>(
                    this->config_.interval
                );
        } else if (auto err = this->create_waitable_timer()) {
            CloseHandle(this->wake_event_);
            return err;
        }

        if (!this->rt_handle_) {
            auto rt_cfg = this->config_.rt();
            rt_cfg.use_mmcss = true;
            x::thread::rt::apply_config(rt_cfg);
        } else {
            this->rt_handle_->apply();
        }

        // The thread config can take milliseconds, so the timer arms after it.
        if (this->timer_enabled_ && !this->arm_timer()) {
            const auto err = GetLastError();
            this->close_handles();
            return x::errors::Error(
                "Failed to set waitable timer: " + std::to_string(err)
            );
        }
        return x::errors::NIL;
    }

    void wake() override {
        if (this->wake_event_ == NULL) return;
        SetEvent(this->wake_event_);
    }

    bool watch(x::notify::Notifier &notifier) override {
        auto *handle = static_cast<HANDLE>(notifier.native_handle());
        if (handle == nullptr) {
            LOG(ERROR) << "[arc.loop] Notifier has no native handle";
            return false;
        }
        if (this->watched_handle_ != NULL && this->watched_handle_ != handle) {
            LOG(ERROR) << "[arc.loop] Only one external notifier can be watched";
            return false;
        }
        this->watched_handle_ = handle;
        return true;
    }

private:
    // Try CREATE_WAITABLE_TIMER_HIGH_RESOLUTION first for sub-millisecond precision
    // without global side effects. Falls back to a standard timer with
    // timeBeginPeriod(1) on pre-Windows 10 1803 systems. Both use one-shot re-arming
    // instead of periodic mode because the periodic lPeriod parameter doesn't benefit
    // from the high-resolution mechanism.
    x::errors::Error create_waitable_timer() {
        this->timer_event_ = CreateWaitableTimerExW(
            NULL,
            NULL,
            CREATE_WAITABLE_TIMER_HIGH_RESOLUTION,
            TIMER_ALL_ACCESS
        );
        if (this->timer_event_ != NULL) {
            this->high_res_timer_ = true;
            VLOG(1) << "[arc.loop] using high-resolution waitable timer";
        } else {
            this->timer_event_ = CreateWaitableTimer(NULL, FALSE, NULL);
            if (this->timer_event_ == NULL)
                return x::errors::Error(
                    "Failed to create waitable timer: " + std::to_string(GetLastError())
                );
            timeBeginPeriod(1);
            this->used_time_begin_period_ = true;
            VLOG(1) << "[arc.loop] using standard waitable timer with "
                    << "timeBeginPeriod(1) fallback";
        }
        this->timer_enabled_ = true;
        return x::errors::NIL;
    }

    bool arm_timer() const { return this->arm_timer(this->config_.interval); }

    // Arms the timer to fire once after span, or leaves it disarmed for a non-positive
    // span. A cancel does not clear a fire, so the drain clears one left from an
    // earlier arm.
    bool arm_timer(const x::telem::TimeSpan span) const {
        CancelWaitableTimer(this->timer_event_);
        WaitForSingleObject(this->timer_event_, 0);
        if (span.nanoseconds() <= 0) return true;
        LARGE_INTEGER due_time;
        const int64_t span_100ns = span.nanoseconds() /
                                   timing::WINDOWS_TIMER_UNIT.nanoseconds();
        due_time.QuadPart = -span_100ns;
        return SetWaitableTimer(this->timer_event_, &due_time, 0, NULL, NULL, FALSE);
    }

    // Only HYBRID and RT_EVENT spin. Without the spin, the timer wakes about 0.5 ms
    // late.
    x::telem::TimeSpan deadline_spin() const {
        if (this->config_.mode == ExecutionMode::HYBRID ||
            this->config_.mode == ExecutionMode::RT_EVENT)
            return timing::WINDOWS_DEADLINE_SPIN;
        return x::telem::TimeSpan(0);
    }

    // A deadline inside the spin span has no time for a timer wake, which takes
    // about 0.5 ms.
    bool inside_spin(const x::telem::TimeSpan max_timeout) const {
        return this->timer_enabled_ && max_timeout.nanoseconds() > 0 &&
               max_timeout <= this->deadline_spin();
    }

    // Arms the timer the spin span ahead of a deadline. With no deadline and no
    // interval, it disarms the timer so that an earlier deadline does not wake the
    // loop.
    bool arm_deadline(const x::telem::TimeSpan max_timeout) const {
        if (!this->timer_enabled_) return false;
        if (max_timeout.nanoseconds() <= 0) {
            if (this->config_.interval.nanoseconds() <= 0) this->arm_timer();
            return false;
        }
        return this->arm_timer(max_timeout - this->deadline_spin());
    }

    // Spins to the deadline after a timer wake. The timer handle is last and unwatched.
    WakeReason finish_timer_wake(
        const HANDLE *handles,
        const DWORD count,
        const x::telem::Stopwatch &sw,
        const x::telem::TimeSpan deadline
    ) const {
        auto reason = WakeReason::Timer;
        while (reason == WakeReason::Timer && sw.elapsed() < deadline) {
            const DWORD result = WaitForMultipleObjects(count - 1, handles, FALSE, 0);
            if (result < WAIT_OBJECT_0 + count - 1)
                reason = this->classify_result(result, handles);
        }
        this->arm_timer();
        return reason;
    }

    void close_handles() {
        this->timer_.reset();

        if (this->timer_event_ != NULL) {
            CancelWaitableTimer(this->timer_event_);
            CloseHandle(this->timer_event_);
            this->timer_event_ = NULL;
        }

        if (this->used_time_begin_period_) {
            timeEndPeriod(1);
            this->used_time_begin_period_ = false;
        }

        if (this->wake_event_ != NULL) {
            CloseHandle(this->wake_event_);
            this->wake_event_ = NULL;
        }

        this->timer_enabled_ = false;
    }

    WakeReason busy_wait(x::breaker::Breaker &breaker) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);
        if (count == 0) return WakeReason::Shutdown;

        while (breaker.running()) {
            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);
            if (result < WAIT_OBJECT_0 + count) {
                const auto reason = this->classify_result(result, handles);
                if (reason == WakeReason::Timer) this->arm_timer();
                return reason;
            }
            if (result == WAIT_FAILED) {
                LOG(ERROR) << "[arc.loop] WaitForMultipleObjects failed: "
                           << GetLastError();
                return WakeReason::Shutdown;
            }
        }
        return WakeReason::Shutdown;
    }

    WakeReason high_rate_wait(x::breaker::Breaker &breaker) {
        this->timer_->wait(breaker);
        return WakeReason::Timer;
    }

    WakeReason event_driven_wait(bool blocking, const x::telem::TimeSpan max_timeout) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);
        if (count == 0) return WakeReason::Shutdown;

        const DWORD default_ms = blocking
                                   ? static_cast<DWORD>(
                                         timing::EVENT_DRIVEN_TIMEOUT.milliseconds()
                                     )
                                   : static_cast<DWORD>(
                                         timing::HYBRID_BLOCK_TIMEOUT.milliseconds()
                                     );
        const auto sw = x::telem::Stopwatch();
        if (this->inside_spin(max_timeout))
            return this->finish_timer_wake(handles, count, sw, max_timeout);
        const bool deadline = this->arm_deadline(max_timeout);
        const DWORD timeout_ms = deadline ? INFINITE : default_ms;

        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout_ms);
        if (result == WAIT_TIMEOUT) return WakeReason::Timeout;
        if (result == WAIT_FAILED) {
            LOG(ERROR) << "[arc.loop] WaitForMultipleObjects failed: "
                       << GetLastError();
            return WakeReason::Shutdown;
        }
        const auto reason = this->classify_result(result, handles);
        if (reason != WakeReason::Timer) return reason;
        return this->finish_timer_wake(
            handles,
            count,
            sw,
            deadline ? max_timeout : x::telem::TimeSpan(0)
        );
    }

    WakeReason
    hybrid_wait(x::breaker::Breaker &breaker, const x::telem::TimeSpan max_timeout) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);
        if (count == 0) return WakeReason::Shutdown;
        const auto sw = x::telem::Stopwatch();
        if (this->inside_spin(max_timeout))
            return this->finish_timer_wake(handles, count, sw, max_timeout);
        const bool deadline = this->arm_deadline(max_timeout);
        const auto spin_until = deadline ? max_timeout : x::telem::TimeSpan(0);

        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = std::chrono::nanoseconds(
            this->config_.spin_duration.nanoseconds()
        );

        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return WakeReason::Shutdown;

            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);
            if (result < WAIT_OBJECT_0 + count) {
                const auto reason = this->classify_result(result, handles);
                if (reason != WakeReason::Timer) return reason;
                return this->finish_timer_wake(handles, count, sw, spin_until);
            }
        }

        const DWORD timeout_ms = deadline
                                   ? INFINITE
                                   : static_cast<DWORD>(
                                         timing::HYBRID_BLOCK_TIMEOUT.milliseconds()
                                     );
        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout_ms);
        if (result == WAIT_TIMEOUT) return WakeReason::Timeout;
        if (result < WAIT_OBJECT_0 + count) {
            const auto reason = this->classify_result(result, handles);
            if (reason != WakeReason::Timer) return reason;
            return this->finish_timer_wake(handles, count, sw, spin_until);
        }
        return WakeReason::Shutdown;
    }

    /// @brief Classifies which handle was signaled to determine wake reason.
    WakeReason classify_result(const DWORD result, const HANDLE *handles) const {
        const DWORD index = result - WAIT_OBJECT_0;
        if (this->timer_enabled_ && handles[index] == this->timer_event_)
            return WakeReason::Timer;
        if (handles[index] == this->watched_handle_) return WakeReason::Input;
        return WakeReason::Shutdown;
    }

    DWORD build_handles(HANDLE *handles) const {
        DWORD count = 0;
        if (this->wake_event_ != NULL) handles[count++] = this->wake_event_;
        if (this->watched_handle_ != NULL) handles[count++] = this->watched_handle_;
        if (this->timer_enabled_) handles[count++] = this->timer_event_;
        return count;
    }

    Config config_;
    std::shared_ptr<x::thread::rt::Handle> rt_handle_;
    HANDLE wake_event_ = NULL;
    HANDLE timer_event_ = NULL;
    HANDLE watched_handle_ = NULL;
    bool timer_enabled_ = false;
    bool high_res_timer_ = false;
    bool used_time_begin_period_ = false;
    std::unique_ptr<::x::loop::Timer> timer_;
};

std::unique_ptr<Loop>
create(const Config &cfg, std::shared_ptr<x::thread::rt::Handle> rt_handle) {
    return std::make_unique<WindowsLoop>(cfg, std::move(rt_handle));
}

x::telem::TimeSpan min_timer_span() {
    return timing::WINDOWS_MIN_TIMER_SPAN;
}

x::telem::TimeSpan hybrid_threshold() {
    return timing::WINDOWS_HYBRID_THRESHOLD;
}

}
