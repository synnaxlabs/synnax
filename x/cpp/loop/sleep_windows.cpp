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
#include "x/cpp/loop/waitable_timer.h"

namespace x::loop {
void sleep_for(const telem::TimeSpan &dur) {
    if (dur <= telem::TimeSpan::ZERO()) return;
    const WaitableTimer timer;
    if (!timer.error() && timer.arm(dur) &&
        WaitForSingleObject(timer.handle(), INFINITE) == WAIT_OBJECT_0)
        return;
    std::this_thread::sleep_for(dur.chrono());
}
}
