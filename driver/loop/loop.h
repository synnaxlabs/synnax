// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <chrono>
#include <thread>
#include "client/cpp/synnax.h"
#include "glog/logging.h"


namespace loop {
class Timer {
public:
    explicit  Timer(const synnax::TimeSpan &interval): interval(interval), last(std::chrono::high_resolution_clock::now()) {
    }

    void wait() {
        const auto now = std::chrono::high_resolution_clock::now();
        const auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(now - last);
        const auto interval_nanos = interval.nanoseconds();
        if (elapsed < interval_nanos) {
            const auto remaining = interval_nanos - elapsed;
            std::this_thread::sleep_for(remaining);
            last = now + remaining;
        } else last = now;
    }
private:
    synnax::TimeSpan interval;
    std::chrono::time_point<std::chrono::high_resolution_clock> last;
};
}