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
    Timer() = default;

    explicit  Timer(const synnax::TimeSpan &interval): interval(interval), last(std::chrono::high_resolution_clock::now()) {
    }

    explicit Timer(const synnax::Rate &rate): interval(rate.period()), last(std::chrono::high_resolution_clock::now()) {
        LOG(INFO) << "Timer interval set to " << rate.period() << " ns";
    }

    void sleep(std::chrono::nanoseconds ns){
        auto end = std::chrono::high_resolution_clock::now() + ns;
        while(end > std::chrono::high_resolution_clock::now());
    }

    std::pair<std::chrono::nanoseconds, bool> wait() {
        const auto now = std::chrono::high_resolution_clock::now();
        const auto elapsed = now - last;
        const auto interval_nanos = interval.nanoseconds();
        if (elapsed < interval_nanos) {
            auto remaining = interval_nanos - elapsed;
            std::this_thread::sleep_for(std::chrono::nanoseconds(remaining));
            last = std::chrono::high_resolution_clock::now();
            return {elapsed, true};
        }
        last = now;
        return {elapsed, false};
    }

 
private:
    synnax::TimeSpan interval;
    std::chrono::time_point<std::chrono::high_resolution_clock> last;
};
}