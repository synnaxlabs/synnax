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

class BaseDarwinLoop : public Loop {
protected:
    Config cfg;
    int kqueue_fd_ = -1;
    bool timer_enabled = false;
    std::atomic<bool> data_available{false};
    std::atomic<bool> running{false};

    explicit BaseDarwinLoop(const Config &config): cfg(config) {
        if (cfg.lock_memory)
            LOG(WARNING) << "[loop] Memory locking not fully supported on macOS";
    }

    xerrors::Error setup_kqueue() {
        kqueue_fd_ = kqueue();
        if (kqueue_fd_ == -1)
            return xerrors::Error(
                "Failed to create kqueue: " + std::string(strerror(errno))
            );

        // Register user event for data notifications
        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, EV_ADD | EV_CLEAR, 0, 0, nullptr);
        if (kevent(kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            close(kqueue_fd_);
            return xerrors::Error(
                "Failed to register user event: " + std::string(strerror(errno))
            );
        }

        return xerrors::NIL;
    }

    xerrors::Error setup_timer() {
        if (cfg.interval.nanoseconds() <= 0) return xerrors::NIL;

        const uint64_t interval_ms = cfg.interval.milliseconds();
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
        if (kevent(kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1)
            return xerrors::Error(
                "Failed to register timer event: " + std::string(strerror(errno))
            );

        timer_enabled = true;
        return xerrors::NIL;
    }

    void apply_thread_config() {
        mach_port_t thread_port = pthread_mach_thread_np(pthread_self());

        // Set thread priority using Mach APIs
        if (cfg.rt_priority > 0) {
            // Use precedence policy instead of time constraint
            // Time constraint policy can throttle threads that exceed their computation
            // budget, which may hurt performance for variable workloads
            thread_precedence_policy_data_t precedence;
            precedence.importance = cfg.rt_priority;

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
                LOG(INFO) << "[loop] Set thread precedence to " << cfg.rt_priority;
            }
        }

        // Set CPU affinity using thread affinity policy
        if (cfg.cpu_affinity >= 0) {
            thread_affinity_policy_data_t affinity_policy;
            affinity_policy.affinity_tag = cfg.cpu_affinity;

            kern_return_t result = thread_policy_set(
                thread_port,
                THREAD_AFFINITY_POLICY,
                reinterpret_cast<thread_policy_t>(&affinity_policy),
                THREAD_AFFINITY_POLICY_COUNT
            );

            if (result != KERN_SUCCESS) {
                LOG(WARNING) << "[loop] Failed to set CPU affinity to "
                             << cfg.cpu_affinity << ": " << mach_error_string(result);
            } else {
                LOG(INFO) << "[loop] Set thread affinity tag to " << cfg.cpu_affinity;
            }
        }
    }

public:
    ~BaseDarwinLoop() override { stop(); }

    void notify_data() override {
        data_available.store(true, std::memory_order_release);
        if (!running) return;

        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);

        if (kevent(kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            LOG(ERROR) << "[loop] Failed to trigger user event: " << strerror(errno);
        }
    }

    void stop() override {
        if (!running) return;

        running = false;

        if (kqueue_fd_ != -1) {
            close(kqueue_fd_);
            kqueue_fd_ = -1;
        }

        timer_enabled = false;
    }
};

class BusyWaitLoop final : public BaseDarwinLoop {
public:
    explicit BusyWaitLoop(const Config &config): BaseDarwinLoop(config) {}

    xerrors::Error start() override {
        if (running) return xerrors::NIL;

        auto err = setup_kqueue();
        if (err) return err;

        if (cfg.interval.nanoseconds() > 0) {
            err = setup_timer();
            if (err) {
                close(kqueue_fd_);
                return err;
            }
        }

        apply_thread_config();
        running = true;

        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!running) return;

        constexpr timespec timeout = {0, 0};
        struct kevent events[2];

        while (breaker.running()) {
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
};

class HighRateLoop final : public BaseDarwinLoop {
    std::unique_ptr<::loop::Timer> timer;

public:
    explicit HighRateLoop(const Config &config): BaseDarwinLoop(config) {}

