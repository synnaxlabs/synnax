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
#include <cmath>
#include <cstdint>
#include <numeric>
#include <utility>
#include <vector>

#include "x/cpp/telem/telem.h"

namespace driver::ethercat::engine {
/// @brief holds the bus cycle at a fixed phase of the distributed clock. The grid is
/// the largest span that divides every SYNC0 cycle and the period, so the SYNC0 events
/// of all slaves fall at the same phases in each grid. Each frame passes the reference
/// clock midway in the longest span between those phases, the phase farthest from
/// every SYNC0 event.
class PhaseLock {
public:
    /// @brief the SYNC0 events of a slave.
    struct Sync0 {
        /// @brief the SYNC0 cycle. Must be positive.
        x::telem::TimeSpan cycle;
        /// @brief the time of a SYNC0 event.
        x::telem::TimeStamp event;
    };

private:
    /// @brief the share of the phase error that each cycle corrects.
    static constexpr double KP = 0.05;
    /// @brief the share of the summed phase error that each cycle corrects. It cancels
    /// the drift between the host and DC clocks.
    static constexpr double KI = 0.0001;
    /// @brief how far in nanoseconds a multiple of the grid can be from the period. The
    /// period of a rate such as 300 Hz is not a whole number of nanoseconds, and a
    /// SYNC0 cycle made from that period is rounded.
    static constexpr double TOLERANCE = 1;
    double period;
    /// @brief the greatest common divisor of the SYNC0 cycles.
    int64_t cycle;
    double grid;
    /// @brief the longest span between SYNC0 events in a grid.
    double gap = 0;
    /// @brief a time half a grid before the target phase.
    x::telem::TimeStamp origin;
    double error = 0;
    double integral = 0;
    bool held = false;

    /// @brief returns the fraction with the smallest denominator in [lo, hi], as a
    /// numerator and denominator. Requires 0 < lo <= hi.
    static std::pair<double, double>
    simplest_fraction(const double lo, const double hi) {
        if (std::ceil(lo) <= hi) return {std::ceil(lo), 1};
        const double whole = std::floor(lo);
        const auto [num, den] = simplest_fraction(1 / (hi - whole), 1 / (lo - whole));
        return {whole * num + den, num};
    }

    /// @brief returns the greatest common divisor of the SYNC0 cycles in nanoseconds.
    static int64_t common_cycle(const std::vector<Sync0> &syncs) {
        int64_t cycle = 0;
        for (const auto &s: syncs)
            cycle = std::gcd(cycle, s.cycle.nanoseconds());
        return cycle;
    }

    /// @brief returns the largest span that divides sync0 a whole number of times and
    /// divides period within TOLERANCE.
    static double common_grid(const double period, const double sync0) {
        // sync0 / n divides period within TOLERANCE when period / sync0 is within
        // TOLERANCE / sync0 of a fraction m / n.
        const double lo = (period - TOLERANCE) / sync0;
        const double hi = (period + TOLERANCE) / sync0;
        return sync0 / simplest_fraction(lo, hi).second;
    }

public:
    /// @param rate the bus rate.
    /// @param syncs the SYNC0 events of each slave that runs SYNC0. Must not be empty.
    PhaseLock(const x::telem::Rate rate, const std::vector<Sync0> &syncs):
        period(static_cast<double>(x::telem::SECOND.nanoseconds()) / rate.hz()),
        cycle(common_cycle(syncs)),
        grid(common_grid(this->period, static_cast<double>(this->cycle))) {
        const auto first = syncs.front().event;
        std::vector<double> phases;
        for (const auto &s: syncs) {
            const auto since = (s.event - first).nanoseconds() % this->cycle;
            const auto phase = std::fmod(static_cast<double>(since), this->grid);
            phases.push_back(phase < 0 ? phase + this->grid : phase);
        }
        std::ranges::sort(phases);
        double start = phases.back();
        this->gap = phases.front() + this->grid - phases.back();
        for (size_t i = 1; i < phases.size(); ++i)
            if (phases[i] - phases[i - 1] > this->gap) {
                this->gap = phases[i] - phases[i - 1];
                start = phases[i - 1];
            }
        const auto target = start + this->gap / 2;
        this->origin = first +
                       x::telem::TimeSpan(
                           static_cast<int64_t>(std::llround(target - this->grid / 2))
                       );
    }

    /// @brief returns how far to move the start of the next cycle, at most a tenth of
    /// the period either way. A positive value delays it.
    /// @param dc the DC time at which the latest frame passed the reference clock.
    x::telem::TimeSpan update(const x::telem::TimeStamp dc) {
        const auto since = ((dc - this->origin).nanoseconds() % this->cycle +
                            this->cycle) %
                           this->cycle;
        this->error = std::fmod(static_cast<double>(since), this->grid) -
                      this->grid / 2;
        const auto limit = this->period / 10;
        const auto shift = -(KP * this->error + KI * (this->integral + this->error));
        if (std::abs(shift) < limit) this->integral += this->error;
        if (this->locked()) this->held = true;
        return x::telem::TimeSpan(
            static_cast<int64_t>(std::clamp(shift, -limit, limit))
        );
    }

    /// @brief true when the latest frame passed the reference clock within a quarter
    /// of the longest span between SYNC0 events of the target phase.
    [[nodiscard]] bool locked() const { return std::abs(this->error) <= this->gap / 4; }

    /// @brief true when the lock held at an earlier frame and the latest frame is out
    /// of it.
    [[nodiscard]] bool lost() const { return this->held && !this->locked(); }
};
}
