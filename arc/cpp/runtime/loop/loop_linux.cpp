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
#include <thread>

#include "glog/logging.h"
#include <sched.h>
#include <sys/epoll.h>
#include <sys/eventfd.h>
#include <sys/mman.h>
#include <sys/timerfd.h>
#include <unistd.h>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

class LinuxLoop final : public Loop {
public:
    explicit LinuxLoop(const Config &config): config_(config) {}

    ~LinuxLoop() override { this->close_fds(); }

    void notify_data() override {
        if (!this->running_ || this->event_fd_ == -1) return;
        const uint64_t val = 1;
        if (write(this->event_fd_, &val, sizeof(val)) != sizeof(val)) {
            LOG(ERROR) << "[loop] Failed to write to eventfd: " << strerror(errno);
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
                // RT_EVENT: True RT with indefinite blocking (relies on
                // eventfd/timerfd)
                this->event_driven_wait(breaker, /*blocking=*/true);
                break;
            case ExecutionMode::EVENT_DRIVEN:
                this->event_driven_wait(breaker, /*blocking=*/true);
                break;
            case ExecutionMode::HYBRID:
                this->hybrid_wait(breaker);
                break;
        }
    }

    x::errors::Error start() override {
        if (this->running_) return x::errors::NIL;

        this->epoll_fd_ = epoll_create1(0);
        if (this->epoll_fd_ == -1) {
            return x::errors::Error(
                "Failed to create epoll: " + std::string(strerror(errno))
            );
        }

        this->event_fd_ = eventfd(0, EFD_NONBLOCK);
        if (this->event_fd_ == -1) {
            close(this->epoll_fd_);
            return x::errors::Error(
                "Failed to create eventfd: " + std::string(strerror(errno))
            );
        }

        struct epoll_event ev;
        ev.events = EPOLLIN;
        ev.data.fd = this->event_fd_;
        if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_ADD, this->event_fd_, &ev) == -1) {
            close(this->event_fd_);
            close(this->epoll_fd_);
            return x::errors::Error(
                "Failed to add eventfd to epoll: " + std::string(strerror(errno))
            );
        }

        if (this->config_.interval.nanoseconds() > 0) {
            this->timer_fd_ = timerfd_create(CLOCK_MONOTONIC, TFD_NONBLOCK);
            if (this->timer_fd_ == -1) {
                close(this->event_fd_);
                close(this->epoll_fd_);
                return x::errors::Error(
                    "Failed to create timerfd: " + std::string(strerror(errno))
                );
            }

            const uint64_t interval_ns = this->config_.interval.nanoseconds();
            struct itimerspec ts;
            ts.it_interval.tv_sec = interval_ns / 1'000'000'000;
            ts.it_interval.tv_nsec = interval_ns % 1'000'000'000;
            ts.it_value = ts.it_interval;

            if (timerfd_settime(this->timer_fd_, 0, &ts, nullptr) == -1) {
                close(this->timer_fd_);
                close(this->event_fd_);
                close(this->epoll_fd_);
                return x::errors::Error(
                    "Failed to set timerfd interval: " + std::string(strerror(errno))
                );
            }

            ev.events = EPOLLIN;
            ev.data.fd = this->timer_fd_;
            if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_ADD, this->timer_fd_, &ev) == -1) {
                close(this->timer_fd_);
                close(this->event_fd_);
                close(this->epoll_fd_);
                return x::errors::Error(
                    "Failed to add timerfd to epoll: " + std::string(strerror(errno))
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
            if (auto err = this->set_rt_priority(this->config_.rt_priority); err) {
                LOG(WARNING) << "[loop] Failed to set RT priority: " << err.message();
            }
        }

        if (this->config_.cpu_affinity >= 0) {
            if (auto err = this->set_cpu_affinity(this->config_.cpu_affinity); err) {
                LOG(WARNING) << "[loop] Failed to set CPU affinity: " << err.message();
            }
        }

        if (this->config_.lock_memory) {
            if (auto err = this->lock_memory(); err) {
                LOG(WARNING) << "[loop] Failed to lock memory: " << err.message();
            }
        }

        this->running_ = true;
        this->data_available_.store(false, std::memory_order_release);

        return x::errors::NIL;
    }

    void stop() override {
        if (!this->running_) return;

        // Signal the thread to stop by writing to eventfd and setting running_ to
        // false. We intentionally do NOT close file descriptors here - that happens
        // in close_fds() called by the destructor, after the thread has been joined.
        // This avoids a race condition where closing fds before the thread exits
        // could cause the eventfd wake-up signal to be lost.
        if (this->event_fd_ != -1) {
            const uint64_t val = 1;
            [[maybe_unused]] auto _ = write(this->event_fd_, &val, sizeof(val));
        }

        this->running_ = false;
    }

    bool watch(x::notify::Notifier &notifier) override {
        const int fd = notifier.fd();
        if (fd == -1 || this->epoll_fd_ == -1) return false;

        struct epoll_event ev;
        ev.events = EPOLLIN;
        ev.data.fd = fd;

        if (epoll_ctl(this->epoll_fd_, EPOLL_CTL_ADD, fd, &ev) == -1) {
            LOG(ERROR) << "[loop] Failed to watch notifier fd " << fd << ": "
                       << strerror(errno);
            return false;
        }

        return true;
    }

