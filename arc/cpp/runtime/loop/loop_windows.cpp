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
#include <windows.h>

#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xthread/rt.h"

#include "arc/cpp/runtime/loop/loop.h"

namespace arc::runtime::loop {

class WindowsLoop final : public Loop {
    static constexpr DWORD MAX_HANDLES = MAXIMUM_WAIT_OBJECTS;

public:
    explicit WindowsLoop(const Config &config): config_(config) {
        if (this->config_.lock_memory) {
            LOG(WARNING) << "[loop] Memory locking on Windows requires "
                         << "VirtualLock API (not implemented)";
        }
    }

    ~WindowsLoop() override { this->close_handles(); }

    void wait(breaker::Breaker &breaker) override {
        if (this->wake_event_ == NULL) return;

        switch (this->config_.mode) {
            case ExecutionMode::BUSY_WAIT:
                this->busy_wait(breaker);
                break;
            case ExecutionMode::HIGH_RATE:
                this->high_rate_wait(breaker);
                break;
            case ExecutionMode::RT_EVENT:
                this->event_driven_wait(false);
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
        if (this->wake_event_ != NULL) return xerrors::NIL;

        this->wake_event_ = CreateEvent(NULL, FALSE, FALSE, NULL);
        if (this->wake_event_ == NULL) {
            return xerrors::Error(
                "Failed to create wake event: " + std::to_string(GetLastError())
            );
        }

        if (this->config_.interval.nanoseconds() > 0) {
            if (this->config_.mode == ExecutionMode::HIGH_RATE) {
                // HIGH_RATE uses precise software timer
                this->timer_ = std::make_unique<::loop::Timer>(this->config_.interval);
            } else {
                // Other modes use WaitableTimer
                this->timer_event_ = CreateWaitableTimer(NULL, FALSE, NULL);
                if (this->timer_event_ == NULL) {
                    CloseHandle(this->wake_event_);
                    return xerrors::Error(
                        "Failed to create waitable timer: " +
                        std::to_string(GetLastError())
                    );
                }

                LARGE_INTEGER due_time;
                const int64_t interval_100ns = this->config_.interval.nanoseconds() /
                                               timing::WINDOWS_TIMER_UNIT.nanoseconds();
                due_time.QuadPart = -interval_100ns;

                const LONG period_ms = static_cast<LONG>(
                    this->config_.interval.nanoseconds() /
                    telem::MILLISECOND.nanoseconds()
                );

                if (!SetWaitableTimer(
                        this->timer_event_,
                        &due_time,
                        period_ms,
                        NULL,
                        NULL,
                        FALSE
                    )) {
                    CloseHandle(this->timer_event_);
                    CloseHandle(this->wake_event_);
                    return xerrors::Error(
                        "Failed to set waitable timer: " +
                        std::to_string(GetLastError())
                    );
                }

                this->timer_enabled_ = true;
            }
        }

        auto rt_cfg = this->config_.rt();
        rt_cfg.use_mmcss = true;
        if (auto err = xthread::apply_rt_config(rt_cfg); err)
            LOG(WARNING) << "[loop] Failed to apply RT config: " << err.message();

        return xerrors::NIL;
    }

    void wake() override {
        if (this->wake_event_ == NULL) return;
        SetEvent(this->wake_event_);
    }

    bool watch(notify::Notifier &notifier) override {
        auto *handle = static_cast<HANDLE>(notifier.native_handle());
        if (handle == nullptr) {
            LOG(ERROR) << "[loop] Notifier has no native handle";
            return false;
        }
        if (this->watched_handle_ != NULL) {
            LOG(ERROR) << "[loop] Only one external notifier can be watched";
            return false;
        }
        this->watched_handle_ = handle;
        return true;
    }

private:
    void close_handles() {
        this->timer_.reset();

        if (this->timer_event_ != NULL) {
            CancelWaitableTimer(this->timer_event_);
            CloseHandle(this->timer_event_);
            this->timer_event_ = NULL;
        }

        if (this->wake_event_ != NULL) {
            CloseHandle(this->wake_event_);
            this->wake_event_ = NULL;
        }

        this->timer_enabled_ = false;
    }

    void busy_wait(breaker::Breaker &breaker) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);
        if (count == 0) return;

        while (breaker.running()) {
            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);
            if (result < WAIT_OBJECT_0 + count) return;
            if (result == WAIT_FAILED) {
                LOG(ERROR) << "[loop] WaitForMultipleObjects failed: "
                           << GetLastError();
                return;
            }
        }
    }

    void high_rate_wait(breaker::Breaker &breaker) { this->timer_->wait(breaker); }

    void event_driven_wait(bool blocking) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);
        if (count == 0) return;

        // Use timeout to ensure we periodically check breaker.running()
        // in the caller's loop.
        const DWORD timeout_ms = blocking
                                   ? static_cast<DWORD>(
                                         timing::EVENT_DRIVEN_TIMEOUT.milliseconds()
                                     )
                                   : static_cast<DWORD>(
                                         timing::HYBRID_BLOCK_TIMEOUT.milliseconds()
                                     );

        const DWORD result = WaitForMultipleObjects(count, handles, FALSE, timeout_ms);
        if (result == WAIT_FAILED)
            LOG(ERROR) << "[loop] WaitForMultipleObjects failed: " << GetLastError();
    }

    void hybrid_wait(breaker::Breaker &breaker) {
        HANDLE handles[3];
        const DWORD count = this->build_handles(handles);
        if (count == 0) return;

        const auto spin_start = std::chrono::steady_clock::now();
        const auto spin_duration = std::chrono::nanoseconds(
            this->config_.spin_duration.nanoseconds()
        );

        while (std::chrono::steady_clock::now() - spin_start < spin_duration) {
            if (!breaker.running()) return;

            const DWORD result = WaitForMultipleObjects(count, handles, FALSE, 0);
            if (result < WAIT_OBJECT_0 + count) return;
        }

        const DWORD timeout_ms = static_cast<DWORD>(
            timing::HYBRID_BLOCK_TIMEOUT.milliseconds()
        );
        WaitForMultipleObjects(count, handles, FALSE, timeout_ms);
    }

    DWORD build_handles(HANDLE *handles) const {
        DWORD count = 0;
        if (this->wake_event_ != NULL) handles[count++] = this->wake_event_;
        if (this->watched_handle_ != NULL) handles[count++] = this->watched_handle_;
        if (this->timer_enabled_) handles[count++] = this->timer_event_;
        return count;
    }

    Config config_;
    HANDLE wake_event_ = NULL;
    HANDLE timer_event_ = NULL;
    HANDLE watched_handle_ = NULL;
    bool timer_enabled_ = false;
    std::unique_ptr<::loop::Timer> timer_;
};

std::pair<std::unique_ptr<Loop>, xerrors::Error> create(const Config &cfg) {
    auto loop = std::make_unique<WindowsLoop>(cfg);
    if (auto err = loop->start(); err) return {nullptr, err};
    return {std::move(loop), xerrors::NIL};
}

}
