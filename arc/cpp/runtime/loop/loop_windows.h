// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "absl/log/log.h"
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#ifndef NOMINMAX
#define NOMINMAX
#endif
#include <windows.h>

// timeBeginPeriod/timeEndPeriod from winmm.lib. We declare them manually instead of
// including <timeapi.h> because WIN32_LEAN_AND_MEAN excludes multimedia headers and
// their transitive type dependencies.
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

/// @brief the OS calls of the waitable timer that the Windows loop uses.
struct WaitableTimerApi {
    /// @brief creates a high-resolution timer. Returns NULL before Windows 10 1803,
    /// which has no such timer.
    static HANDLE create_high_resolution() {
        return CreateWaitableTimerExW(
            NULL,
            NULL,
            CREATE_WAITABLE_TIMER_HIGH_RESOLUTION,
            TIMER_ALL_ACCESS
        );
    }

    /// @brief creates a timer that fires on the system tick. Returns NULL on failure.
    static HANDLE create() { return CreateWaitableTimer(NULL, FALSE, NULL); }

    /// @brief sets timer to fire once at due. Returns FALSE on failure, with the cause
    /// in GetLastError.
    static BOOL set(const HANDLE timer, const LARGE_INTEGER &due) {
        return SetWaitableTimer(timer, &due, 0, NULL, NULL, FALSE);
    }

    /// @brief raises the system tick to 1 ms. Before Windows 10 2004, the raise
    /// applies to every process on the machine.
    static void raise_tick() { timeBeginPeriod(1); }

    /// @brief lowers the system tick that raise_tick raised.
    static void lower_tick() { timeEndPeriod(1); }
};

/// @brief the loop of Windows, built on a waitable timer. Api makes the timer calls.
template<typename Api = WaitableTimerApi>
class Windows final : public Loop {
public:
    explicit Windows(
        const Config &config,
        std::shared_ptr<x::thread::rt::Handle> rt_handle = nullptr,
        Api api = Api{}
    ):
        config_(config), rt_handle_(std::move(rt_handle)), api_(std::move(api)) {
        if (this->config_.memory_locked) {
            LOG(WARNING) << "[arc.loop] Memory locking on Windows requires "
                         << "VirtualLock API (not implemented)";
        }
    }

    ~Windows() override { this->close_handles(); }

    WakeReason wait(
        x::breaker::Breaker &breaker,
        x::telem::TimeSpan max_timeout = x::telem::TimeSpan(0),
        x::telem::TimeSpan span = x::telem::TimeSpan::max()
    ) override {
        if (this->wake_event_ == NULL) return WakeReason::Shutdown;

        switch (this->config_.mode) {
            case ExecutionMode::BUSY_WAIT:
                return this->busy_wait(breaker, max_timeout);
            case ExecutionMode::HIGH_RATE:
                return this->high_rate_wait(breaker, max_timeout);
            case ExecutionMode::HYBRID:
                return this->hybrid_wait(breaker, max_timeout);
            case ExecutionMode::AUTO:
            case ExecutionMode::RT_EVENT:
                return this->event_driven_wait(
                    breaker,
                    max_timeout,
                    timing::WINDOWS_DEADLINE_SPIN
                );
            case ExecutionMode::EVENT_DRIVEN:
                return this
                    ->event_driven_wait(breaker, max_timeout, x::telem::TimeSpan(0));
        }
        return WakeReason::Shutdown;
    }

