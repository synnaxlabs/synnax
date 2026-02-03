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
#include <sys/epoll.h>
#include <sys/eventfd.h>
#include <sys/timerfd.h>
#include <unistd.h>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xthread/rt.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

class LinuxLoop final : public Loop {
public:
    explicit LinuxLoop(const Config &config): config_(config) {}

    ~LinuxLoop() override { this->close_fds(); }

    void wait(breaker::Breaker &breaker) override {
        if (this->epoll_fd_ == -1) return;

        switch (this->config_.mode) {
            case ExecutionMode::BUSY_WAIT:
                this->busy_wait(breaker);
                break;
            case ExecutionMode::HIGH_RATE:
                this->high_rate_wait(breaker);
                break;
            case ExecutionMode::RT_EVENT:
                this->event_driven_wait(true);
                break;
            case ExecutionMode::HYBRID:
                this->hybrid_wait(breaker);
                break;
            case ExecutionMode::AUTO:
            case ExecutionMode::EVENT_DRIVEN:
                this->event_driven_wait(true);
                break;
        }
    }

    xerrors::Error start() override {
        if (this->epoll_fd_ != -1) return xerrors::NIL;

        this->epoll_fd_ = epoll_create1(0);
        if (this->epoll_fd_ == -1)
            return xerrors::Error(
                "Failed to create epoll: " + std::string(strerror(errno))
            );

        this->event_fd_ = eventfd(0, EFD_NONBLOCK);
        if (this->event_fd_ == -1) {
            close(this->epoll_fd_);
            return xerrors::Error(
                "Failed to create eventfd: " + std::string(strerror(errno))
            );
        }

        struct epoll_event ev;
        ev.events = EPOLLIN;
        ev.data.fd = this->event_fd_;
        if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_ADD, this->event_fd_, &ev) == -1) {
            close(this->event_fd_);
            close(this->epoll_fd_);
            return xerrors::Error(
                "Failed to add eventfd to epoll: " + std::string(strerror(errno))
            );
        }

        if (this->config_.interval.nanoseconds() > 0) {
            if (this->config_.mode == ExecutionMode::HIGH_RATE)
                this->timer_ = std::make_unique<::loop::Timer>(this->config_.interval);
            else {
                this->timer_fd_ = timerfd_create(CLOCK_MONOTONIC, TFD_NONBLOCK);
                if (this->timer_fd_ == -1) {
                    close(this->event_fd_);
                    close(this->epoll_fd_);
                    return xerrors::Error(
                        "Failed to create timerfd: " + std::string(strerror(errno))
                    );
                }

                const uint64_t interval_ns = this->config_.interval.nanoseconds();
                struct itimerspec ts;
                ts.it_interval.tv_sec = interval_ns / telem::SECOND.nanoseconds();
                ts.it_interval.tv_nsec = interval_ns % telem::SECOND.nanoseconds();
                ts.it_value = ts.it_interval;

                if (timerfd_settime(this->timer_fd_, 0, &ts, nullptr) == -1) {
                    close(this->timer_fd_);
                    close(this->event_fd_);
                    close(this->epoll_fd_);
                    return xerrors::Error(
                        "Failed to set timerfd interval: " +
                        std::string(strerror(errno))
                    );
                }

                ev.events = EPOLLIN;
                ev.data.fd = this->timer_fd_;
                if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_ADD, this->timer_fd_, &ev) ==
                    -1) {
                    close(this->timer_fd_);
                    close(this->event_fd_);
                    close(this->epoll_fd_);
                    return xerrors::Error(
                        "Failed to add timerfd to epoll: " +
                        std::string(strerror(errno))
                    );
                }

                this->timer_enabled_ = true;
            }
        }

        auto rt_cfg = this->config_.rt();
        rt_cfg.prefer_deadline_scheduler = true;
        if (auto err = xthread::apply_rt_config(rt_cfg); err)
            LOG(WARNING) << "[loop] Failed to apply RT config: " << err.message();

        return xerrors::NIL;
    }

    void wake() override {
        if (this->event_fd_ == -1) return;
        const uint64_t val = 1;
        [[maybe_unused]] auto _ = write(this->event_fd_, &val, sizeof(val));
    }

    bool watch(notify::Notifier &notifier) override {
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
                    LOG(ERROR) << "[loop] Failed to modify watched fd " << fd << ": "
                               << strerror(errno);
                    return false;
                }
                return true;
            }
            LOG(ERROR) << "[loop] Failed to watch notifier fd " << fd << ": "
                       << strerror(errno);
            return false;
        }

        return true;
    }

private:
    void close_fds() {
        this->timer_.reset();

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

    void busy_wait(breaker::Breaker &breaker) {
        struct epoll_event events[2];

        while (breaker.running()) {
            const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
            if (n > 0) {
                this->consume_events(events, n);
                return;
            }
            if (n == -1 && errno != EINTR) {
                LOG(ERROR) << "[loop] epoll_wait error: " << strerror(errno);
                return;
            }
        }
    }

    void high_rate_wait(breaker::Breaker &breaker) {
        this->timer_->wait(breaker);
        struct epoll_event events[2];
        const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
        if (n > 0) this->drain_events(events, n);
    }

    void event_driven_wait(bool blocking) {
        struct epoll_event events[2];
        // Use a short timeout to ensure we periodically check breaker.running()
        // in the caller's loop.
        const int timeout_ms = blocking ? timing::EVENT_DRIVEN_TIMEOUT.milliseconds()
                                        : timing::POLL_TIMEOUT.milliseconds();
        const int n = epoll_wait(this->epoll_fd_, events, 2, timeout_ms);

        if (n > 0)
            this->consume_events(events, n);
        else if (n == -1 && errno != EINTR)
            LOG(ERROR) << "[loop] epoll_wait error: " << strerror(errno);
    }

    void hybrid_wait(const breaker::Breaker &breaker) {
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = std::chrono::nanoseconds(
            this->config_.spin_duration.nanoseconds()
        );

        struct epoll_event events[2];

        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;

            const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
            if (n > 0) {
                this->consume_events(events, n);
                return;
            }
        }

        const int timeout_ms = timing::HYBRID_BLOCK_TIMEOUT.milliseconds();
        const int n = epoll_wait(this->epoll_fd_, events, 2, timeout_ms);
        if (n > 0) this->consume_events(events, n);
    }

    /// @brief Consumes events from epoll, returning total timer expirations.
    uint64_t consume_events(struct epoll_event *events, const int n) {
        uint64_t total_expirations = 0;
        for (int i = 0; i < n; i++) {
            uint64_t val;
            const ssize_t ret = read(events[i].data.fd, &val, sizeof(val));
            if (ret == sizeof(val) && events[i].data.fd == this->timer_fd_) {
                total_expirations += val;
                if (val > 1)
                    LOG(WARNING) << "[loop] timer drift detected: " << val
                                 << " expirations in single read";
            }
        }
        return total_expirations;
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
    int epoll_fd_ = -1;
    int event_fd_ = -1;
    int timer_fd_ = -1;
    bool timer_enabled_ = false;
    std::unique_ptr<::loop::Timer> timer_;
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<LinuxLoop>(cfg);
    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), xerrors::NIL};
}

}
