// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <cstring>
#include <string>
#include <thread>

#include "absl/log/log.h"
#include <sys/event.h>
#include <sys/time.h>
#include <sys/types.h>
#include <unistd.h>

#include "x/cpp/errors/errors.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/thread/rt/rt.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

static constexpr uintptr_t USER_EVENT_IDENT = 1;
static constexpr uintptr_t DEADLINE_EVENT_IDENT = 2;

/// @brief applies a timer change to a kqueue with the OS call.
struct KqueueArm {
    /// @brief applies kev to the kqueue kq. Returns 0, or -1 with errno set.
    static int set(const int kq, const struct kevent &kev) {
        return kevent(kq, &kev, 1, nullptr, 0, nullptr);
    }
};

/// @brief the loop of macOS, built on kqueue. Arm sets the deadline timer.
template<typename Arm = KqueueArm>
class Darwin final : public Loop {
public:
    explicit Darwin(
        Config config,
        std::shared_ptr<x::thread::rt::Handle> rt_handle = nullptr
    ):
        config_(std::move(config)), rt_handle_(std::move(rt_handle)) {
        if (this->config_.memory_locked)
            LOG(WARNING) << "[arc.loop] Memory locking not fully supported on macOS";
    }

    ~Darwin() override { this->close_fds(); }

    WakeReason wait(
        x::breaker::Breaker &breaker,
        x::telem::TimeSpan max_timeout = x::telem::TimeSpan(0),
        x::telem::TimeSpan span = x::telem::TimeSpan::max()
    ) override {
        if (this->kqueue_fd_ == -1) return WakeReason::Shutdown;

        switch (this->config_.mode) {
            case ExecutionMode::AUTO:
                if (auto_spins(span)) return this->hybrid_wait(breaker, max_timeout);
                return this->event_driven_wait(breaker, max_timeout);
            case ExecutionMode::EVENT_DRIVEN:
                return this->event_driven_wait(breaker, max_timeout);
            case ExecutionMode::BUSY_WAIT:
                return this->busy_wait(breaker, max_timeout);
            case ExecutionMode::HIGH_RATE:
                return this->high_rate_wait(breaker, max_timeout);
            case ExecutionMode::HYBRID:
            case ExecutionMode::RT_EVENT:
                return this->hybrid_wait(breaker, max_timeout);
        }
        return WakeReason::Shutdown;
    }

    x::errors::Error start() override {
        if (this->kqueue_fd_ != -1) return x::errors::NIL;

        // Create kqueue for event multiplexing
        this->kqueue_fd_ = kqueue();
        if (this->kqueue_fd_ == -1)
            return this->fail_start(
                x::errors::Error(
                    "Failed to create kqueue: " + std::string(strerror(errno))
                )
            );

        // Register user event filter for data notifications
        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, EV_ADD | EV_CLEAR, 0, 0, nullptr);
        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1)
            return this->fail_start(
                x::errors::Error(
                    "Failed to register user event: " + std::string(strerror(errno))
                )
            );

        if (!this->rt_handle_) {
            x::thread::rt::apply_config(this->config_.rt());
        } else {
            this->rt_handle_->apply();
        }

        return x::errors::NIL;
    }

    void wake() override {
        if (this->kqueue_fd_ == -1) return;
        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);
        kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr);
    }

    bool watch(x::notify::Notifier &notifier) override {
        const int fd = notifier.fd();
        if (fd == -1 || this->kqueue_fd_ == -1) return false;

        struct kevent kev;
        EV_SET(&kev, fd, EVFILT_READ, EV_ADD | EV_CLEAR, 0, 0, nullptr);

        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            LOG(ERROR) << "[arc.loop] Failed to watch notifier fd " << fd << ": "
                       << strerror(errno);
            return false;
        }

        return true;
    }

