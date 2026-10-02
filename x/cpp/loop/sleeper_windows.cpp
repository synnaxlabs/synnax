// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <thread>

#include <windows.h>

#include "x/cpp/loop/sleeper.h"

// Windows 10 1803 added the flag. Older SDKs do not define it.
#ifndef CREATE_WAITABLE_TIMER_HIGH_RESOLUTION
#define CREATE_WAITABLE_TIMER_HIGH_RESOLUTION 0x00000002
#endif

namespace x::loop {
struct Sleeper::Impl {
    /// @brief the high-resolution timer, or null before Windows 10 1803.
    const HANDLE timer = CreateWaitableTimerExW(
        nullptr,
        nullptr,
        CREATE_WAITABLE_TIMER_HIGH_RESOLUTION,
        TIMER_ALL_ACCESS
    );

    Impl() = default;
    Impl(const Impl &) = delete;
    Impl &operator=(const Impl &) = delete;

    ~Impl() {
        if (this->timer != nullptr) CloseHandle(this->timer);
    }

    void sleep(const telem::TimeSpan &dur) const {
        // A negative due time is relative, in 100 ns units.
        LARGE_INTEGER due;
        due.QuadPart = -std::max<LONGLONG>(dur.nanoseconds() / 100, 1);
        if (this->timer == nullptr ||
            !SetWaitableTimerEx(this->timer, &due, 0, nullptr, nullptr, nullptr, 0)) {
            std::this_thread::sleep_for(dur.chrono());
            return;
        }
        WaitForSingleObject(this->timer, INFINITE);
    }
};

Sleeper::Sleeper(): impl_(std::make_unique<Impl>()) {}

Sleeper::~Sleeper() = default;

Sleeper::Sleeper(Sleeper &&other) noexcept = default;

Sleeper &Sleeper::operator=(Sleeper &&other) noexcept = default;

void Sleeper::sleep(const telem::TimeSpan &dur) {
    this->impl_->sleep(dur);
}
}
