// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <chrono>
#include <cstring>
#include <string>
#include <thread>

#include "absl/log/log.h"
#include <sys/epoll.h>
#include <sys/eventfd.h>
#include <sys/timerfd.h>
#include <unistd.h>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/thread/rt/rt.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

/// @brief arms a timerfd with the OS call.
struct TimerfdArm {
    /// @brief sets the timer of fd to ts. Returns 0, or -1 with errno set.
    static int set(const int fd, const itimerspec &ts) {
        return timerfd_settime(fd, 0, &ts, nullptr);
    }
};

/// @brief the loop of Linux, built on epoll and a timerfd. Arm sets the timerfd.
template<typename Arm = TimerfdArm>
class Linux final : public Loop {
public:
    explicit Linux(
        const Config &config,
        std::shared_ptr<x::thread::rt::Handle> rt_handle = nullptr
    ):
        config_(config), rt_handle_(std::move(rt_handle)) {}

    ~Linux() override { this->close_fds(); }

    WakeReason wait(
        x::breaker::Breaker &breaker,
        x::telem::TimeSpan max_timeout = x::telem::TimeSpan(0),
        x::telem::TimeSpan span = x::telem::TimeSpan::max()
    ) override {
        if (this->epoll_fd_ == -1) return WakeReason::Shutdown;

        switch (this->config_.mode) {
            case ExecutionMode::BUSY_WAIT:
                return this->busy_wait(breaker, max_timeout);
            case ExecutionMode::HIGH_RATE:
                return this->high_rate_wait(max_timeout);
            case ExecutionMode::HYBRID:
                return this->hybrid_wait(breaker, max_timeout);
            case ExecutionMode::AUTO:
                if (auto_spins(span)) return this->hybrid_wait(breaker, max_timeout);
                return this->event_driven_wait(breaker, max_timeout);
            case ExecutionMode::RT_EVENT:
            case ExecutionMode::EVENT_DRIVEN:
                return this->event_driven_wait(breaker, max_timeout);
        }
        return WakeReason::Shutdown;
    }