    x::errors::Error start() override {
        if (this->wake_event_ != NULL) return x::errors::NIL;

        this->wake_event_ = CreateEvent(NULL, FALSE, FALSE, NULL);
        if (this->wake_event_ == NULL)
            return this->fail_start(
                x::errors::Error(
                    "Failed to create wake event: " + std::to_string(GetLastError())
                )
            );

        // HIGH_RATE and BUSY_WAIT check the deadline against the clock.
        if (this->config_.mode != ExecutionMode::HIGH_RATE &&
            this->config_.mode != ExecutionMode::BUSY_WAIT) {
            if (auto err = this->create_waitable_timer()) return this->fail_start(err);
        }

        if (!this->rt_handle_) {
            auto rt_cfg = this->config_.rt();
            rt_cfg.use_mmcss = true;
            x::thread::rt::apply_config(rt_cfg);
        } else {
            this->rt_handle_->apply();
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
    /// @brief prefers a high-resolution timer. Before Windows 10 1803, a standard
    /// timer fires on the system tick. The loop raises the tick to 1 ms when it arms a
    /// deadline and lowers it when a wait has no deadline. Both timers re-arm once per
    /// deadline, as the periodic lPeriod parameter does not use the high-resolution
    /// mechanism.
    x::errors::Error create_waitable_timer() {
        this->timer_event_ = this->api_.create_high_resolution();
        if (this->timer_event_ != NULL) {
            this->high_res_timer_ = true;
            VLOG(1) << "[arc.loop] using high-resolution waitable timer";
        } else {
            this->timer_event_ = this->api_.create();
            if (this->timer_event_ == NULL)
                return x::errors::Error(
                    "Failed to create waitable timer: " + std::to_string(GetLastError())
                );
            VLOG(1) << "[arc.loop] using standard waitable timer with a 1 ms tick";
        }
        return x::errors::NIL;
    }

    /// @brief raises or lowers the system tick for a standard timer.
    void set_tick_raised(const bool raised) {
        if (this->high_res_timer_ || this->tick_raised_ == raised) return;
        if (raised)
            this->api_.raise_tick();
        else
            this->api_.lower_tick();
        this->tick_raised_ = raised;
    }

    /// @brief disarms the timer. A cancel does not clear a fire, so the drain clears
    /// one left from an earlier arm.
    void disarm_timer() const {
        CancelWaitableTimer(this->timer_event_);
        WaitForSingleObject(this->timer_event_, 0);
    }

    /// @brief arms the timer to fire once after span.
    bool arm_timer(const x::telem::TimeSpan span) {
        this->disarm_timer();
        LARGE_INTEGER due_time;
        const int64_t span_100ns = span.nanoseconds() /
                                   timing::WINDOWS_TIMER_UNIT.nanoseconds();
        due_time.QuadPart = -span_100ns;
        if (this->api_.set(this->timer_event_, due_time)) return true;
        report_arm_failure(this->arm_failed_, std::to_string(GetLastError()));
        return false;
    }

    /// @brief returns true when a deadline is inside the spin span, which has no time
    /// for a timer wake of about 0.5 ms. The timer also cannot arm less than one
    /// WINDOWS_TIMER_UNIT ahead.
    bool inside_spin(
        const x::telem::TimeSpan max_timeout,
        const x::telem::TimeSpan spin
    ) const {
        return this->timer_event_ != NULL && max_timeout.nanoseconds() > 0 &&
               max_timeout < spin + timing::WINDOWS_TIMER_UNIT;
    }

    /// @brief arms the timer the spin span ahead of a deadline. With no deadline, it
    /// disarms the timer so that an earlier deadline does not wake the loop.
    bool
    arm_deadline(const x::telem::TimeSpan max_timeout, const x::telem::TimeSpan spin) {
        if (this->timer_event_ == NULL) return false;
        if (max_timeout.nanoseconds() <= 0) {
            this->disarm_timer();
            this->set_tick_raised(false);
            return false;
        }
        this->set_tick_raised(true);
        return this->arm_timer(max_timeout - spin);
    }

    /// @brief spins to the deadline after a timer wake. The timer handle is last and
    /// unwatched.
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
        this->disarm_timer();
        return reason;
    }

    /// @brief closes and clears the handles that start opened, so a later start opens
    /// them again. Returns err.
    x::errors::Error fail_start(x::errors::Error err) {
        this->close_handles();
        return err;
    }

    void close_handles() {
        if (this->timer_event_ != NULL) {
            CancelWaitableTimer(this->timer_event_);
            CloseHandle(this->timer_event_);
            this->timer_event_ = NULL;
        }

        this->set_tick_raised(false);

        if (this->wake_event_ != NULL) {
            CloseHandle(this->wake_event_);
            this->wake_event_ = NULL;
        }
    }

    WakeReason
    busy_wait(x::breaker::Breaker &breaker, const x::telem::TimeSpan max_timeout) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);
        const auto sw = x::telem::Stopwatch();

