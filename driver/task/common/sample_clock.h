// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/telem/series.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"

namespace common {
/// @brief used to regulate the acquisition speed of a task, and provide timing
/// information for generating timestamps.
struct SampleClock {
    virtual ~SampleClock() = default;

    /// @brief resets the sample clock, making it ready for task startup.
    virtual void reset() {
    }

    /// @brief waits for the next acquisition loop to begin, returning the timestamp
    /// of the first sample.
    virtual telem::TimeStamp wait(breaker::Breaker &breaker) = 0;

    /// @brief ends the acquisition loop, interpolating an ending timestamp based
    /// on the number of samples read.
    virtual telem::TimeStamp end(const size_t &n_read) = 0;
};

/// @brief a sample clock that regulates the acquisition rate at the application
/// layer by using a software timer.
class SoftwareTimedSampleClock final : public SampleClock {
    /// @brief the timer used to regulate the acquisition rate.
    loop::Timer timer;

public:
    explicit SoftwareTimedSampleClock(const telem::Rate &stream_rate):
        timer(stream_rate) {
    }

    telem::TimeStamp wait(breaker::Breaker &breaker) override {
        const auto start = telem::TimeStamp::now();
        this->timer.wait(breaker);
        return start;
    }

    telem::TimeStamp end(const size_t &n_read) override {
        return telem::TimeStamp::now();
    }
};

/// @brief a sample clock that relies on an external, steady hardware clock to
/// regulate the acquisition rate. Timestamps are interpolated based on a fixed
/// sample rate.
class HardwareTimedSampleClock final : public SampleClock {
    /// @brief the sample rate of the task.
    const telem::Rate sample_rate;
    /// @brief the stream rate of the task.
    const telem::Rate stream_rate;
    /// @brief the number of samples per acquisition loop.
    const size_t samples_per_channel;
    size_t iterations = 0;
    /// @brief the high water-mark for the next acquisition loop.
    telem::TimeStamp high_water = telem::TimeStamp(0);

    // PID control variables
    double integral = 0.0;
    double prev_error = 0.0;
    
    // PID constants
    static constexpr double Kp = 0.01;  // Proportional gain
    static constexpr double Ki = 0.001; // Integral gain
    static constexpr double Kd = 0.005; // Derivative gain
    
    // Anti-windup limit for integral term
    static constexpr double MAX_INTEGRAL = 1000.0;

public:
    explicit HardwareTimedSampleClock(
        const telem::Rate sample_rate,
        const telem::Rate stream_rate
    ): sample_rate(sample_rate),
       stream_rate(stream_rate),
       samples_per_channel(sample_rate / stream_rate) {
    }

    void reset() override {
        this->high_water = telem::TimeStamp(0);
        this->iterations = 0;
        this->integral = 0.0;
        this->prev_error = 0.0;
    }

    telem::TimeStamp wait(breaker::Breaker &_) override {
        if (this->high_water == 0)
            this->high_water = telem::TimeStamp::now();
        return this->high_water;
    }

    telem::TimeStamp end(const size_t &_) override {
        auto end = this->high_water + (this->samples_per_channel - 1) * this->sample_rate.period();
        const auto system_end = telem::TimeStamp::now();
        this->iterations++;

        // Calculate error (process variable)
        const auto error = (system_end - end).nanoseconds();

        // PID control calculation
        // Proportional term
        double p_term = Kp * error;
        
        // Integral term with anti-windup
        this->integral += error;
        if (this->integral > MAX_INTEGRAL) this->integral = MAX_INTEGRAL;
        if (this->integral < -MAX_INTEGRAL) this->integral = -MAX_INTEGRAL;
        double i_term = Ki * this->integral;
        
        // Derivative term
        double d_term = Kd * (error - this->prev_error);
        this->prev_error = error;

        // Calculate correction using PID output
        const auto pid_output = p_term + i_term + d_term;
        const auto correction = telem::TimeStamp(pid_output);
        if (end + correction > this->high_water)
            end += correction;
        else
            LOG(WARNING) << "[sample_clock] correction would result in out of order timestamps"
                         << "correction: " << telem::TimeSpan(correction.nanoseconds()) << "\n"
                         << "high_water: " << this->high_water << "\n"
                         << "end: " << end << "\n"
                         << "system clock: " << system_end;
        this->high_water = end + this->sample_rate.period();
        return end;
    }
};


inline void generate_index_data(
    const synnax::Frame &f,
    const std::set<synnax::ChannelKey> &index_keys,
    const telem::TimeStamp &start,
    const telem::TimeStamp &end,
    const size_t n_read,
    const bool inclusive = false
) {
    if (index_keys.empty()) return;
    auto index_data = telem::Series::linspace(start, end, n_read, inclusive);
    if (index_keys.size() == 1)
        f.emplace(*index_keys.begin(), std::move(index_data));
    else
        for (const auto &idx: index_keys)
            f.emplace(idx, std::move(index_data.deep_copy()));
}
}