    xerrors::Error start() override {
        if (running) return xerrors::NIL;
        if (cfg.interval.nanoseconds() > 0)
            timer = std::make_unique<::loop::Timer>(cfg.interval);
        if (auto err = this->setup_kqueue()) return err;
        this->apply_thread_config();
        running = true;
        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!running) return;
        if (timer) {
            timer->wait(breaker);
        } else {
            std::this_thread::sleep_for(std::chrono::microseconds(100));
        }
        data_available.store(false, std::memory_order_release);
    }

    void stop() override {
        timer.reset();
        BaseDarwinLoop::stop();
    }
};

class HybridLoop final : public BaseDarwinLoop {
    std::unique_ptr<::loop::Timer> timer;

public:
    explicit HybridLoop(const Config &config): BaseDarwinLoop(config) {}

    xerrors::Error start() override {
        if (running) return xerrors::NIL;
        if (auto err = setup_kqueue()) return err;

        if (cfg.interval.nanoseconds() > 0)
            timer = std::make_unique<::loop::Timer>(cfg.interval);

        apply_thread_config();
        running = true;

        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!running) return;

        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = cfg.spin_duration.chrono();

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

        timeout.tv_sec = 0;
        timeout.tv_nsec = 10'000'000;

        const int n = kevent(kqueue_fd_, nullptr, 0, events, 2, &timeout);
        if (n > 0 || data_available.load(std::memory_order_acquire)) {
            data_available.store(false, std::memory_order_release);
        }
    }

    void stop() override {
        timer.reset();
        BaseDarwinLoop::stop();
    }
};

class EventDrivenLoop final : public BaseDarwinLoop {
public:
    explicit EventDrivenLoop(const Config &config): BaseDarwinLoop(config) {}

    xerrors::Error start() override {
        if (running) return xerrors::NIL;

        auto err = setup_kqueue();
        if (err) return err;

        if (cfg.interval.nanoseconds() > 0) {
            err = setup_timer();
            if (err) {
                close(kqueue_fd_);
                return err;
            }
        }

        apply_thread_config();
        running = true;

        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!running) return;

        // Check if data is already available (handles early notifications)
        if (data_available.load(std::memory_order_acquire)) {
            data_available.store(false, std::memory_order_release);
            return;
        }

        struct kevent events[2];
        const int n = kevent(kqueue_fd_, nullptr, 0, events, 2, nullptr);
        if (n > 0 || data_available.load(std::memory_order_acquire))
            return data_available.store(false, std::memory_order_release);
        if (n == -1 && errno != EINTR)
            LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
    }
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg) {
    Config adjusted_cfg = cfg;

    // Adjust unsupported modes
    if (adjusted_cfg.mode == ExecutionMode::RT_EVENT) {
        LOG(INFO) << "[loop] RT_EVENT mode not supported on macOS, "
                  << "falling back to HIGH_RATE";
        adjusted_cfg.mode = ExecutionMode::HIGH_RATE;
    }

    std::unique_ptr<Loop> loop;
    switch (adjusted_cfg.mode) {
        case ExecutionMode::BUSY_WAIT:
            LOG(INFO) << "[loop] creating BusyWaitLoop";
            loop = std::make_unique<BusyWaitLoop>(adjusted_cfg);
            break;

        case ExecutionMode::HIGH_RATE:
            LOG(INFO) << "[loop] creating HighRateLoop";
            loop = std::make_unique<HighRateLoop>(adjusted_cfg);
            break;

        case ExecutionMode::HYBRID:
            LOG(INFO) << "[loop] creating HybridLoop";
            loop = std::make_unique<HybridLoop>(adjusted_cfg);
            break;

        case ExecutionMode::EVENT_DRIVEN:
            LOG(INFO) << "[loop] creating EventDrivenLoop";
            loop = std::make_unique<EventDrivenLoop>(adjusted_cfg);
            break;

        case ExecutionMode::RT_EVENT:
        default:
            LOG(INFO) << "[loop] creating EventDrivenLoop (default)";
            loop = std::make_unique<EventDrivenLoop>(adjusted_cfg);
            break;
    }

    if (auto err = loop->start(); err) { return {nullptr, err}; }
    return {std::move(loop), xerrors::NIL};
}

}