        while (breaker.running()) {
            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);
            if (result < WAIT_OBJECT_0 + count)
                return this->classify_result(result, handles);
            if (result == WAIT_FAILED) {
                LOG(ERROR) << "[arc.loop] WaitForMultipleObjects failed: "
                           << GetLastError();
                return WakeReason::Shutdown;
            }
            if (max_timeout.nanoseconds() > 0 && sw.elapsed() >= max_timeout)
                return WakeReason::Timer;
        }
        return WakeReason::Shutdown;
    }

    /// @brief HIGH_RATE: Precise software sleep to the interval or the deadline,
    /// whichever is first, or until the breaker stops.
    WakeReason high_rate_wait(
        const x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) {
        const auto span = high_rate_span(this->config_, max_timeout);
        if (!this->sleeper_.precise_sleep(span, breaker)) return WakeReason::Shutdown;
        return WakeReason::Timer;
    }

    /// @brief blocks until an event or the spin span ahead of the deadline, then spins
    /// to it.
    WakeReason event_driven_wait(
        x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout,
        const x::telem::TimeSpan spin
    ) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);

        const auto sw = x::telem::Stopwatch();
        if (this->inside_spin(max_timeout, spin))
            return this->finish_timer_wake(handles, count, sw, max_timeout);
        const bool armed = this->arm_deadline(max_timeout, spin);
        if (!armed && max_timeout.nanoseconds() > 0)
            return this->busy_wait(breaker, max_timeout);
        const DWORD timeout_ms = armed ? INFINITE
                                       : static_cast<DWORD>(
                                             timing::EVENT_DRIVEN_TIMEOUT.milliseconds()
                                         );

        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout_ms);
        if (result == WAIT_TIMEOUT) return WakeReason::Timeout;
        if (result == WAIT_FAILED) {
            LOG(ERROR) << "[arc.loop] WaitForMultipleObjects failed: "
                       << GetLastError();
            return WakeReason::Shutdown;
        }
        const auto reason = this->classify_result(result, handles);
        if (reason != WakeReason::Timer) return reason;
        return this->finish_timer_wake(handles, count, sw, max_timeout);
    }

    WakeReason
    hybrid_wait(x::breaker::Breaker &breaker, const x::telem::TimeSpan max_timeout) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);
        const auto sw = x::telem::Stopwatch();
        const auto spin = timing::WINDOWS_DEADLINE_SPIN;
        if (this->inside_spin(max_timeout, spin))
            return this->finish_timer_wake(handles, count, sw, max_timeout);
        const bool armed = this->arm_deadline(max_timeout, spin);
        if (!armed && max_timeout.nanoseconds() > 0)
            return this->busy_wait(breaker, max_timeout);

        while (sw.elapsed() < this->config_.spin_duration) {
            if (!breaker.running()) return WakeReason::Shutdown;

            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);
            if (result < WAIT_OBJECT_0 + count) {
                const auto reason = this->classify_result(result, handles);
                if (reason != WakeReason::Timer) return reason;
                return this->finish_timer_wake(handles, count, sw, max_timeout);
            }
        }

        const DWORD timeout_ms = armed ? INFINITE
                                       : static_cast<DWORD>(
                                             timing::HYBRID_BLOCK_TIMEOUT.milliseconds()
                                         );
        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout_ms);
        if (result == WAIT_TIMEOUT) return WakeReason::Timeout;
        if (result < WAIT_OBJECT_0 + count) {
            const auto reason = this->classify_result(result, handles);
            if (reason != WakeReason::Timer) return reason;
            return this->finish_timer_wake(handles, count, sw, max_timeout);
        }
        return WakeReason::Shutdown;
    }

    /// @brief Classifies which handle was signaled to determine wake reason.
    WakeReason classify_result(const DWORD result, const HANDLE *handles) const {
        const DWORD index = result - WAIT_OBJECT_0;
        if (handles[index] == this->timer_event_) return WakeReason::Timer;
        if (handles[index] == this->watched_handle_) return WakeReason::Input;
        return WakeReason::Shutdown;
    }

    DWORD build_handles(HANDLE *handles) const {
        handles[0] = this->wake_event_;
        DWORD count = 1;
        if (this->watched_handle_ != NULL) handles[count++] = this->watched_handle_;
        if (this->timer_event_ != NULL) handles[count++] = this->timer_event_;
        return count;
    }

    Config config_;
    std::shared_ptr<x::thread::rt::Handle> rt_handle_;
    HANDLE wake_event_ = NULL;
    HANDLE timer_event_ = NULL;
    HANDLE watched_handle_ = NULL;
    bool arm_failed_ = false;
    bool high_res_timer_ = false;
    bool tick_raised_ = false;
    Api api_;
    ::x::loop::Timer sleeper_;
};
}
