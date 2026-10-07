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
#include <cstring>
#include <string>
#include <thread>

#include "absl/log/log.h"
#include <fcntl.h>
#include <sched.h>
#include <sys/epoll.h>
#include <sys/eventfd.h>
#include <sys/syscall.h>
#include <sys/timerfd.h>
#include <unistd.h>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/thread/rt/rt.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

#ifndef SCHED_DEADLINE
#define SCHED_DEADLINE 6
#endif

/// @brief the OS calls of the Linux loop.
struct EpollApi {
    /// @brief waits like epoll_wait, but for a timeout in nanoseconds. Returns -1 with
    /// errno ENOSYS before Linux 5.11, which has no epoll_pwait2.
    static int
    wait(const int epfd, epoll_event *events, const int max, const timespec &timeout) {
#ifdef SYS_epoll_pwait2
        return static_cast<int>(
            syscall(SYS_epoll_pwait2, epfd, events, max, &timeout, nullptr, 0)
        );
#else
        errno = ENOSYS;
        return -1;
#endif
    }

    /// @brief sets the timer of fd to ts. Returns 0, or -1 with errno set.
    static int set(const int fd, const itimerspec &ts) {
        return timerfd_settime(fd, 0, &ts, nullptr);
    }

    /// @brief returns true when the calling thread has a real-time policy.
    static bool realtime() {
        const int policy = sched_getscheduler(0);
        return policy == SCHED_FIFO || policy == SCHED_RR || policy == SCHED_DEADLINE;
    }

    /// @brief keeps every core out of idle states with a nonzero exit latency until
    /// the returned descriptor closes. Returns -1 with errno set on failure. Writing
    /// the device needs root.
    static int hold_latency() {
        const int fd = open("/dev/cpu_dma_latency", O_WRONLY | O_CLOEXEC);
        if (fd == -1) return -1;
        constexpr int32_t latency = 0;
        if (write(fd, &latency, sizeof(latency)) == sizeof(latency)) return fd;
        const int err = errno;
        close(fd);
        errno = err;
        return -1;
    }
};

/// @brief the loop of Linux, built on epoll. A real-time thread blocks to a deadline
/// on an epoll_pwait2 timeout. On PREEMPT_RT, a timerfd wakes through a softirq thread
/// at priority 1, so any busy real-time thread on the core delays it. Other threads
/// arm a timerfd, as an epoll timeout adds 0.1% of the wait as slack to them. Api
/// makes the OS calls.
template<typename Api = EpollApi>
class Linux final : public Loop {
public:
    explicit Linux(
        const Config &config,
        std::shared_ptr<x::thread::rt::Handle> rt_handle = nullptr,
        Api api = Api{}
    ):
        config_(config), rt_handle_(std::move(rt_handle)), api_(std::move(api)) {}

    ~Linux() override { this->close_fds(); }

    WakeReason wait(
        x::breaker::Breaker &breaker,
        x::telem::TimeSpan max_timeout = x::telem::TimeSpan(0),
        x::telem::TimeSpan span = x::telem::TimeSpan::max()
    ) override {
        if (this->epoll_fd_ == -1) return WakeReason::Shutdown;

        switch (this->config_.mode) {
            case ExecutionMode::BUSY_WAIT:
                if (this->spins_) return this->busy_wait(breaker, max_timeout);
                return this->hybrid_wait(breaker, max_timeout, span);
            case ExecutionMode::HIGH_RATE:
                return this->high_rate_wait(breaker, max_timeout);
            case ExecutionMode::HYBRID:
                return this->hybrid_wait(breaker, max_timeout, span);
            case ExecutionMode::AUTO:
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

        if (!this->rt_handle_) {
            auto rt_cfg = this->config_.rt();
            x::thread::rt::apply_config(rt_cfg);
        } else {
            this->rt_handle_->apply();
        }

        const bool realtime = this->api_.realtime();
        const bool busy = this->config_.mode == ExecutionMode::BUSY_WAIT;
        this->spins_ = busy && !realtime;
        // HIGH_RATE and a spinning BUSY_WAIT check the deadline against the clock.
        if (this->config_.mode == ExecutionMode::HIGH_RATE || this->spins_)
            return x::errors::NIL;
        if (busy) this->hold_latency();
        this->direct_ = realtime && this->direct_waits();
        if (this->direct_) return x::errors::NIL;
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
                    "Failed to add timerfd to epoll: " + std::string(strerror(errno))
                )
            );
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
    /// @brief returns true when epoll_pwait2 exists. Linux before 5.11 has no
    /// epoll_pwait2, and some seccomp filters block it.
    bool direct_waits() {
        epoll_event event;
        if (this->api_.wait(this->epoll_fd_, &event, 1, timespec{}) != -1) return true;
        return errno != ENOSYS && errno != EPERM;
    }

    /// @brief waits for events until span elapses. Returns the number of events, 0
    /// when span elapses first, or -1 with errno set.
    int timed_wait(epoll_event *events, const x::telem::TimeSpan span) {
        const int64_t ns = std::max<int64_t>(span.nanoseconds(), 0);
        const int64_t second = x::telem::SECOND.nanoseconds();
        timespec ts{};
        ts.tv_sec = ns / second;
        ts.tv_nsec = ns % second;
        return this->api_.wait(this->epoll_fd_, events, 2, ts);
    }

    /// @brief closes and clears the descriptors that start opened, so a later start
    /// opens them again. Returns err.
    x::errors::Error fail_start(x::errors::Error err) {
        this->close_fds();
        return err;
    }

