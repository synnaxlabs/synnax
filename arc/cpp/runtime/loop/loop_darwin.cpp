// Copyright 2025 Synnax Labs, Inc.
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
#include <mach/mach.h>
#include <mach/thread_policy.h>
#include <sys/event.h>
#include <sys/time.h>
#include <sys/types.h>
#include <unistd.h>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

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

    ~DarwinLoop() override { this->stop(); }

    void notify_data() override {
        this->data_available_.store(true, std::memory_order_release);
        if (!this->running_.load(std::memory_order_acquire)) return;

        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);

        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            LOG(ERROR) << "[loop] Failed to trigger user event: " << strerror(errno);
        }
    }

    void wait(breaker::Breaker &breaker) override {
        if (!this->running_.load(std::memory_order_acquire)) return;

        switch (this->config_.mode) {
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
                // RT_EVENT falls back to HIGH_RATE on macOS (no true RT kernel)
                this->high_rate_wait(breaker);
                break;
            case ExecutionMode::EVENT_DRIVEN:
                this->event_driven_wait(breaker);
                break;
        }
    }

    xerrors::Error start() override {
        if (this->running_.load(std::memory_order_acquire)) return xerrors::NIL;

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

            if (use_software_timer) {
                // Use software timer for sub-millisecond precision
                this->timer_ = std::make_unique<::loop::Timer>(this->config_.interval);
            } else {
                // Use kqueue timer for EVENT_DRIVEN/HYBRID/BUSY_WAIT (ms precision OK)
                if (auto err = this->setup_kqueue_timer(); err) {
                    close(this->kqueue_fd_);
                    return err;
                }
            }
        }

        this->apply_thread_config();
        this->running_.store(true, std::memory_order_release);

        return xerrors::NIL;
    }

    void stop() override {
        if (!this->running_.load(std::memory_order_acquire)) return;

        this->running_.store(false, std::memory_order_release);

        // Wake up any blocked kevent() calls before closing the fd
        if (this->kqueue_fd_ != -1) {
            struct kevent kev;
            EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);
            kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr);
        }

        this->timer_.reset();

        if (this->kqueue_fd_ != -1) {
            close(this->kqueue_fd_);
            this->kqueue_fd_ = -1;
        }

        this->kqueue_timer_enabled_ = false;
    }

    uint64_t watch(notify::Notifier &notifier) override {
        const int fd = notifier.fd();
        if (fd == -1) return 0;
        if (this->kqueue_fd_ == -1) return 0;

        struct kevent kev;
        EV_SET(&kev, fd, EVFILT_READ, EV_ADD | EV_CLEAR, 0, 0, nullptr);

        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            LOG(ERROR) << "[loop] Failed to watch notifier fd " << fd << ": "
                       << strerror(errno);
            return 0;
        }

        return static_cast<uint64_t>(fd);
    }

    void unwatch(const uint64_t handle) override {
        if (handle == 0 || this->kqueue_fd_ == -1) return;

        const int fd = static_cast<int>(handle);
        struct kevent kev;
        EV_SET(&kev, fd, EVFILT_READ, EV_DELETE, 0, 0, nullptr);

        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            LOG(WARNING) << "[loop] Failed to unwatch fd " << fd << ": "
                         << strerror(errno);
        }
    }