    x::errors::Error start() override {
        if (this->epoll_fd_ != -1) return x::errors::NIL;

        this->epoll_fd_ = epoll_create1(0);
        if (this->epoll_fd_ == -1)
            return this->fail_start(
                x::errors::Error(
                    "Failed to create epoll: " + std::string(strerror(errno))
                )
            );

        this->event_fd_ = eventfd(0, EFD_NONBLOCK);
        if (this->event_fd_ == -1)
            return this->fail_start(
                x::errors::Error(
                    "Failed to create eventfd: " + std::string(strerror(errno))
                )
            );

        struct epoll_event ev;
        ev.events = EPOLLIN;
        ev.data.fd = this->event_fd_;
        if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_ADD, this->event_fd_, &ev) == -1)
            return this->fail_start(
                x::errors::Error(
                    "Failed to add eventfd to epoll: " + std::string(strerror(errno))
                )
            );

        // HIGH_RATE and BUSY_WAIT check the deadline against the clock.
        if (this->config_.mode != ExecutionMode::HIGH_RATE &&
            this->config_.mode != ExecutionMode::BUSY_WAIT) {
            this->timer_fd_ = timerfd_create(CLOCK_MONOTONIC, TFD_NONBLOCK);
            if (this->timer_fd_ == -1)
                return this->fail_start(
                    x::errors::Error(
                        "Failed to create timerfd: " + std::string(strerror(errno))
                    )
                );
            ev.events = EPOLLIN;
            ev.data.fd = this->timer_fd_;
            if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_ADD, this->timer_fd_, &ev) == -1)
                return this->fail_start(
                    x::errors::Error(
                        "Failed to add timerfd to epoll: " +
                        std::string(strerror(errno))
                    )
                );
            this->timer_enabled_ = true;
        }

        if (!this->rt_handle_) {
            auto rt_cfg = this->config_.rt();
            x::thread::rt::apply_config(rt_cfg);
        } else {
            this->rt_handle_->apply();
        }

        return x::errors::NIL;
    }

    void wake() override {
        if (this->event_fd_ == -1) return;
        const uint64_t val = 1;
        [[maybe_unused]] auto _ = write(this->event_fd_, &val, sizeof(val));
    }

    bool watch(x::notify::Notifier &notifier) override {
        const int fd = notifier.fd();
        if (fd == -1 || this->epoll_fd_ == -1) return false;

        struct epoll_event ev;
        ev.events = EPOLLIN;
        ev.data.fd = fd;

        if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_ADD, fd, &ev) == -1) {
            if (errno == EEXIST) {
                // fd already registered (e.g., from a previous run after restart).
                // Update the registration instead - this makes watch() idempotent.
                if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_MOD, fd, &ev) == -1) {
                    LOG(ERROR) << "[arc.loop] Failed to modify watched fd " << fd
                               << ": " << strerror(errno);
                    return false;
                }
                return true;
            }
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
        if (this->timer_fd_ != -1) {
            close(this->timer_fd_);
            this->timer_fd_ = -1;
        }

        if (this->event_fd_ != -1) {
            close(this->event_fd_);
            this->event_fd_ = -1;
        }

        if (this->epoll_fd_ != -1) {
            close(this->epoll_fd_);
            this->epoll_fd_ = -1;
        }

        this->timer_enabled_ = false;
    }

    WakeReason
    busy_wait(x::breaker::Breaker &breaker, const x::telem::TimeSpan max_timeout) {
        const auto sw = x::telem::Stopwatch();
        struct epoll_event events[2];

        while (breaker.running()) {
            const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
            if (n > 0) return this->consume_events(events, n);
            if (n == -1 && errno != EINTR) {
                LOG(ERROR) << "[arc.loop] epoll_wait error: " << strerror(errno);
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
    /// whichever is first, then a non-blocking epoll drain.
    WakeReason high_rate_wait(const x::telem::TimeSpan max_timeout) {
        this->sleeper_.precise_sleep(high_rate_span(this->config_, max_timeout));
        struct epoll_event events[2];
        const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
        if (n > 0) this->drain_events(events, n);
        return WakeReason::Timer;
    }

    /// @brief arms the timer to fire once at the deadline and returns true. With no
    /// deadline, it disarms the timer, which also clears a fire from an earlier
    /// deadline.
    bool arm_deadline(const x::telem::TimeSpan max_timeout) {
        if (!this->timer_enabled_) return false;
        const int64_t deadline = std::max<int64_t>(max_timeout.nanoseconds(), 0);
        const int64_t second = x::telem::SECOND.nanoseconds();
        struct itimerspec ts{};
        ts.it_value.tv_sec = deadline / second;
        ts.it_value.tv_nsec = deadline % second;
        if (Arm::set(this->timer_fd_, ts) == -1) {
            this->report_arm_failure(strerror(errno));
            return false;
        }
        return deadline > 0;
    }

    /// @brief logs the first failed arm. The loop then spins to each deadline.
    void report_arm_failure(const std::string &cause) {
        if (this->arm_failed_) return;
        this->arm_failed_ = true;
        LOG(ERROR) << "[arc.loop] failed to arm the deadline timer, spinning to each "
                   << "deadline instead: " << cause;
    }

    WakeReason event_driven_wait(
        x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) {
        const bool armed = this->arm_deadline(max_timeout);
        if (!armed && max_timeout.nanoseconds() > 0)
            return this->busy_wait(breaker, max_timeout);
        struct epoll_event events[2];
        const int timeout_ms = armed ? -1 : timing::EVENT_DRIVEN_TIMEOUT.milliseconds();
        const int n = epoll_wait(this->epoll_fd_, events, 2, timeout_ms);

        if (n > 0) return this->consume_events(events, n);
        if (n == 0) return WakeReason::Timeout;
        if (errno != EINTR)
            LOG(ERROR) << "[arc.loop] epoll_wait error: " << strerror(errno);
        return WakeReason::Shutdown;
    }

    WakeReason
    hybrid_wait(x::breaker::Breaker &breaker, const x::telem::TimeSpan max_timeout) {
        const bool armed = this->arm_deadline(max_timeout);
        if (!armed && max_timeout.nanoseconds() > 0)
            return this->busy_wait(breaker, max_timeout);
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = std::chrono::nanoseconds(
            this->config_.spin_duration.nanoseconds()
        );

        struct epoll_event events[2];

        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return WakeReason::Shutdown;

            const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
            if (n > 0) return this->consume_events(events, n);
        }

        const int timeout_ms = armed ? -1 : timing::HYBRID_BLOCK_TIMEOUT.milliseconds();
        const int n = epoll_wait(this->epoll_fd_, events, 2, timeout_ms);
        if (n > 0) return this->consume_events(events, n);
        return WakeReason::Timeout;
    }

    /// @brief Consumes events from epoll, returning the wake reason.
    WakeReason consume_events(struct epoll_event *events, const int n) {
        bool timer_fired = false;
        bool input_fired = false;
        for (int i = 0; i < n; i++) {
            uint64_t val;
            const ssize_t ret = read(events[i].data.fd, &val, sizeof(val));
            if (ret == sizeof(val)) {
                if (events[i].data.fd == this->timer_fd_)
                    timer_fired = true;
                else if (events[i].data.fd != this->event_fd_) { input_fired = true; }
                // event_fd_ fires when wake() is called - falls through to Shutdown
            }
        }
        if (timer_fired) return WakeReason::Timer;
        if (input_fired) return WakeReason::Input;
        return WakeReason::Shutdown;
    }

    /// @brief Drains pending events without tracking expirations.
    void drain_events(struct epoll_event *events, const int n) {
        for (int i = 0; i < n; i++) {
            uint64_t val;
            [[maybe_unused]] const ssize_t ret = read(
                events[i].data.fd,
                &val,
                sizeof(val)
            );
        }
    }

    Config config_;
    std::shared_ptr<x::thread::rt::Handle> rt_handle_;
    int epoll_fd_ = -1;
    int event_fd_ = -1;
    int timer_fd_ = -1;
    bool timer_enabled_ = false;
    bool arm_failed_ = false;
    ::x::loop::Timer sleeper_;
};
}
