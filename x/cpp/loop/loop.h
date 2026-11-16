// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <chrono>
#include <cmath>
#include <functional>
#include <memory>
#include <optional>
#include <thread>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/telem/telem.h"

using hs_clock = std::chrono::steady_clock;
using nanos = std::chrono::nanoseconds;

namespace x::loop {
/// @brief Threshold below which high-resolution timing is used.
const telem::TimeSpan HIGH_RES_THRESHOLD = telem::Rate(200).period();
/// @brief Threshold below which medium-resolution timing is used.
const telem::TimeSpan MEDIUM_RES_THRESHOLD = telem::Rate(20).period();
/// @brief Span of one sleep step of a precise sleep.
const telem::TimeSpan RESOLUTION = (100 * telem::MICROSECOND);

/// @brief a clock that a Timer reads and sleeps on in place of the host's steady clock.
/// A test passes one whose sleep moves its time forward at once.
struct Clock {
    /// @brief returns the current time.
    std::function<hs_clock::time_point()> now;
    /// @brief returns once the span has passed on this clock.
    std::function<void(telem::TimeSpan)> sleep;
};

class Timer {
public:
    Timer() = default;

    /// @brief a timer that waits on the host's steady clock.
    explicit Timer(const telem::TimeSpan &interval):
        interval(interval), last(hs_clock::now()) {}

    /// @brief a timer that waits on the host's steady clock. A precise timer spins to
    /// each deadline at any interval. Otherwise, only an interval under
    /// HIGH_RES_THRESHOLD spins, and a longer one sleeps, which on Windows can wake up
    /// to 2 ms late.
    Timer(const telem::TimeSpan &interval, const bool precise):
        interval(interval), precise(precise), last(hs_clock::now()) {}

    /// @brief a timer that reads and sleeps on clock in place of the host's steady
    /// clock.
    Timer(const telem::TimeSpan &interval, Clock clock):
        interval(interval), clock(std::move(clock)), last(this->clock_now()) {}

    explicit Timer(const telem::Rate &rate): Timer(rate.period()) {}

    telem::TimeSpan elapsed(const hs_clock::time_point now) {
        if (!last_set) {
            last_set = true;
            return telem::TimeSpan::ZERO();
        }
        const auto elapsed = now - last;
        return telem::TimeSpan(elapsed);
    }

    std::pair<telem::TimeSpan, bool> wait() {
        const auto now = this->clock_now();
        const auto elapsed = this->elapsed(now);
        if (elapsed > interval) {
            last = now;
            return {telem::TimeSpan(elapsed), false};
        }
        const auto remaining = interval - elapsed;
        const auto deadline = now + remaining.chrono();
        if (this->clock)
            this->clock->sleep(remaining);
        else if (this->precise || this->high_rate())
            this->precise_sleep(remaining);
        else
            std::this_thread::sleep_for(remaining.chrono());
        // Anchor to the deadline so a late wake shortens the next sleep instead of
        // stretching every period. An early wake keeps the wake time.
        last = std::min(deadline, this->clock_now());
        return {telem::TimeSpan(elapsed), true};
    }

    std::pair<telem::TimeSpan, bool> wait(breaker::Breaker &breaker) {
        const auto now = this->clock_now();
        const auto elapsed = this->elapsed(now);
        if (elapsed > interval) {
            last = now;
            return {telem::TimeSpan(elapsed), false};
        }
        const auto remaining = interval - elapsed;
        const auto deadline = now + remaining.chrono();
        if (this->clock)
            this->clock->sleep(remaining);
        else if (this->precise || this->high_rate())
            this->precise_sleep(remaining, breaker);
        else if (this->medium_rate())
            std::this_thread::sleep_for(remaining.chrono());
        else
            breaker.wait_for(remaining);
        last = std::min(deadline, this->clock_now());
        return {telem::TimeSpan(elapsed), true};
    }

    /// @brief moves the end of the next wait by offset. A positive offset delays it.
    /// Has no effect before the first wait(), which always lasts one interval.
    void shift(const telem::TimeSpan &offset) { this->last += offset.chrono(); }

    /// @brief sleeps for dur without spinning. On Windows 10 1803 and later, the sleep
    /// waits on a high-resolution timer, so it does not round up to the system tick.
    void sleep(const telem::TimeSpan &dur) {
        if (this->clock)
            this->clock->sleep(dur);
        else
            this->step(dur);
    }

