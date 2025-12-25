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
        if (this->cfg.lock_memory)
            LOG(WARNING) << "[loop] Memory locking not fully supported on macOS";
    }

    xerrors::Error setup_kqueue() {
        this->kqueue_fd_ = kqueue();
        if (this->kqueue_fd_ == -1)
            return xerrors::Error(
                "Failed to create kqueue: " + std::string(strerror(errno))
            );

        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, EV_ADD | EV_CLEAR, 0, 0, nullptr);
        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            close(this->kqueue_fd_);
            return xerrors::Error(
                "Failed to register user event: " + std::string(strerror(errno))
            );
        }

        return xerrors::NIL;
    }

    xerrors::Error setup_timer() {
        if (this->cfg.interval.nanoseconds() <= 0) return xerrors::NIL;

        const uint64_t interval_ms = this->cfg.interval.milliseconds();
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

        this->timer_enabled = true;
        return xerrors::NIL;
    }

    void apply_thread_config() {
        mach_port_t thread_port = pthread_mach_thread_np(pthread_self());

        if (this->cfg.rt_priority > 0) {
            thread_precedence_policy_data_t precedence;
            precedence.importance = this->cfg.rt_priority;

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
                          << this->cfg.rt_priority;
            }
        }

        if (this->cfg.cpu_affinity >= 0) {
            thread_affinity_policy_data_t affinity_policy;
            affinity_policy.affinity_tag = this->cfg.cpu_affinity;

            kern_return_t result = thread_policy_set(
                thread_port,
                THREAD_AFFINITY_POLICY,
                reinterpret_cast<thread_policy_t>(&affinity_policy),
                THREAD_AFFINITY_POLICY_COUNT
            );

            if (result != KERN_SUCCESS) {
                LOG(WARNING) << "[loop] Failed to set CPU affinity to "
                             << this->cfg.cpu_affinity << ": "
                             << mach_error_string(result);
            } else {
                LOG(INFO) << "[loop] Set thread affinity tag to "
                          << this->cfg.cpu_affinity;
            }
        }
    }

public:
    ~BaseDarwinLoop() override { this->stop(); }

    void notify_data() override {
        this->data_available.store(true, std::memory_order_release);
        if (!this->running) return;

        struct kevent kev;
        EV_SET(&kev, USER_EVENT_IDENT, EVFILT_USER, 0, NOTE_TRIGGER, 0, nullptr);

        if (kevent(this->kqueue_fd_, &kev, 1, nullptr, 0, nullptr) == -1) {
            LOG(ERROR) << "[loop] Failed to trigger user event: " << strerror(errno);
        }
    }

    void stop() override {
        if (!this->running) return;

        this->running = false;

        if (this->kqueue_fd_ != -1) {
            close(this->kqueue_fd_);
            this->kqueue_fd_ = -1;
        }

        this->timer_enabled = false;
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
};

class BusyWaitLoop final : public BaseDarwinLoop {
public:
    explicit BusyWaitLoop(const Config &config): BaseDarwinLoop(config) {}

    xerrors::Error start() override {
        if (this->running) return xerrors::NIL;

        auto err = this->setup_kqueue();
        if (err) return err;

        if (this->cfg.interval.nanoseconds() > 0) {
            err = this->setup_timer();
            if (err) {
                close(this->kqueue_fd_);
                return err;
            }
        }

        this->apply_thread_config();
        this->running = true;

        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!this->running) return;

        constexpr timespec timeout = {0, 0};
        struct kevent events[2];

        while (breaker.running()) {
            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 2, &timeout);
            if (n > 0 || this->data_available.load(std::memory_order_acquire)) {
                this->data_available.store(false, std::memory_order_release);
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
        if (this->running) return xerrors::NIL;
        if (this->cfg.interval.nanoseconds() > 0)
            this->timer = std::make_unique<::loop::Timer>(this->cfg.interval);
        if (auto err = this->setup_kqueue()) return err;
        this->apply_thread_config();
        this->running = true;
        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!this->running) return;
        if (this->timer) {
            this->timer->wait(breaker);
        } else {
            std::this_thread::sleep_for(std::chrono::microseconds(100));
        }
        this->data_available.store(false, std::memory_order_release);
    }

    void stop() override {
        this->timer.reset();
        BaseDarwinLoop::stop();
    }
};

class HybridLoop final : public BaseDarwinLoop {
    std::unique_ptr<::loop::Timer> timer;

public:
    explicit HybridLoop(const Config &config): BaseDarwinLoop(config) {}

    xerrors::Error start() override {
        if (this->running) return xerrors::NIL;
        if (auto err = this->setup_kqueue()) return err;

        if (this->cfg.interval.nanoseconds() > 0)
            this->timer = std::make_unique<::loop::Timer>(this->cfg.interval);

        this->apply_thread_config();
        this->running = true;

        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!this->running) return;

        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = this->cfg.spin_duration.chrono();

        struct timespec timeout = {0, 0};
        struct kevent events[2];

        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;

            const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 2, &timeout);
            if (n > 0 || this->data_available.load(std::memory_order_acquire)) {
                this->data_available.store(false, std::memory_order_release);
                return;
            }
        }

        timeout.tv_sec = 0;
        timeout.tv_nsec = 10'000'000;

        const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 2, &timeout);
        if (n > 0 || this->data_available.load(std::memory_order_acquire)) {
            this->data_available.store(false, std::memory_order_release);
        }
    }

    void stop() override {
        this->timer.reset();
        BaseDarwinLoop::stop();
    }
};

class EventDrivenLoop final : public BaseDarwinLoop {
public:
    explicit EventDrivenLoop(const Config &config): BaseDarwinLoop(config) {}

    xerrors::Error start() override {
        if (this->running) return xerrors::NIL;

        auto err = this->setup_kqueue();
        if (err) return err;

        if (this->cfg.interval.nanoseconds() > 0) {
            err = this->setup_timer();
            if (err) {
                close(this->kqueue_fd_);
                return err;
            }
        }

        this->apply_thread_config();
        this->running = true;

        return xerrors::NIL;
    }

    void wait(breaker::Breaker &breaker) override {
        if (!this->running) return;

        if (this->data_available.load(std::memory_order_acquire)) {
            this->data_available.store(false, std::memory_order_release);
            return;
        }

        struct kevent events[2];
        const int n = kevent(this->kqueue_fd_, nullptr, 0, events, 2, nullptr);
        if (n > 0 || this->data_available.load(std::memory_order_acquire))
            return this->data_available.store(false, std::memory_order_release);
        if (n == -1 && errno != EINTR)
            LOG(ERROR) << "[loop] kevent error: " << strerror(errno);
    }
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg) {
    Config adjusted_cfg = cfg;

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

    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), xerrors::NIL};
}

}
