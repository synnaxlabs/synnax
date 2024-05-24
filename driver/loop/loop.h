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

    void exactSleep(std::chrono::nanoseconds ns){ 
        auto end = std::chrono::high_resolution_clock::now() + ns;
        uint64_t resolution = 100000; // 0.1 ms 
        uint64_t nanoseconds = ns.count();
        // estimate for 0.1 millseconds  (100000 nanoseconds)
        static uint64_t estimate = resolution*10; // overestimate innitially
        static uint64_t mean = resolution*10;
        static uint64_t M2 = 0;
        static uint64_t count = 1;

        while(nanoseconds > estimate){
            // sleep for specified resolution
            auto start = std::chrono::high_resolution_clock::now();
            sleep(std::chrono::nanoseconds(resolution));
            auto end = std::chrono::high_resolution_clock::now();

            // get actual elapsed time
            auto elapsed = std::chrono::duration_cast<std::chrono::nanoseconds>(end - start).count();
            uint64_t delta = elapsed - mean;
            mean += delta / count;
            M2 += delta * (elapsed - mean);
            estimate = mean + 1* std::sqrt(M2 / count);
            count++;
        }
        while(end > std::chrono::high_resolution_clock::now());
    }

    std::pair<std::chrono::nanoseconds, bool> wait() {
        const auto now = std::chrono::high_resolution_clock::now();
        const auto elapsed = now - last;
        const auto interval_nanos = interval.nanoseconds();
        if (elapsed < interval_nanos) {
            auto remaining = interval_nanos - elapsed;
            // std::this_thread::sleep_for(std::chrono::nanoseconds(remaining));
            this->exactSleep(std::chrono::nanoseconds(remaining));
            last = std::chrono::high_resolution_clock::now();
            return {elapsed, true};
        } else {
            last = now;
            LOG(WARNING) << "Timer interval exceeded by " << (elapsed - interval_nanos)/1e6 << " ms";
            return {elapsed, false};
        }
    }

   

 
private:
    synnax::TimeSpan interval;
    std::chrono::time_point<std::chrono::high_resolution_clock> last;
};
}