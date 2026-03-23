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
static constexpr uintptr_t TIMER_EVENT_IDENT = 2;

/// @brief Unified Darwin loop implementation using kqueue for event multiplexing.
/// Consolidates all execution modes into a single class following the Linux pattern.
class DarwinLoop final : public Loop {
public:
    explicit DarwinLoop(Config config): config_(std::move(config)) {
        if (this->config_.lock_memory)
            LOG(WARNING) << "[loop] Memory locking not fully supported on macOS";
    }

    ~DarwinLoop() override { this->close_fds(); }

    WakeReason wait(
        x::breaker::Breaker &breaker,
        x::telem::TimeSpan max_timeout = x::telem::TimeSpan(0)
    ) override {
        if (this->kqueue_fd_ == -1) return WakeReason::Shutdown;

        switch (this->config_.mode) {
            case ExecutionMode::AUTO:
            case ExecutionMode::EVENT_DRIVEN:
                return this->event_driven_wait(max_timeout);
            case ExecutionMode::BUSY_WAIT:
                return this->busy_wait(breaker);
            case ExecutionMode::HIGH_RATE:
                return this->high_rate_wait(breaker);
            case ExecutionMode::HYBRID:
                return this->hybrid_wait(breaker, max_timeout);
            case ExecutionMode::RT_EVENT:
                return this->high_rate_wait(breaker);
        }
        return WakeReason::Shutdown;
    }

    x::errors::Error start() override {
        if (this->kqueue_fd_ != -1) return x::errors::NIL;

        // Handle RT_EVENT fallback on macOS
        if (this->config_.mode == ExecutionMode::RT_EVENT) {
            LOG(INFO) << "[loop] RT_EVENT mode not supported on macOS, "
                      << "falling back to HIGH_RATE";
        }

        // Create kqueue for event multiplexing
        this->kqueue_fd_ = kqueue();
        if (this->kqueue_fd_ == -1)
            return x::errors::Error(
                "Failed to create kqueue: " + std::string(strerror(errno))
            );

        // Register user event filter for data notifications
        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, EV_ADD | EV_CLEAR, 0, 0, nullptr);
        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            close(this->kqueue_fd_);
            return x::errors::Error(
                "Failed to register user event: " + std::string(strerror(errno))
            );
        }

        // Set up timer based on mode and interval
        if (this->config_.interval.nanoseconds() > 0) {
            const bool use_software_timer = this->config_.mode ==
                                                ExecutionMode::HIGH_RATE ||
                                            this->config_.mode ==
                                                ExecutionMode::RT_EVENT ||
                                            this->config_.interval <
                                                timing::KQUEUE_TIMER_MIN;

            // Use software timer for sub-millisecond precision
            if (use_software_timer)
                this->timer_ = std::make_unique<x::loop::Timer>(this->config_.interval);
            // Use kqueue timer for EVENT_DRIVEN/HYBRID/BUSY_WAIT (ms precision OK)
            else if (auto err = this->setup_kqueue_timer(); err) {
                close(this->kqueue_fd_);
                return err;
            }
        }

        x::thread::rt::apply_config(this->config_.rt());

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
            LOG(ERROR) << "[loop] Failed to watch notifier fd " << fd << ": "
                       << strerror(errno);
            return false;
        }

        return true;
    }