    /// @brief Fine-grained sleep that ends early when the breaker stops.
    /// @returns false if the breaker stopped before or during the sleep.
    bool precise_sleep(const telem::TimeSpan &dur, const breaker::Breaker &breaker) {
        this->sleep_while(dur, [&breaker] { return breaker.running(); });
        return breaker.running();
    }

private:
    /// @brief Fine-grained sleep using Welford's online algorithm for calibration.
    void precise_sleep(const telem::TimeSpan &dur) {
        this->sleep_while(dur, [] { return true; });
    }

    /// @brief sleeps precisely for dur, and ends early once running returns false.
    template<typename Running>
    void sleep_while(const telem::TimeSpan &dur, const Running &running) {
        if (this->clock) {
            if (running()) this->clock->sleep(dur);
            return;
        }
        const auto end = hs_clock::now() + dur.chrono();
        while (true) {
            if (!running()) return;
            const auto start = hs_clock::now();
            if (telem::TimeSpan(end - start) <= sleep_estimate) break;
            this->step(RESOLUTION);
            const auto curr_end = hs_clock::now();
            const auto elapsed_ns = std::chrono::duration_cast<nanos>(curr_end - start)
                                        .count();
            const telem::TimeSpan delta = elapsed_ns - sleep_mean;
            sleep_mean += delta / sleep_count;
            sleep_M2 += delta * (elapsed_ns - sleep_mean);
            const auto stddev = std::sqrt((sleep_M2 / sleep_count).nanoseconds());
            sleep_estimate = sleep_mean + static_cast<int64_t>(stddev);
            sleep_count++;
        }
        while (end > hs_clock::now())
            if (!running()) return;
    }

    [[nodiscard]] hs_clock::time_point clock_now() const {
        return this->clock ? this->clock->now() : hs_clock::now();
    }

    [[nodiscard]] bool high_rate() const { return interval < HIGH_RES_THRESHOLD; }

    [[nodiscard]] bool medium_rate() const { return interval < MEDIUM_RES_THRESHOLD; }

    /// @brief the OS state of a sleep step, defined in the source file of each OS.
    struct Impl;
    /// @brief owns an Impl. The deleter is defined with Impl, so Timer needs no
    /// special members defined out of line.
    using ImplPtr = std::unique_ptr<Impl, void (*)(Impl *)>;

    /// @brief creates the OS state of a sleep step.
    static ImplPtr make_impl();

    /// @brief blocks for dur. On Windows 10 1803 and later, the step waits on a
    /// high-resolution timer, so it does not round up to the system tick.
    void step(const telem::TimeSpan &dur);

    telem::TimeSpan interval;
    std::optional<Clock> clock;
    bool precise = false;
    bool last_set = false;
    hs_clock::time_point last;
    /// @brief the OS state of a sleep step.
    ImplPtr impl = make_impl();

    /// @brief Welford's algorithm state: the most one step is expected to take. It
    /// starts at one step, so a sleep longer than one step measures a step.
    telem::TimeSpan sleep_estimate = RESOLUTION;
    /// @brief Welford's algorithm state: running mean of step durations.
    telem::TimeSpan sleep_mean = RESOLUTION;
    /// @brief Welford's algorithm state: sum of squared deviations.
    telem::TimeSpan sleep_M2 = telem::TimeSpan::ZERO();
    /// @brief Welford's algorithm state: sample count.
    int64_t sleep_count = 1;
};

/// @brief the host clocks that a loop reads: steady time to wait on and wall time to
/// stamp with. A default HostClock reads the host.
class HostClock {
    telem::TimeStamp wall_start;
    hs_clock::time_point steady_start;
    std::optional<Clock> source;

public:
    HostClock() = default;

    /// @brief reads both times from source. The wall time starts at the host's wall
    /// time and then moves with source.
    explicit HostClock(Clock source):
        wall_start(telem::TimeStamp::now()),
        steady_start(source.now()),
        source(std::move(source)) {}

    /// @brief returns the steady time.
    [[nodiscard]] hs_clock::time_point steady() const {
        return this->source ? this->source->now() : hs_clock::now();
    }

    /// @brief returns once span has passed on this clock.
    void sleep(const telem::TimeSpan &span) const {
        if (this->source)
            this->source->sleep(span);
        else
            std::this_thread::sleep_for(span.chrono());
    }

    /// @brief returns a timer that waits on this clock. precise is as for Timer. A
    /// timer on a source waits exactly at any interval.
    [[nodiscard]] Timer
    timer(const telem::TimeSpan &interval, const bool precise = false) const {
        if (!this->source) return {interval, precise};
        return {interval, *this->source};
    }

    /// @brief returns the wall time.
    [[nodiscard]] telem::TimeStamp now() const {
        if (!this->source) return telem::TimeStamp::now();
        return this->wall_start +
               telem::TimeSpan(this->source->now() - this->steady_start);
    }
};
}
