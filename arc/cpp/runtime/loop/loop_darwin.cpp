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

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xthread/rt.h"

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

    void wait(breaker::Breaker &breaker) override {
        if (this->kqueue_fd_ == -1) return;

        switch (this->config_.mode) {
            case ExecutionMode::AUTO:
            case ExecutionMode::EVENT_DRIVEN:
                this->event_driven_wait();
                break;
            case ExecutionMode::BUSY_WAIT:
                this->busy_wait(breaker);
                break;
            case ExecutionMode::HIGH_RATE:
                this->high_rate_wait(breaker);
                break;
            case ExecutionMode::HYBRID:
                this->hybrid_wait(breaker);
                break;
            case ExecutionMode::RT_EVENT:
                this->high_rate_wait(breaker);
                break;
        }
    }

    xerrors::Error start() override {
        if (this->kqueue_fd_ != -1) return xerrors::NIL;

        // Handle RT_EVENT fallback on macOS
        if (this->config_.mode == ExecutionMode::RT_EVENT) {
            LOG(INFO) << "[loop] RT_EVENT mode not supported on macOS, "
                      << "falling back to HIGH_RATE";
        }

        // Create kqueue for event multiplexing
        this->kqueue_fd_ = kqueue();
        if (this->kqueue_fd_ == -1)
            return xerrors::Error(
                "Failed to create kqueue: " + std::string(strerror(errno))
            );

        // Register user event filter for data notifications
        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, EV_ADD | EV_CLEAR, 0, 0, nullptr);
        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            close(this->kqueue_fd_);
            return xerrors::Error(
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
                this->timer_ = std::make_unique<::loop::Timer>(this->config_.interval);
            // Use kqueue timer for EVENT_DRIVEN/HYBRID/BUSY_WAIT (ms precision OK)
            else if (auto err = this->setup_kqueue_timer(); err) {
                close(this->kqueue_fd_);
                return err;
            }
        }

        xthread::RTConfig rt_cfg;
        rt_cfg.enabled = this->config_.rt_priority > 0;
        rt_cfg.priority = this->config_.rt_priority;
        rt_cfg.cpu_affinity = this->config_.cpu_affinity;
        rt_cfg.lock_memory = this->config_.lock_memory;
        if (rt_cfg.enabled && this->config_.interval.nanoseconds() > 0) {
            rt_cfg.period = this->config_.interval;
            rt_cfg.computation = this->config_.interval * 0.2;
            rt_cfg.deadline = this->config_.interval * 0.8;
        }
        if (auto err = xthread::apply_rt_config(rt_cfg); err)
            LOG(WARNING) << "[loop] Failed to apply RT config: " << err.message();

        return xerrors::NIL;
    }

    void wake() override {
        if (this->kqueue_fd_ == -1) return;
        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);
        kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr);
    }

    bool watch(notify::Notifier &notifier) override {
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

    xerrors::Error setup_kqueue_timer() {
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
            return xerrors::Error(
                "Failed to register timer event: " + std::string(strerror(errno))
            );

        this->kqueue_timer_enabled_ = true;
        return xerrors::NIL;
    }

    /// @brief BUSY_WAIT: Non-blocking kqueue poll in tight loop.
    void busy_wait(const breaker::Breaker &breaker) const {
        constexpr timespec timeout = {0, 0};
        struct kevent events[8];

        while (breaker.running()) {
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n > 0) return;
            if (n == -1 && errno != EINTR && errno != EBADF) {
                LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
                return;
            }
        }
    }

    /// @brief HIGH_RATE: Precise software timer + non-blocking kqueue drain.
    void high_rate_wait(breaker::Breaker &breaker) const {
        this->timer_->wait(breaker);
        constexpr timespec timeout = {0, 0};
        struct kevent events[8];
        kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
    }

    /// @brief HYBRID: Spin for configured duration, then block with timeout.
    void hybrid_wait(const breaker::Breaker &breaker) const {
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = this->config_.spin_duration.chrono();
        struct timespec timeout = {0, 0};
        struct kevent events[8];
        // Spin phase: non-blocking poll
        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n > 0) return;
        }
        // Block phase: wait with timeout
        const auto block_timeout_ns = timing::HYBRID_BLOCK_TIMEOUT.nanoseconds();
        timeout.tv_sec = 0;
        timeout.tv_nsec = block_timeout_ns;
        kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
    }

    /// @brief EVENT_DRIVEN: Block on kqueue events with timeout.
    void event_driven_wait() const {
        struct kevent events[8];
        // Use timeout to ensure we periodically check breaker.running()
        // in the caller's loop.
        const struct timespec timeout = {0, timing::EVENT_DRIVEN_TIMEOUT.nanoseconds()};
        const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);

        if (n == -1 && errno != EINTR)
            LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
    }

    Config config_;
    int kqueue_fd_ = -1;
    bool kqueue_timer_enabled_ = false;
    std::unique_ptr<::loop::Timer> timer_;
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<DarwinLoop>(cfg);
    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), xerrors::NIL};
}

}
