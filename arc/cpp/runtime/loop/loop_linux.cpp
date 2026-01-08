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
/// @brief Linux epoll-based implementation of Loop.
///
/// Uses Linux-specific primitives for efficient and RT-safe event handling:
/// - epoll: Event multiplexing
/// - timerfd: Periodic timer with nanosecond precision
/// - eventfd: User-triggered events for data notifications
/// - SCHED_FIFO: Real-time scheduling policy
/// - mlockall: Memory locking to prevent paging
///
/// Supports all execution modes including full RT guarantees on PREEMPT_RT kernels.
class LinuxLoop final : public Loop {
public:
    explicit LinuxLoop(const Config &config): config_(config) {}

    ~LinuxLoop() override { stop(); }

    void notify_data() override {
        if (!running_ || event_fd_ == -1) return;

        // Write to eventfd to wake up epoll
        const uint64_t val = 1;
        if (write(event_fd_, &val, sizeof(val)) != sizeof(val)) {
            LOG(ERROR) << "[loop] Failed to write to eventfd: " << strerror(errno);
        }

        data_available_.store(true, std::memory_order_release);
    }

    void wait(x::breaker::Breaker &breaker) override {
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

    x::errors::Error start() override {
        if (running_) {
            return x::errors::NIL; // Already started
        }

        // Create epoll instance
        epoll_fd_ = epoll_create1(0);
        if (epoll_fd_ == -1) {
            return x::errors::Error(
                "Failed to create epoll: " + std::string(strerror(errno))
            );
        }

        // Create eventfd for data notifications
        event_fd_ = eventfd(0, EFD_NONBLOCK);
        if (event_fd_ == -1) {
            close(epoll_fd_);
            return x::errors::Error(
                "Failed to create eventfd: " + std::string(strerror(errno))
            );
        }

        // Add eventfd to epoll
        struct epoll_event ev;
        ev.events = EPOLLIN;
        ev.data.fd = event_fd_;
        if (epoll_ctl(epoll_fd_, EPOLL_CTL_ADD, event_fd_, &ev) == -1) {
            close(event_fd_);
            close(epoll_fd_);
            return x::errors::Error(
                "Failed to add eventfd to epoll: " + std::string(strerror(errno))
            );
        }

        // Create timerfd if interval is configured
        if (config_.interval.nanoseconds() > 0) {
            timer_fd_ = timerfd_create(CLOCK_MONOTONIC, TFD_NONBLOCK);
            if (timer_fd_ == -1) {
                close(event_fd_);
                close(epoll_fd_);
                return x::errors::Error(
                    "Failed to create timerfd: " + std::string(strerror(errno))
                );
            }

            // Configure timer interval
            const uint64_t interval_ns = config_.interval.nanoseconds();
            struct itimerspec ts;
            ts.it_interval.tv_sec = interval_ns / 1'000'000'000;
            ts.it_interval.tv_nsec = interval_ns % 1'000'000'000;
            ts.it_value = ts.it_interval; // Initial expiration

            if (timerfd_settime(timer_fd_, 0, &ts, nullptr) == -1) {
                close(timer_fd_);
                close(event_fd_);
                close(epoll_fd_);
                return x::errors::Error(
                    "Failed to set timerfd interval: " + std::string(strerror(errno))
                );
            }

            // Add timerfd to epoll
            ev.events = EPOLLIN;
            ev.data.fd = timer_fd_;
            if (epoll_ctl(epoll_fd_, EPOLL_CTL_ADD, timer_fd_, &ev) == -1) {
                close(timer_fd_);
                close(event_fd_);
                close(epoll_fd_);
                return x::errors::Error(
                    "Failed to add timerfd to epoll: " + std::string(strerror(errno))
                );
            }

            timer_enabled_ = true;
        }

        // Initialize high-rate timer if needed
        if (config_.mode == ExecutionMode::HIGH_RATE ||
            config_.mode == ExecutionMode::HYBRID) {
            if (config_.interval.nanoseconds() > 0) {
                timer_ = std::make_unique<::x::loop::Timer>(config_.interval);
            }
        }

        // Apply RT configuration
        if (config_.rt_priority > 0) {
            if (auto err = set_rt_priority(config_.rt_priority); err) {
                LOG(WARNING) << "[loop] Failed to set RT priority: " << err.message();
            }
        }

        if (config_.cpu_affinity >= 0) {
            if (auto err = set_cpu_affinity(config_.cpu_affinity); err) {
                LOG(WARNING) << "[loop] Failed to set CPU affinity: " << err.message();
            }
        }

        if (config_.lock_memory) {
            if (auto err = lock_memory(); err) {
                LOG(WARNING) << "[loop] Failed to lock memory: " << err.message();
            }
        }

        running_ = true;
        data_available_.store(false, std::memory_order_release);

        return x::errors::NIL;
    }

    void stop() override {
        if (!running_) return;

        running_ = false;
        timer_.reset();

        if (timer_fd_ != -1) {
            close(timer_fd_);
            timer_fd_ = -1;
        }

        if (event_fd_ != -1) {
            close(event_fd_);
            event_fd_ = -1;
        }

        if (epoll_fd_ != -1) {
            close(epoll_fd_);
            epoll_fd_ = -1;
        }

        timer_enabled_ = false;
    }

private:
    /// @brief Busy-wait mode - continuously check epoll with zero timeout.
    void busy_wait(x::breaker::Breaker &breaker) {
        struct epoll_event events[2];

        while (!!breaker.running()) {
            const int n = epoll_wait(epoll_fd_, events, 2, 0);
            if (n > 0) {
                consume_events(events, n);
                return;
            }
            if (data_available_.load(std::memory_order_acquire)) {
                data_available_.store(false, std::memory_order_release);
                return;
            }
            if (n == -1 && errno != EINTR) {
                LOG(ERROR) << "[loop] epoll_wait error: " << strerror(errno);
                return;
            }
        }
    }

    /// @brief High-rate polling with precise sleep.
    void high_rate_wait(x::breaker::Breaker &breaker) {
        if (timer_) {
            timer_->wait(breaker);
        } else {
            // No timer configured, use short sleep
            std::this_thread::sleep_for(std::chrono::microseconds(100));
        }

        data_available_.store(false, std::memory_order_release);
    }

    /// @brief Event-driven wait using epoll.
    void event_driven_wait(x::breaker::Breaker &breaker, bool blocking) {
        struct epoll_event events[2];

        // Blocking: -1 (infinite), Non-blocking (RT): use short timeout
        const int timeout_ms = blocking ? -1 : 10;

        const int n = epoll_wait(epoll_fd_, events, 2, timeout_ms);

        if (n > 0) {
            consume_events(events, n);
        } else if (n == -1 && errno != EINTR) {
            LOG(ERROR) << "[loop] epoll_wait error: " << strerror(errno);
        }

        data_available_.store(false, std::memory_order_release);
    }

    /// @brief Hybrid mode - spin briefly, then block on epoll.
    void hybrid_wait(x::breaker::Breaker &breaker) {
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = std::chrono::nanoseconds(
            config_.spin_duration.nanoseconds()
        );

        struct epoll_event events[2];

        // Spin phase - non-blocking epoll checks
        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;

            const int n = epoll_wait(epoll_fd_, events, 2, 0);
            if (n > 0) {
                consume_events(events, n);
                return;
            }
            if (data_available_.load(std::memory_order_acquire)) {
                data_available_.store(false, std::memory_order_release);
                return;
            }
        }

        // Block phase - wait on epoll with timeout
        const int n = epoll_wait(epoll_fd_, events, 2, 10); // 10ms max block
        if (n > 0) { consume_events(events, n); }

        data_available_.store(false, std::memory_order_release);
    }

    /// @brief Consume events from epoll (read eventfd/timerfd to clear them).
    void consume_events(struct epoll_event *events, int n) {
        for (int i = 0; i < n; i++) {
            uint64_t val;
            // Read and discard - just to clear the event
            ssize_t ret = read(events[i].data.fd, &val, sizeof(val));
            (void) ret; // Ignore return value
        }
    }

    /// @brief Set real-time scheduling priority.
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

    /// @brief Set CPU affinity.
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

    /// @brief Lock memory to prevent paging.
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
    bool running_ = false;
};

std::pair<std::unique_ptr<Loop>, x::errors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<LinuxLoop>(cfg);
    if (auto err = loop->start(); err) { return {nullptr, err}; }
    return {std::move(loop), x::errors::NIL};
}
}