private:
    void close_fds() {
        this->running_ = false;
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

    void busy_wait(x::breaker::Breaker &breaker) {
        struct epoll_event events[2];

        while (!!breaker.running()) {
            const int n = epoll_wait(this->epoll_fd_, events, 2, 0);
            if (n > 0) {
                this->consume_events(events, n);
                return;
            }
            if (this->data_available_.load(std::memory_order_acquire)) {
                this->data_available_.store(false, std::memory_order_release);
                return;
            }
            if (n == -1 && errno != EINTR) {
                LOG(ERROR) << "[loop] epoll_wait error: " << strerror(errno);
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
        this->data_available_.store(false, std::memory_order_release);
    }

    void event_driven_wait(x::breaker::Breaker &breaker, bool blocking) {
        struct epoll_event events[2];
        // Use 100ms timeout instead of infinite blocking to handle shutdown race
        // where the eventfd wake-up signal could be lost if fds are closed too
        // quickly. This ensures we check running_ periodically.
        const int timeout_ms = blocking ? 100 : 10;
        VLOG(1) << "[loop] epoll_wait entering with timeout_ms=" << timeout_ms;
        const int n = epoll_wait(this->epoll_fd_, events, 2, timeout_ms);
        VLOG(1) << "[loop] epoll_wait returned n=" << n;

        if (n > 0) {
            for (int i = 0; i < n; i++) {
                VLOG(1) << "[loop] event[" << i << "] fd=" << events[i].data.fd;
            }
            this->consume_events(events, n);
        } else if (n == -1 && errno != EINTR) {
            LOG(ERROR) << "[loop] epoll_wait error: " << strerror(errno);
        }

        this->data_available_.store(false, std::memory_order_release);
    }

    void hybrid_wait(x::breaker::Breaker &breaker) {
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
            if (this->data_available_.load(std::memory_order_acquire)) {
                this->data_available_.store(false, std::memory_order_release);
                return;
            }
        }

        const int timeout_ms = timing::HYBRID_BLOCK_TIMEOUT.milliseconds();
        const int n = epoll_wait(this->epoll_fd_, events, 2, timeout_ms);
        if (n > 0) this->consume_events(events, n);

        this->data_available_.store(false, std::memory_order_release);
    }

    void consume_events(struct epoll_event *events, int n) {
        for (int i = 0; i < n; i++) {
            uint64_t val;
            ssize_t ret = read(events[i].data.fd, &val, sizeof(val));
            (void) ret;
        }
    }

    x::errors::Error set_rt_priority(int priority) {
        struct sched_param param;
        param.sched_priority = priority;

        if (sched_setscheduler(0, SCHED_FIFO, &param) == -1) {
            return x::errors::Error(
                "Failed to set SCHED_FIFO priority (requires CAP_SYS_NICE): " +
                std::string(strerror(errno))
            );
        }

        return x::errors::NIL;
    }

    x::errors::Error set_cpu_affinity(int cpu) {
        cpu_set_t cpuset;
        CPU_ZERO(&cpuset);
        CPU_SET(cpu, &cpuset);

        if (sched_setaffinity(0, sizeof(cpuset), &cpuset) == -1) {
            return x::errors::Error(
                "Failed to set CPU affinity: " + std::string(strerror(errno))
            );
        }

        return x::errors::NIL;
    }

    x::errors::Error lock_memory() {
        if (mlockall(MCL_CURRENT | MCL_FUTURE) == -1) {
            return x::errors::Error(
                "Failed to lock memory (requires CAP_IPC_LOCK): " +
                std::string(strerror(errno))
            );
        }

        return x::errors::NIL;
    }

    Config config_;
    int epoll_fd_ = -1;
    int event_fd_ = -1;
    int timer_fd_ = -1;
    bool timer_enabled_ = false;
    std::unique_ptr<::x::loop::Timer> timer_;
    std::atomic<bool> data_available_{false};
    std::atomic<bool> running_{false};
};

std::pair<std::unique_ptr<Loop>, x::errors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<LinuxLoop>(cfg);
    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), x::errors::NIL};
}

}
