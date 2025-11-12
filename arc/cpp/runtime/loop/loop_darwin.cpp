// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/loop/loop.h"

#include <atomic>
#include <chrono>
#include <sys/event.h>
#include <sys/time.h>
#include <sys/types.h>
#include <thread>
#include <unistd.h>

#include "glog/logging.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"

namespace arc::runtime::loop {
class DarwinLoop final : public Loop {
    Config cfg;
    int kqueue_fd_ = -1;
    bool timer_enabled = false;
    std::unique_ptr<::loop::Timer> timer;
    std::atomic<bool> data_available{false};
    bool running = false;
public:
    explicit DarwinLoop(const Config &config): cfg(config) {
        if (cfg.rt_priority > 0)
            LOG(WARNING) << "[loop] RT priority support is limited on macOS";

        if (cfg.lock_memory)
            LOG(WARNING) << "[loop] Memory locking not fully supported on macOS";

        if (cfg.mode == ExecutionMode::RT_EVENT) {
            LOG(INFO)
                << "[loop] RT_EVENT mode not supported on macOS, "
                << "falling back to HIGH_RATE";
            cfg.mode = ExecutionMode::HIGH_RATE;
        }
    }

    ~DarwinLoop() override { stop(); }

    void notify_data() override {
        if (!this->running) return;

        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);

        if (kevent(kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            LOG(ERROR) << "[loop] Failed to trigger user event: "
                       << strerror(errno);
        }

        data_available.store(true, std::memory_order_release);
    }

    void wait(breaker::Breaker &breaker) override {
        if (!this->running) return;
        switch (this->cfg.mode) {
        case ExecutionMode::BUSY_WAIT:
            busy_wait(breaker);
            break;
        case ExecutionMode::HIGH_RATE:
            high_rate_wait(breaker);
            break;
        case ExecutionMode::HYBRID:
            hybrid_wait(breaker);
            break;
        case ExecutionMode::EVENT_DRIVEN:
        case ExecutionMode::RT_EVENT:
        default:
            event_driven_wait(breaker);
            break;
        }
    }

    xerrors::Error start() override {
        if (this->running) return xerrors::NIL;

        this->kqueue_fd_ = kqueue();
        if (this->kqueue_fd_ == -1)
            return xerrors::Error(
                "Failed to create kqueue: " +
                                  std::string(strerror(errno))
            );

        // Register user event for data notifications
        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, EV_ADD | EV_CLEAR, 0, 0,
               nullptr);
        if (kevent(kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            close(kqueue_fd_);
            return xerrors::Error("Failed to register user event: " +
                                  std::string(strerror(errno)));
        }

        // Register timer event if interval is configured
        if (cfg.interval > 0) {
            // Convert nanoseconds to milliseconds for kqueue
            // NOTE_USECONDS is available on some systems, but NOTE_NSECONDS is not
            const uint64_t interval_ms = cfg.interval / 1'000'000;
            if (interval_ms == 0) {
                LOG(WARNING) << "[loop] Interval too small for kqueue timer "
                             << "(<1ms), using 1ms";
            }

            EV_SET(&kev, TIMER_EVENT_IDENT, EVFILT_TIMER, EV_ADD | EV_ENABLE, 0,
                   interval_ms > 0 ? interval_ms : 1, nullptr);
            if (kevent(kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
                close(kqueue_fd_);
                return xerrors::Error("Failed to register timer event: " +
                                      std::string(strerror(errno)));
            }

            timer_enabled = true;
        }

        // Initialize high-rate timer if needed
        if (cfg.mode == ExecutionMode::HIGH_RATE ||
            cfg.mode == ExecutionMode::HYBRID) {
            if (cfg.interval > 0) {
                const auto interval =
                    telem::TimeSpan(static_cast<int64_t>(cfg.interval));
                timer = std::make_unique<::loop::Timer>(interval);
            }
        }

        // Apply thread priority and affinity if configured
        if (cfg.rt_priority > 0) {
            set_thread_priority(cfg.rt_priority);
        }
        if (cfg.cpu_affinity >= 0) {
            set_cpu_affinity(cfg.cpu_affinity);
        }

        running = true;
        data_available.store(false, std::memory_order_release);

        return xerrors::NIL;
    }

    void stop() override {
        if (!running) return;

        running = false;
        timer.reset();

        if (kqueue_fd_ != -1) {
            close(kqueue_fd_);
            kqueue_fd_ = -1;
        }

        timer_enabled = false;
    }

  private:
    static constexpr uintptr_t USER_EVENT_IDENT = 1;
    static constexpr uintptr_t TIMER_EVENT_IDENT = 2;

    /// @brief Busy-wait mode - continuously check kqueue with zero timeout.
    void busy_wait(breaker::Breaker &breaker) {
        struct timespec timeout = {0, 0};  // Zero timeout
        struct kevent events[2];

        while (!!breaker.running()) {
            const int n = kevent(kqueue_fd_, nullptr, 0, events, 2, &timeout);
            if (n > 0 || data_available.load(std::memory_order_acquire)) {
                data_available.store(false, std::memory_order_release);
                return;
            }
            if (n == -1 && errno != EINTR) {
                LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
                return;
            }
        }
    }

    /// @brief High-rate polling with precise sleep.
    void high_rate_wait(breaker::Breaker &breaker) {
        if (timer) {
            timer->wait(breaker);
        } else {
            std::this_thread::sleep_for(std::chrono::microseconds(100));
        }
        data_available.store(false, std::memory_order_release);
    }

    /// @brief Hybrid mode - spin briefly, then block on kqueue.
    void hybrid_wait(breaker::Breaker &breaker) {
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration =
            std::chrono::microseconds(cfg.spin_duration_us);

        // Spin phase - non-blocking kevent checks
        struct timespec timeout = {0, 0};
        struct kevent events[2];

        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;

            const int n = kevent(kqueue_fd_, nullptr, 0, events, 2, &timeout);
            if (n > 0 || data_available.load(std::memory_order_acquire)) {
                data_available.store(false, std::memory_order_release);
                return;
            }
        }

        // Block phase - wait on kqueue with timeout
        timeout.tv_sec = 0;
        timeout.tv_nsec = 10'000'000;  // 10ms max block

        const int n = kevent(kqueue_fd_, nullptr, 0, events, 2, &timeout);
        if (n > 0 || data_available.load(std::memory_order_acquire)) {
            data_available.store(false, std::memory_order_release);
        }
    }

    /// @brief Event-driven mode - block on kqueue indefinitely.
    void event_driven_wait(breaker::Breaker &breaker) {
        struct kevent events[2];

        // Block indefinitely (nullptr timeout)
        const int n = kevent(kqueue_fd_, nullptr, 0, events, 2, nullptr);

        if (n > 0 || data_available.load(std::memory_order_acquire)) {
            data_available.store(false, std::memory_order_release);
            return;
        }

        if (n == -1 && errno != EINTR) {
            LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
        }
    }

    void set_thread_priority(int priority) {
        struct sched_param param;
        param.sched_priority = priority;

        if (pthread_setschedparam(pthread_self(), SCHED_FIFO, &param) != 0) {
            LOG(WARNING) << "[loop] Failed to set SCHED_FIFO priority: "
                         << strerror(errno)
                         << " (may require root)";
        }
    }

    void set_cpu_affinity(int cpu) {
        LOG(WARNING) << "[loop] CPU affinity setting on macOS requires "
                     << "Mach thread APIs (not implemented)";
    }


};

std::unique_ptr<Loop> create() {
    return std::make_unique<DarwinLoop>();
}
}  // namespace arc::runtime::loop
