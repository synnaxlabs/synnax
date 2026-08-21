// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>

#include "absl/log/log.h"
#include <windows.h>

// timeBeginPeriod/timeEndPeriod from winmm.lib. We declare them manually instead of
// including <timeapi.h> because WIN32_LEAN_AND_MEAN (set by the build) excludes
// multimedia headers and their transitive type dependencies.
extern "C" {
__declspec(dllimport) UINT WINAPI timeBeginPeriod(UINT uPeriod);
__declspec(dllimport) UINT WINAPI timeEndPeriod(UINT uPeriod);
}

#include "x/cpp/loop/waitable_timer.h"

namespace x::loop {
#ifndef CREATE_WAITABLE_TIMER_HIGH_RESOLUTION
#define CREATE_WAITABLE_TIMER_HIGH_RESOLUTION 0x00000002
#endif

/// @brief Windows WaitableTimer uses 100-nanosecond units.
inline const x::telem::TimeSpan WINDOWS_TIMER_UNIT = 100 * x::telem::NANOSECOND;

// Try CREATE_WAITABLE_TIMER_HIGH_RESOLUTION first for sub-millisecond precision
// without global side effects. Falls back to a standard timer with
// timeBeginPeriod(1) on pre-Windows 10 1803 systems. Both use one-shot re-arming
// instead of periodic mode because the periodic lPeriod parameter doesn't benefit
// from the high-resolution mechanism.
WaitableTimer::WaitableTimer() {
    this->timer_event_ = CreateWaitableTimerExW(
        NULL,
        NULL,
        CREATE_WAITABLE_TIMER_HIGH_RESOLUTION,
        TIMER_ALL_ACCESS
    );
    if (this->timer_event_ != NULL) {
        this->high_res_timer_ = true;
        VLOG(1) << "[xloop] using high-resolution waitable timer";
    } else {
        this->timer_event_ = CreateWaitableTimer(NULL, FALSE, NULL);
        if (this->timer_event_ == NULL) {
            this->error_ = x::errors::Error(
                "Failed to create waitable timer: " + std::to_string(GetLastError())
            );
            return;
        }
        timeBeginPeriod(1);
        this->used_time_begin_period_ = true;
        VLOG(1) << "[xloop] using standard waitable timer with "
                << "timeBeginPeriod(1) fallback";
    }
}

WaitableTimer::~WaitableTimer() {
    if (this->timer_event_ != NULL) {
        CancelWaitableTimer(this->timer_event_);
        CloseHandle(this->timer_event_);
        this->timer_event_ = NULL;
    }

    if (this->used_time_begin_period_) {
        timeEndPeriod(1);
        this->used_time_begin_period_ = false;
    }
}

bool WaitableTimer::arm(const telem::TimeSpan &interval) const {
    LARGE_INTEGER due_time;
    const int64_t interval_100ns = interval.nanoseconds() /
                                   WINDOWS_TIMER_UNIT.nanoseconds();
    due_time.QuadPart = -interval_100ns;
    return SetWaitableTimer(this->timer_event_, &due_time, 0, NULL, NULL, FALSE);
}
}