private:
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
            interval_ms > 0 ? interval_ms : 1,
            nullptr
        );
        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1)
            return xerrors::Error(
                "Failed to register timer event: " + std::string(strerror(errno))
            );

        this->kqueue_timer_enabled_ = true;
        return xerrors::NIL;
    }

    void apply_thread_config() {
        mach_port_t thread_port = pthread_mach_thread_np(pthread_self());

        if (this->config_.rt_priority > 0) {
            thread_precedence_policy_data_t precedence;
            precedence.importance = this->config_.rt_priority;

            kern_return_t result = thread_policy_set(
                thread_port,
                THREAD_PRECEDENCE_POLICY,
                reinterpret_cast<thread_policy_t>(&precedence),
                THREAD_PRECEDENCE_POLICY_COUNT
            );

            if (result != KERN_SUCCESS) {
                LOG(WARNING) << "[loop] Failed to set thread precedence: "
                             << mach_error_string(result);
            } else {
                LOG(INFO) << "[loop] Set thread precedence to "
                          << this->config_.rt_priority;
            }
        }

        if (this->config_.cpu_affinity >= 0) {
            thread_affinity_policy_data_t affinity_policy;
            affinity_policy.affinity_tag = this->config_.cpu_affinity;

            kern_return_t result = thread_policy_set(
                thread_port,
                THREAD_AFFINITY_POLICY,
                reinterpret_cast<thread_policy_t>(&affinity_policy),
                THREAD_AFFINITY_POLICY_COUNT
            );

            if (result != KERN_SUCCESS) {
                LOG(WARNING) << "[loop] Failed to set CPU affinity to "
                             << this->config_.cpu_affinity << ": "
                             << mach_error_string(result);
            } else {
                LOG(INFO) << "[loop] Set thread affinity tag to "
                          << this->config_.cpu_affinity;
            }
        }
    }

    /// @brief BUSY_WAIT: Non-blocking kqueue poll in tight loop.
    void busy_wait(breaker::Breaker &breaker) {
        constexpr timespec timeout = {0, 0};
        struct kevent events[8];

        while (breaker.running()) {
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n > 0 || this->data_available_.load(std::memory_order_acquire)) {
                this->data_available_.store(false, std::memory_order_release);
                return;
            }
            // Don't log error during shutdown (fd closed is expected)
            if (n == -1 && errno != EINTR && errno != EBADF) {
                LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
                return;
            }
        }
    }

    /// @brief HIGH_RATE: Precise software timer + non-blocking kqueue drain.
    void high_rate_wait(breaker::Breaker &breaker) {
        // Wait on precise software timer
        if (this->timer_) {
            this->timer_->wait(breaker);
        } else {
            std::this_thread::sleep_for(timing::HIGH_RATE_POLL_INTERVAL.chrono());
        }

        // Non-blocking drain of kqueue events that arrived during timer wait
        constexpr timespec timeout = {0, 0};
        struct kevent events[8];
        kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);

        this->data_available_.store(false, std::memory_order_release);
    }

    /// @brief HYBRID: Spin for configured duration, then block with timeout.
    void hybrid_wait(breaker::Breaker &breaker) {
        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = this->config_.spin_duration.chrono();

        struct timespec timeout = {0, 0};
        struct kevent events[8];

        // Spin phase: non-blocking poll
        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;

            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
            if (n > 0 || this->data_available_.load(std::memory_order_acquire)) {
                this->data_available_.store(false, std::memory_order_release);
                return;
            }
        }

        // Block phase: wait with timeout
        const auto block_timeout_ns = timing::HYBRID_BLOCK_TIMEOUT.nanoseconds();
        timeout.tv_sec = 0;
        timeout.tv_nsec = block_timeout_ns;

        const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, &timeout);
        if (n > 0 || this->data_available_.load(std::memory_order_acquire)) {
            this->data_available_.store(false, std::memory_order_release);
        }
    }

    /// @brief EVENT_DRIVEN: Block indefinitely on kqueue events.
    void event_driven_wait(breaker::Breaker &breaker) {
        // Fast path: check if data already available or shutting down
        if (this->data_available_.load(std::memory_order_acquire)) {
            this->data_available_.store(false, std::memory_order_release);
            return;
        }
        if (!this->running_.load(std::memory_order_acquire)) return;

        struct kevent events[8];
        const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 8, nullptr);

        // Check if we're shutting down (user event triggered by stop())
        if (!this->running_.load(std::memory_order_acquire)) return;

        if (n > 0 || this->data_available_.load(std::memory_order_acquire))
            this->data_available_.store(false, std::memory_order_release);
        if (n == -1 && errno != EINTR)
            LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
    }

    Config config_;
    int kqueue_fd_ = -1;
    bool kqueue_timer_enabled_ = false;
    std::unique_ptr<::loop::Timer> timer_;
    std::atomic<bool> data_available_{false};
    std::atomic<bool> running_{false};
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<DarwinLoop>(cfg);
    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), xerrors::NIL};
}

}