private:
    void close_fds() {
        this->timer_.reset();

        if (this->kqueue_fd_ != -1) {
            close(this->kqueue_fd_);
            this->kqueue_fd_ = -1;
        }

        this->kqueue_timer_enabled_ = false;
    }

    x::errors::Error setup_kqueue_timer() {
        const uint64_t interval_ms = this->config_.interval.milliseconds();
        if (interval_ms == 0)
            LOG(WARNING) << "[loop] Interval too small for kqueue timer "
                         << "(<1ms), using 1ms";

        struct kevent kev;
        EV_SET(
            &kev,
            TIMER_EVENT_IDENT,
            EVFILT_TIMER,
            EV_ADD | EV_ENABLE,
            0,
            interval_ms > 0 ? interval_ms : timing::KQUEUE_TIMER_MIN.milliseconds(),
            nullptr
        );
        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1)
            return x::errors::Error(
                "Failed to register timer event: " + std::string(strerror(errno))
            );

        this->kqueue_timer_enabled_ = true;
        return x::errors::NIL;
    }

    /// @brief BUSY_WAIT: Non-blocking kqueue poll in tight loop.
    WakeReason busy_wait(const x::breaker::Breaker &breaker) const {
        constexpr timespec timeout = {0, 0};
        struct kevent events[8];

        while (breaker.running()) {
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n > 0) return this->classify_events(events, n);
            if (n == -1 && errno != EINTR && errno != EBADF) {
                LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
                return WakeReason::Shutdown;
            }
            // Prevent starvation of breaker-stopping threads. yield() over
            // sleep_for() to avoid adding ~50-100us of kernel timer overhead.
            std::this_thread::yield();
        }
        return WakeReason::Shutdown;
    }

    /// @brief HIGH_RATE: Precise software timer + non-blocking kqueue drain.
    WakeReason high_rate_wait(x::breaker::Breaker &breaker) const {
        this->timer_->wait(breaker);
        constexpr timespec timeout = {0, 0};
        struct kevent events[8];
        kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
        return WakeReason::Timer;
    }

    /// @brief HYBRID: Spin for configured duration, then block with timeout.
    WakeReason hybrid_wait(
        const x::breaker::Breaker &breaker,
        const x::telem::TimeSpan max_timeout
    ) const {
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = this->config_.spin_duration.chrono();
        struct timespec timeout = {0, 0};
        struct kevent events[8];
        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return WakeReason::Shutdown;
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n > 0) return this->classify_events(events, n);
        }
        const auto block_ns = max_timeout.nanoseconds() > 0
                                ? max_timeout.nanoseconds()
                                : timing::HYBRID_BLOCK_TIMEOUT.nanoseconds();
        timeout = ns_to_timespec(block_ns);
        const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
        if (n > 0) return this->classify_events(events, n);
        return WakeReason::Timeout;
    }

    /// @brief EVENT_DRIVEN: Block on kqueue events with timeout.
    WakeReason event_driven_wait(const x::telem::TimeSpan max_timeout) const {
        struct kevent events[8];
        const auto timeout_ns = max_timeout.nanoseconds() > 0
                                  ? max_timeout.nanoseconds()
                                  : timing::EVENT_DRIVEN_TIMEOUT.nanoseconds();
        const auto timeout = ns_to_timespec(timeout_ns);
        const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);

        if (n > 0) return this->classify_events(events, n);
        if (n == 0) return WakeReason::Timeout;
        if (errno != EINTR) LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
        return WakeReason::Shutdown;
    }

    /// @brief Classifies kqueue events to determine wake reason.
    WakeReason classify_events(struct kevent *events, const int n) const {
        bool input_fired = false;
        for (int i = 0; i < n; i++) {
            if (events[i].ident == TIMER_EVENT_IDENT) return WakeReason::Timer;
            if (events[i].ident != USER_EVENT_IDENT) input_fired = true;
            // USER_EVENT_IDENT fires when wake() is called - falls through to Shutdown
        }
        if (input_fired) return WakeReason::Input;
        return WakeReason::Shutdown;
    }

    static constexpr timespec ns_to_timespec(const int64_t ns) {
        return {ns / 1'000'000'000, ns % 1'000'000'000};
    }

    Config config_;
    int kqueue_fd_ = -1;
    bool kqueue_timer_enabled_ = false;
    std::unique_ptr<x::loop::Timer> timer_;
};

std::unique_ptr<Loop> create(const Config &cfg) {
    return std::make_unique<DarwinLoop>(cfg);
}

}
