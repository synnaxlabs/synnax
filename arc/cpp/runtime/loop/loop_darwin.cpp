// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <atomic>
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
    bool running = false;

    explicit BaseDarwinLoop(const Config &config): cfg(config) {
        if (cfg.rt_priority > 0)
            LOG(WARNING) << "[loop] RT priority support is limited on macOS";
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
        if (cfg.rt_priority > 0) {
            struct sched_param param;
            param.sched_priority = cfg.rt_priority;
            if (pthread_setschedparam(pthread_self(), SCHED_FIFO, &param) != 0) {
                LOG(WARNING) << "[loop] Failed to set SCHED_FIFO priority: "
                             << strerror(errno) << " (may require root)";
            }
        }
        if (cfg.cpu_affinity >= 0) {
            LOG(WARNING) << "[loop] CPU affinity setting on macOS requires "
                         << "Mach thread APIs (not implemented)";
        }
    }

public:
    ~BaseDarwinLoop() override { stop(); }

    void notify_data() override {
        if (!running) return;

        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);

        if (kevent(kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            LOG(ERROR) << "[loop] Failed to trigger user event: " << strerror(errno);
        }

        data_available.store(true, std::memory_order_release);
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
        data_available.store(false, std::memory_order_release);

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
        data_available.store(false, std::memory_order_release);
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

// ============================================================================
// HYBRID Implementation
// ============================================================================
class HybridLoop final : public BaseDarwinLoop {
    std::unique_ptr<::loop::Timer> timer;

public:
    explicit HybridLoop(const Config &config): BaseDarwinLoop(config) {}

    xerrors::Error start() override {
        if (running) return xerrors::NIL;

        auto err = setup_kqueue();
        if (err) return err;

        if (cfg.interval.nanoseconds() > 0) {
            timer = std::make_unique<::loop::Timer>(cfg.interval);
        }

        apply_thread_config();
        running = true;
        data_available.store(false, std::memory_order_release);

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

        // Block phase - wait on kqueue with timeout
        timeout.tv_sec = 0;
        timeout.tv_nsec = 10'000'000; // 10ms max block

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
        data_available.store(false, std::memory_order_release);

        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!running) return;
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

    switch (adjusted_cfg.mode) {
        case ExecutionMode::BUSY_WAIT:
            return {std::make_unique<BusyWaitLoop>(adjusted_cfg), xerrors::NIL};

        case ExecutionMode::HIGH_RATE:
            return {std::make_unique<HighRateLoop>(adjusted_cfg), xerrors::NIL};

        case ExecutionMode::HYBRID:
            return {std::make_unique<HybridLoop>(adjusted_cfg), xerrors::NIL};

        case ExecutionMode::EVENT_DRIVEN:
        case ExecutionMode::RT_EVENT:
        default:
            return {std::make_unique<EventDrivenLoop>(adjusted_cfg), xerrors::NIL};
    }
}

}
