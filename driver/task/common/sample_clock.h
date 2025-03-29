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
    virtual telem::TimeStamp end() = 0;
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

    telem::TimeStamp end() override {
        return telem::TimeStamp::now();
    }
};

using NowFunc = std::function<telem::TimeStamp()>;

/// @brief a sample clock that relies on an external, steady hardware clock to
/// regulate the acquisition rate. Timestamps are interpolated based on a fixed
/// sample rate.
class HardwareTimedSampleClock final : public SampleClock {
    NowFunc now;
    /// @brief the sample rate of the task.
    const telem::Rate sample_rate;
    /// @brief the stream rate of the task.
    const telem::Rate stream_rate;
    /// @brief the number of samples per acquisition loop.
    const size_t samples_per_channel;


    /// @brief track the system time marking the end of the previous acquisition loop.
    telem::TimeStamp prev_system_end = telem::TimeStamp(0);
    /// @brief timestamp of the first sample in the current acquisition loop.
    telem::TimeStamp curr_start_sample_time = telem::TimeStamp(0);

    static constexpr double MAX_INTEGRAL = 1000.0;
    const double k_p = 0.1;
    const double k_i = 0.1;
    const double k_d = 0.0;

    /// @brief the current integral term of the PID controller.
    double integral = 0.0;
    /// @brief the previous error term of the PID controller.
    double prev_error = 0.0;
public:
    explicit HardwareTimedSampleClock(
        const telem::Rate sample_rate,
        const telem::Rate stream_rate,
        const double k_p = 0.1,
        const double k_i = 0.1,
        const double k_d = 0.0,
        const NowFunc &now = telem::TimeStamp::now
    ): sample_rate(sample_rate),
       stream_rate(stream_rate),
       samples_per_channel(sample_rate / stream_rate),
       now(now),
       k_p(k_p),
       k_i(k_i),
       k_d(k_d) {
    }

    void reset() override {
        this->prev_system_end = telem::TimeStamp(0);
        this->curr_start_sample_time = telem::TimeStamp(0);
        this->integral = 0.0;
        this->prev_error = 0.0;
    }

    telem::TimeStamp wait(breaker::Breaker &_) override {
        if (this->curr_start_sample_time == 0) {
            const auto now = this->now();
            this->curr_start_sample_time = now;
            this->prev_system_end = now;
        }
        return this->curr_start_sample_time;
    }

    telem::TimeStamp end() override {
        auto sample_end = this->curr_start_sample_time + this->stream_rate.period();
        const auto system_end = this->now();
        const auto error = (sample_end - system_end).nanoseconds();
        const double dt = (system_end - this->prev_system_end).nanoseconds();
        const double p_term = k_p * error;
        this->integral += error * dt;
        if (this->integral > MAX_INTEGRAL) this->integral = MAX_INTEGRAL;
        if (this->integral < -MAX_INTEGRAL) this->integral = -MAX_INTEGRAL;
        const double i_term = k_i * this->integral;
        const double d_term = k_d * (error - this->prev_error) / dt;
        this->prev_error = error;
        const auto pid_output = p_term + i_term + d_term;
        const auto correction = telem::TimeSpan(pid_output);
        sample_end = sample_end - correction;
        this->prev_system_end = system_end;
        this->curr_start_sample_time = sample_end;
        return sample_end;
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