private:
    /// @brief closes and clears the descriptors that start opened, so a later start
    /// opens them again. Returns err.
    x::errors::Error fail_start(x::errors::Error err) {
        this->close_fds();
        return err;
    }

    void close_fds() {
        if (this->kqueue_fd_ != -1) {
            close(this->kqueue_fd_);
            this->kqueue_fd_ = -1;
        }
    }

    /// @brief BUSY_WAIT: Non-blocking kqueue poll in tight loop.
    WakeReason busy_wait(
        const x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) {
        const auto sw = x::telem::Stopwatch();
        constexpr timespec timeout = {0, 0};
        struct kevent events[8];

        while (breaker.running()) {
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n > 0) return this->classify_events(events, n);
            if (n == -1 && errno != EINTR && errno != EBADF) {
                LOG(ERROR) << "[arc.loop] kevent error: " << strerror(errno);
                return WakeReason::Shutdown;
            }
            if (max_timeout.nanoseconds() > 0 && sw.elapsed() >= max_timeout)
                return WakeReason::Timer;
            // Prevent starvation of breaker-stopping threads. yield() over
            // sleep_for() to avoid adding ~50-100us of kernel timer overhead.
            std::this_thread::yield();
        }
        return WakeReason::Shutdown;
    }

    /// @brief HIGH_RATE: Precise software sleep to the interval or the deadline,
    /// whichever is first, or until the breaker stops. Then a non-blocking kqueue
    /// drain.
    WakeReason high_rate_wait(
        const x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) {
        const auto span = high_rate_span(this->config_, max_timeout);
        if (!this->sleeper_.precise_sleep(span, breaker)) return WakeReason::Shutdown;
        constexpr timespec timeout = {0, 0};
        struct kevent events[8];
        kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
        return WakeReason::Timer;
    }

    /// @brief HYBRID: Spin for configured duration, then block with timeout. It deletes
    /// a timer that an earlier event-driven wait armed, so that it does not wake this
    /// wait.
    WakeReason hybrid_wait(
        const x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) {
        this->arm_deadline(x::telem::TimeSpan(0));
        const auto sw = x::telem::Stopwatch();
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = this->config_.spin_duration.chrono();
        struct timespec timeout = {0, 0};
        struct kevent events[8];
        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return WakeReason::Shutdown;
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n > 0) return this->classify_events(events, n);
        }
        if (max_timeout.nanoseconds() > 0) return this->deadline_wait(sw, max_timeout);
        timeout = ns_to_timespec(timing::HYBRID_BLOCK_TIMEOUT.nanoseconds());
        const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
        if (n > 0) return this->classify_events(events, n);
        return WakeReason::Timeout;
    }

    /// @brief EVENT_DRIVEN: blocks until an event or the deadline, with no spin.
    WakeReason event_driven_wait(
        const x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) {
        const bool armed = this->arm_deadline(max_timeout);
        if (!armed && max_timeout.nanoseconds() > 0)
            return this->busy_wait(breaker, max_timeout);
        const auto timeout = ns_to_timespec(timing::EVENT_DRIVEN_TIMEOUT.nanoseconds());
        struct kevent events[8];
        const int n = kevent(
            this->kqueue_fd_,
            nullptr,
            0,
            events,
            8,
            armed ? nullptr : &timeout
        );
        if (n == 0) return WakeReason::Timeout;
        return this->wake_reason(events, n);
    }

    /// @brief arms a one-shot timer for max_timeout and returns true. NOTE_CRITICAL
    /// removes the slack that the kernel adds to group timers, which is up to 1 ms.
    /// With no deadline, it deletes a pending timer so that it does not wake the loop,
    /// and returns false. A re-arm drops a fire that was not read.
    bool arm_deadline(const x::telem::TimeSpan max_timeout) {
        const bool deadline = max_timeout.nanoseconds() > 0;
        if (!deadline && !this->deadline_armed_) return false;
        struct kevent kev;
        EV_SET(
            &kev,
            DEADLINE_EVENT_IDENT,
            EVFILT_TIMER,
            deadline ? EV_ADD | EV_ONESHOT : EV_DELETE,
            NOTE_NSECONDS | NOTE_CRITICAL,
            deadline ? max_timeout.nanoseconds() : 0,
            nullptr
        );
        if (Arm::set(this->kqueue_fd_, kev) == -1) {
            this->report_arm_failure(strerror(errno));
            this->deadline_armed_ = false;
            return false;
        }
        this->deadline_armed_ = deadline;
        return deadline;
    }

    /// @brief logs the first failed arm. The loop then spins to each deadline.
    void report_arm_failure(const std::string &cause) {
        if (this->arm_failed_) return;
        this->arm_failed_ = true;
        LOG(ERROR) << "[arc.loop] failed to arm the deadline timer, spinning to each "
                   << "deadline instead: " << cause;
    }

    /// @brief HYBRID: blocks until the spin span ahead of deadline, then spins to it.
    /// The deadline counts from the start of sw.
    WakeReason
    deadline_wait(const x::telem::Stopwatch &sw, const x::telem::TimeSpan deadline) {
        struct kevent events[8];
        const auto block = deadline - sw.elapsed() - timing::DARWIN_DEADLINE_SPIN;
        if (block.nanoseconds() > 0) {
            const auto timeout = ns_to_timespec(block.nanoseconds());
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n != 0) return this->wake_reason(events, n);
        }
        constexpr timespec poll = {0, 0};
        while (sw.elapsed() < deadline) {
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &poll);
            if (n != 0) return this->wake_reason(events, n);
        }
        return WakeReason::Timeout;
    }

    /// @brief returns the wake reason of a kevent call that returned n != 0. An
    /// error logs and returns Shutdown.
    WakeReason wake_reason(struct kevent *events, const int n) {
        if (n > 0) return this->classify_events(events, n);
        if (errno != EINTR)
            LOG(ERROR) << "[arc.loop] kevent error: " << strerror(errno);
        return WakeReason::Shutdown;
    }

    /// @brief Classifies kqueue events to determine wake reason.
    WakeReason classify_events(struct kevent *events, const int n) {
        bool timer_fired = false;
        bool input_fired = false;
        for (int i = 0; i < n; i++) {
            if (events[i].filter == EVFILT_TIMER) {
                timer_fired = true;
                if (events[i].ident == DEADLINE_EVENT_IDENT)
                    this->deadline_armed_ = false;
            } else if (events[i].filter == EVFILT_READ)
                input_fired = true;
            // EVFILT_USER fires when wake() is called - falls through to Shutdown
        }
        if (timer_fired) return WakeReason::Timer;
        if (input_fired) return WakeReason::Input;
        return WakeReason::Shutdown;
    }

    static constexpr timespec ns_to_timespec(const int64_t ns) {
        return {ns / 1'000'000'000, ns % 1'000'000'000};
    }

    Config config_;
    std::shared_ptr<x::thread::rt::Handle> rt_handle_;
    int kqueue_fd_ = -1;
    bool deadline_armed_ = false;
    bool arm_failed_ = false;
    x::loop::Timer sleeper_;
};
}
