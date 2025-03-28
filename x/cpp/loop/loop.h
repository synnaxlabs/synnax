// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <chrono>
#include <cmath>
#include <thread>

/// external
#include "glog/logging.h"

/// internal
#include "x/cpp/telem/telem.h"
#include "x/cpp/breaker/breaker.h"

using hs_clock = std::chrono::high_resolution_clock;
using nanos = std::chrono::nanoseconds;

namespace loop {
const telem::TimeSpan HIGH_RES_THRESHOLD = telem::Rate(200).period();
const telem::TimeSpan MEDIUM_RES_THRESHOLD = telem::Rate(20).period();

const telem::TimeSpan RESOLUTION = (100 * telem::MICROSECOND);

/// @brief fine grain sleep function (using Welford's online algorithm)
/// @param dur the duration to sleep for in nanoseconds
inline void precise_sleep(const telem::TimeSpan &dur) {
    const auto end = hs_clock::now() + dur.chrono();
    // static because variance in sleep duration is measured across each call
    // to compute a more accurate sleep time for the machine running the code
    static telem::TimeSpan estimate = RESOLUTION * 10; // overestimate initially
    static telem::TimeSpan mean = RESOLUTION * 10;
    static auto M2 = telem::TimeSpan(0);
    static int64_t count = 1;
    while (dur > estimate) {
        auto start = hs_clock::now();
        if (start >= end) break;
        std::this_thread::sleep_for(RESOLUTION.chrono());
        const auto curr_end = hs_clock::now();
        const auto elapsed = std::chrono::duration_cast<nanos>(curr_end - start).count();
        const telem::TimeSpan delta = elapsed - mean;
        mean += delta / count;
        M2 += delta * (elapsed - mean);
        estimate = mean + std::sqrt((M2 / count).nanoseconds());
        count++;
    }
    // busy wait for the last bit to ensure we sleep for the correct duration
    while (end > hs_clock::now());
}

class Timer {
public:
    Timer() = default;

    explicit Timer(
        const telem::TimeSpan &interval
    ) : interval(interval), last(std::chrono::high_resolution_clock::now()) {
    }

    explicit Timer(
        const telem::Rate &rate
    ) : interval(rate.period()), last(std::chrono::high_resolution_clock::now()) {
    }

    telem::TimeSpan elapsed(const std::chrono::high_resolution_clock::time_point now) {
        if (!last_set) {
            last_set = true;
            return telem::TimeSpan(0);
        }
        const auto elapsed = now - last;
        return telem::TimeSpan(elapsed);
    }

    std::pair<telem::TimeSpan, bool> wait() {
        const auto now = hs_clock::now();
        const auto elapsed = this->elapsed(now);
        if (elapsed > interval) {
            last = now;
            return {telem::TimeSpan(elapsed), false};
        }
        const auto remaining = interval - elapsed;
        if (this->high_rate()) precise_sleep(remaining);
        else std::this_thread::sleep_for(remaining.chrono());
        last = hs_clock::now();
        return {telem::TimeSpan(elapsed), true};
    }

    std::pair<telem::TimeSpan, bool> wait(breaker::Breaker &breaker) {
        const auto now = hs_clock::now();
        const auto elapsed = this->elapsed(now);
        if (elapsed > interval) {
            last = now;
            return {telem::TimeSpan(elapsed), false};
        }
        const auto remaining = interval - elapsed;
        if (this->high_rate()) precise_sleep(remaining);
        else if (this->medium_rate())
            std::this_thread::sleep_for(remaining.chrono());
        else breaker.wait_for(remaining);
        last = hs_clock::now();
        return {telem::TimeSpan(elapsed), true};
    }

private:
    [[nodiscard]] bool high_rate() const { return interval < HIGH_RES_THRESHOLD; }

    [[nodiscard]] bool medium_rate() const {
        return interval < MEDIUM_RES_THRESHOLD;
    }

    telem::TimeSpan interval{};
    bool last_set = false;
    std::chrono::time_point<std::chrono::high_resolution_clock> last;
};

class Gauge {
    size_t count = 0;
    telem::TimeSpan total_duration{0};
    telem::TimeSpan min_duration{std::numeric_limits<int64_t>::max()};
    telem::TimeSpan max_duration{0};
    std::chrono::time_point<std::chrono::high_resolution_clock> curr_start;
    
    // New parameters for logging
    size_t log_every = 0;  // 0 means no logging
    double anomaly_threshold = 0.0;  // percentage threshold for anomaly detection
    std::string name = "gauge";

public:
    // Add constructor with logging parameters
    explicit Gauge(std::string name = "gauge", size_t log_every = 0, double anomaly_threshold = 0.0)
        : log_every(log_every), anomaly_threshold(anomaly_threshold), name(name) {}

    void start() {
        curr_start = std::chrono::high_resolution_clock::now();
    }

    void stop() {
        if (curr_start == std::chrono::time_point<std::chrono::high_resolution_clock>{}) return;
        const auto now = std::chrono::high_resolution_clock::now();
        const auto duration = telem::TimeSpan(std::chrono::duration_cast<nanos>(now - curr_start));
        total_duration += duration;
        min_duration = std::min(min_duration, duration);
        max_duration = std::max(max_duration, duration);
        count++;

        // Log average if log_every is set and we've hit the interval
        if (log_every > 0 && count % log_every == 0) {
            LOG(INFO) << "[" << name << "] average after " << count << " samples: "
                     << average();
        }

        // Check for anomalies if threshold is set
        if (anomaly_threshold > 0.0) {
            const auto curr_avg = average();
            const auto deviation = std::abs((duration.nanoseconds() - curr_avg.nanoseconds()) / 
                                         static_cast<double>(curr_avg.nanoseconds()));
            
            if (deviation > anomaly_threshold) {
                LOG(WARNING) << "[" << name << "] Anomaly detected: Sample duration " << duration
                           << "ns deviates " << (deviation * 100) << "% from average "
                           << curr_avg;
            }
        }
    }

    [[nodiscard]] telem::TimeSpan average() const {
        if (count == 0) return telem::TimeSpan(0);
        return total_duration / count;
    }

    [[nodiscard]] telem::TimeSpan min() const { return min_duration; }
    [[nodiscard]] telem::TimeSpan max() const { return max_duration; }
    [[nodiscard]] size_t iterations() const { return count; }

    void reset() {
        count = 0;
        total_duration = telem::TimeSpan(0);
        min_duration = telem::TimeSpan(std::numeric_limits<int64_t>::max());
        max_duration = telem::TimeSpan(0);
    }
};
}
