// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace x::loop {
/// @brief RAII handle to a Windows waitable timer. Prefers the high-resolution timer
/// type; falls back to a standard timer plus timeBeginPeriod(1), undone on destruction.
class WaitableTimer {
public:
    WaitableTimer();
    ~WaitableTimer();

    WaitableTimer(const WaitableTimer &) = delete;
    WaitableTimer &operator=(const WaitableTimer &) = delete;

    /// @brief the construction error. The timer is unusable unless this is NIL.
    [[nodiscard]] const errors::Error &error() const { return this->error_; }

    /// @brief arms the timer to fire once, interval from now.
    /// @param interval the relative due time.
    /// @returns false if arming failed.
    bool arm(const telem::TimeSpan &interval) const;

    /// @brief the native timer HANDLE for use in Windows wait functions.
    [[nodiscard]] void *handle() const { return this->timer_event_; }

private:
    void *timer_event_ = nullptr;
    bool high_res_timer_ = false;
    bool used_time_begin_period_ = false;
    errors::Error error_ = errors::NIL;
};
}
