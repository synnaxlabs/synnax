// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <utility>

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/telem/series.h"

namespace common {
/// @brief used to regulate the acquisition speed of a task, and provide timing
/// information for generating timestamps.
struct SampleClock {
    virtual ~SampleClock() = default;

    /// @brief resets the sample clock, making it ready for task startup.
    virtual void reset() {}

    /// @brief waits for the next acquisition loop to begin, returning the timestamp
    /// of the first sample.
    virtual telem::TimeStamp wait(breaker::Breaker &breaker) = 0;

    /// @brief ends the acquisition loop, interpolating an ending timestamp based
    /// on the number of samples read.
    virtual telem::TimeStamp end() = 0;
};

/// @brief common timing options for all tasks.
struct TimingConfig {
    /// @brief whether to automatically correct clock skew in hardware timed sample
    /// clocks.
    bool correct_skew = true;

    template<typename Parser>
    void override(Parser &p) {
        this->correct_skew = p.field("correct_skew", this->correct_skew);
    }

    friend std::ostream &operator<<(std::ostream &os, const TimingConfig &cfg) {
        os << "  " << xlog::shale() << "clock skew correction" << xlog::reset() << ": "
           << (cfg.correct_skew ? "enabled" : "disabled");
        return os;
    }
};

/// @brief a sample clock that regulates the acquisition rate at the application
/// layer by using a software timer.
class SoftwareTimedSampleClock final : public SampleClock {
    /// @brief the timer used to regulate the acquisition rate.
    loop::Timer timer;

public:
    explicit SoftwareTimedSampleClock(const telem::Rate &stream_rate):
        timer(stream_rate) {}

    telem::TimeStamp wait(breaker::Breaker &breaker) override {
        const auto start = telem::TimeStamp::now();
        this->timer.wait(breaker);
        return start;
    }

    telem::TimeStamp end() override { return telem::TimeStamp::now(); }
};

struct HardwareTimedSampleClockConfig {
    /// @brief allows the sample clock to use a custom time function for testing.
    telem::NowFunc now = telem::TimeStamp::now;
    /// @brief the sample rate of the task.
    telem::Rate sample_rate, stream_rate;
    /// @brief the proportional, integral, and derivative gains of the PID
    /// controller. See: https://en.wikipedia.org/wiki/PID_controller
    ///
    /// The PID controller implements the following equation:
    /// u(t) = Kp * e(t) + Ki * ∫e(t)dt + Kd * de/dt
    ///
    /// where:
    /// - e(t) = expected_end_time - system_end_time
    ///   (error between expected end time based on period and the actual system
    ///   time)
    /// - u(t) = correction time to subtract from the expected end time
    /// - Kp = proportional gain (unitless)
    /// - Ki = integral gain (1/nanoseconds)
    /// - Kd = derivative gain (nanoseconds)
    ///
    double k_p = 0.01, k_i = 0, k_d = 0;
    /// @brief the maximum value of the integral term of the PID controller. This is
    /// used to prevent windup. The value scales with the stream period to ensure
    /// the integral term remains effective at different sampling rates. Default is
    /// 1x the stream period in nanoseconds.
    double max_integral = 0.1;
    /// @brief max_back_correction_factor sets the maximum that the PID controller
    /// can shift the end time of the acquisition cycle backwards. This is used to
    /// prevent scenarios where the PID controller tries to correct for a large
    /// error by shifting the time of the acquisition cycle to before the previous
    /// cycle, resulting in out of order timestamps.
    ///
    /// Expressed as a fraction of the stream period i.e.
    /// (stream_rate.period() * max_back_correction_factor);
    double max_back_correction_factor = 0.5;

    [[nodiscard]] telem::TimeSpan max_back_correction() const {
        return this->stream_rate.period() * this->max_back_correction_factor;
    }

    [[nodiscard]] double effective_max_integral() const {
        return max_integral * static_cast<double>(stream_rate.period().nanoseconds());
    }