    /// @brief keeps all cores out of deep idle states while the loop runs, so a core
    /// that blocks before a deadline wakes in time. Logs a warning on failure, as an
    /// unprivileged process cannot hold the latency.
    void hold_latency() {
        this->latency_fd_ = this->api_.hold_latency();
        if (this->latency_fd_ == -1)
            LOG(WARNING) << "[arc.loop] failed to keep cores out of deep idle states, "
                         << "so their wake-up can delay deadlines: " << strerror(errno);
    }

    void close_fds() {
        if (this->latency_fd_ != -1) {
            close(this->latency_fd_);
            this->latency_fd_ = -1;
        }

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

        this->direct_ = false;
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
    /// whichever is first, or until the breaker stops. Then a non-blocking epoll drain.
    WakeReason high_rate_wait(
        const x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) {
        const auto span = high_rate_span(this->config_, max_timeout);
        if (!this->sleeper_.precise_sleep(span, breaker)) return WakeReason::Shutdown;
        struct epoll_event events[2];
        const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
        if (n > 0) this->drain_events(events, n);
        return WakeReason::Timer;
    }

    /// @brief arms the timer to fire once at the deadline and returns true. With no
    /// deadline, it disarms the timer, which also clears a fire from an earlier
    /// deadline.
    bool arm_deadline(const x::telem::TimeSpan max_timeout) {
        if (this->timer_fd_ == -1) return false;
        const int64_t deadline = std::max<int64_t>(max_timeout.nanoseconds(), 0);
        const int64_t second = x::telem::SECOND.nanoseconds();
        struct itimerspec ts{};
        ts.it_value.tv_sec = deadline / second;
        ts.it_value.tv_nsec = deadline % second;
        if (this->api_.set(this->timer_fd_, ts) == -1) {
            report_arm_failure(this->arm_failed_, strerror(errno));
            return false;
        }
        return deadline > 0;
    }

    WakeReason event_driven_wait(
        x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) {
        const bool deadline = max_timeout.nanoseconds() > 0;
        const bool armed = this->arm_deadline(max_timeout);
        if (!this->direct_ && !armed && deadline)
            return this->busy_wait(breaker, max_timeout);
        struct epoll_event events[2];
        int n;
        if (this->direct_)
            n = this->timed_wait(
                events,
                deadline ? max_timeout : timing::EVENT_DRIVEN_TIMEOUT
            );
        else
            n = epoll_wait(
                this->epoll_fd_,
                events,
                2,
                armed ? -1 : timing::EVENT_DRIVEN_TIMEOUT.milliseconds()
            );

        if (n > 0) return this->consume_events(events, n);
        if (n == 0)
            return this->direct_ && deadline ? WakeReason::Timer : WakeReason::Timeout;
        if (errno != EINTR)
            LOG(ERROR) << "[arc.loop] epoll_wait error: " << strerror(errno);
        return WakeReason::Shutdown;
    }

    /// @brief HYBRID: blocks until timing::LINUX_DEADLINE_SPIN ahead of the deadline,
    /// then spins to it. The spin takes at most half of span, the period of the timer
    /// that owns the deadline, so a real-time thread leaves its core half of each
    /// period.
    WakeReason hybrid_wait(
        x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout,
        const x::telem::TimeSpan span
    ) {
        const auto sw = x::telem::Stopwatch();
        struct epoll_event events[2];
        const bool deadline = max_timeout.nanoseconds() > 0;
        const auto spin = std::min(timing::LINUX_DEADLINE_SPIN, span / 2);
        const auto block = deadline ? max_timeout - spin : x::telem::TimeSpan(0);
        // A block of zero or less disarms the timer, so no stale fire ends the spin.
        const bool armed = this->arm_deadline(block);
        if (!deadline || block.nanoseconds() > 0) {
            if (!this->direct_ && !armed && deadline)
                return this->busy_wait(breaker, max_timeout);
            int n;
            if (this->direct_)
                n = this->timed_wait(
                    events,
                    deadline ? block : timing::HYBRID_BLOCK_TIMEOUT
                );
            else
                n = epoll_wait(
                    this->epoll_fd_,
                    events,
                    2,
                    armed ? -1 : timing::HYBRID_BLOCK_TIMEOUT.milliseconds()
                );
            if (n > 0) {
                const auto reason = this->consume_events(events, n);
                if (reason != WakeReason::Timer) return reason;
            } else if (n == -1) {
                if (errno != EINTR)
                    LOG(ERROR) << "[arc.loop] epoll_wait error: " << strerror(errno);
                return WakeReason::Shutdown;
            }
            if (!deadline) return WakeReason::Timeout;
        }
        while (sw.elapsed() < max_timeout) {
            if (!breaker.running()) return WakeReason::Shutdown;
            const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
            if (n > 0) return this->consume_events(events, n);
        }
        return WakeReason::Timer;
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
    Api api_;
    int epoll_fd_ = -1;
    int event_fd_ = -1;
    int timer_fd_ = -1;
    /// @brief holds all cores out of deep idle states while open.
    int latency_fd_ = -1;
    /// @brief true when waits block to their deadline on an epoll_pwait2 timeout
    /// instead of a timerfd.
    bool direct_ = false;
    /// @brief true when BUSY_WAIT spins through each wait, which only a thread that is
    /// not real-time does.
    bool spins_ = false;
    bool arm_failed_ = false;
    ::x::loop::Timer sleeper_;
};
}
