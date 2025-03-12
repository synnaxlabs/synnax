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
    virtual telem::TimeStamp end(size_t n_read) = 0;
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
        this->timer.wait(breaker);
        return telem::TimeStamp::now();
    }

    telem::TimeStamp end(const size_t _) override {
        return telem::TimeStamp::now();
    }
};

/// @brief a sample clock that relies on an external, steady hardware clock to
/// regulate the acquisition rate. Timestamps are interpolated based on a fixed
/// sample rate.
class HardwareTimedSampleClock final : public SampleClock {
    /// @brief the sample rate of the task.
    const telem::Rate sample_rate;
    /// @brief the high water-mark for the next acquisition loop.
    telem::TimeStamp high_water{};

public:
    explicit HardwareTimedSampleClock(const telem::Rate sample_rate):
        sample_rate(sample_rate) {
    }

    void reset() override {
        this->high_water = telem::TimeStamp::now();
    }

    telem::TimeStamp wait(breaker::Breaker &_) override {
        if (this->high_water == 0)
            throw std::runtime_error("hardware sample clock not reset before first `wait` called. Call `reset` before waiting on the sample clock.");
        return this->high_water;
    }

    telem::TimeStamp end(const size_t n_read) override {
        if (n_read == 0) return this->high_water;
        const auto end = this->high_water + (n_read - 1) * this->sample_rate.period();
        this->high_water = end + this->sample_rate.period();
        return end;
    }
};

inline void generate_index_data(
    const synnax::Frame &f,
    const std::set<synnax::ChannelKey> &index_keys,
    const telem::TimeStamp &start,
    const telem::TimeStamp &end,
    const size_t n_read
) {
    if (index_keys.empty()) return;
    const auto index_data = telem::Series::linspace(start, end, n_read);
    for (const auto &idx: index_keys)
        f.emplace(idx, std::move(index_data.deep_copy()));
}
}