    static HardwareTimedSampleClockConfig create_simple(
        const telem::Rate &sample_rate,
        const telem::Rate &stream_rate,
        const bool enable_skew_correction = true
    ) {
        common::HardwareTimedSampleClockConfig cfg{
            .sample_rate = sample_rate,
            .stream_rate = stream_rate,
        };
        if (enable_skew_correction) return cfg;
        cfg.k_p = 0;
        cfg.k_d = 0;
        cfg.k_i = 0;
        return cfg;
    }

    void validate() const {
        if (this->k_p < 0) throw std::invalid_argument("k_p must be non-negative");
        if (this->k_i < 0) throw std::invalid_argument("k_i must be non-negative");
        if (this->k_d < 0) throw std::invalid_argument("k_d must be non-negative");
    }
};

/// @brief a sample clock that relies on an external, steady hardware clock to
/// regulate the acquisition rate. Timestamps are interpolated based on a fixed
/// sample rate.
class HardwareTimedSampleClock final : public SampleClock {
    HardwareTimedSampleClockConfig cfg;
    /// @brief track the system time marking the end of the previous acquisition
    /// loop.
    telem::TimeStamp prev_system_end = telem::TimeStamp(0);
    /// @brief timestamp of the first sample in the current acquisition loop.
    telem::TimeStamp curr_start_sample_time = telem::TimeStamp(0);
    /// @brief the current integral term of the PID controller.
    double integral = 0.0;
    /// @brief the previous error term of the PID controller.
    double prev_error = 0.0;
    /// @brief the number of samples per channel acquired during each acquisition
    /// loop.
    size_t samples_per_chan = 0;

public:
    explicit HardwareTimedSampleClock(HardwareTimedSampleClockConfig cfg):
        cfg(std::move(cfg)), samples_per_chan(cfg.sample_rate / cfg.stream_rate) {
        this->cfg.validate();
    }

    void reset() override {
        this->prev_system_end = telem::TimeStamp(0);
        this->curr_start_sample_time = telem::TimeStamp(0);
        this->integral = 0.0;
        this->prev_error = 0.0;
    }

    telem::TimeStamp wait(breaker::Breaker &_) override {
        if (this->curr_start_sample_time == 0) {
            const auto now = this->cfg.now();
            this->curr_start_sample_time = now;
            this->prev_system_end = now;
        }
        return this->curr_start_sample_time;
    }

    telem::TimeStamp end() override {
        // We use a fixed increment based on the number of samples per chan and the
        // sample rate INSTEAD of the stream rate, as sometimes the stream rate does
        // not reflect the actual real stream rate. This is true in scenarios where
        // the sample rate is not an even multiple of the stream rate i.e. 2.5 KHz
        // sample rate and 200 Hz stream rate, where we'd get 12.5 samples per
        // channel.
        const auto fixed_increment = this->cfg.sample_rate.period() *
                                     this->samples_per_chan;
        auto sample_end = this->curr_start_sample_time + fixed_increment;
        const auto system_end = this->cfg.now();
        const double error = static_cast<double>(
            (sample_end - system_end).nanoseconds()
        );
        const double dt = static_cast<double>(
            (system_end - this->prev_system_end).nanoseconds()
        );
        const double p_term = this->cfg.k_p * error;
        this->integral += error * dt;
        this->integral = std::clamp(
            this->integral,
            -this->cfg.effective_max_integral(),
            this->cfg.effective_max_integral()
        );
        const double i_term = this->cfg.k_i * this->integral;
        const double d_term = this->cfg.k_d * (error - this->prev_error) / dt;
        this->prev_error = error;
        const auto pid_output = p_term + i_term + d_term;
        auto correction = telem::TimeSpan(static_cast<int64_t>(pid_output));
        if (correction > this->cfg.max_back_correction())
            correction = this->cfg.max_back_correction();
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
    const size_t offset,
    const bool inclusive = false
) {
    if (index_keys.empty()) return;
    // Hot path: Common to have one index, and it means we can avoid a deep copy.
    if (index_keys.size() == 1) {
        auto &s = f.series->at(offset);
        s.clear();
        s.write_linspace(start, end, n_read, inclusive);
        return;
    }
    const auto index_data = telem::Series::linspace(start, end, n_read, inclusive);
    for (size_t i = offset; i < index_keys.size() + offset; i++) {
        auto &s = f.series->at(i);
        s.clear();
        s.write(index_data);
    }
}
}
