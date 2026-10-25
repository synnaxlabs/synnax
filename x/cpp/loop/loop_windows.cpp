// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>

#include <windows.h>

#include "x/cpp/loop/loop.h"

// Windows 10 1803 added the flag. Older SDKs do not define it.
#ifndef CREATE_WAITABLE_TIMER_HIGH_RESOLUTION
#define CREATE_WAITABLE_TIMER_HIGH_RESOLUTION 0x00000002
#endif

namespace x::loop {
struct Timer::Impl {
    /// @brief the high-resolution timer, or null before Windows 10 1803.
    HANDLE timer;
};

Timer::ImplPtr Timer::make_impl() {
    return {
        new Impl{CreateWaitableTimerExW(
            nullptr,
            nullptr,
            CREATE_WAITABLE_TIMER_HIGH_RESOLUTION,
            TIMER_ALL_ACCESS
        )},
        [](Impl *impl) {
            if (impl->timer != nullptr) CloseHandle(impl->timer);
            delete impl;
        },
    };
}

void Timer::step() {
    // A negative due time is relative, in 100 ns units.
    LARGE_INTEGER due;
    due.QuadPart = -RESOLUTION.nanoseconds() / 100;
    const HANDLE timer = this->impl_->timer;
    if (timer == nullptr ||
        !SetWaitableTimerEx(timer, &due, 0, nullptr, nullptr, nullptr, 0)) {
        std::this_thread::sleep_for(RESOLUTION.chrono());
        return;
    }
    WaitForSingleObject(timer, INFINITE);
}
}
