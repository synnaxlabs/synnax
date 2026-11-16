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
#include <vector>

#include "x/cpp/telem/telem.h"

#include "driver/ethercat/dc/dc.h"

namespace driver::ethercat::engine {
/// @brief holds the bus cycle at a fixed phase of the distributed clock: each frame
/// passes the reference clock midway in the longest span between SYNC0 events. The
/// events repeat on a grid, the largest span that divides the period and each cycle.
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
    double period;
    /// @brief a SYNC0 cycle in nanoseconds. The grid divides it, so phases reduce by it
    /// in integers before the floating point math.
    int64_t cycle;
    double grid;
    /// @brief the longest span between SYNC0 events in a grid.
    double gap = 0;
    /// @brief a time half a grid before the target phase.
    x::telem::TimeStamp origin;
    double error = 0;
    double integral = 0;
    bool held = false;

    static double
    grid_of(const x::telem::TimeSpan period, const std::vector<Sync0> &syncs) {
        std::vector<x::telem::TimeSpan> cycles;
        cycles.reserve(syncs.size());
        for (const auto &s: syncs)
            cycles.push_back(s.cycle);
        return dc::grid(period, cycles);
    }

public:
    /// @param rate the bus rate.
    /// @param syncs the SYNC0 events of each slave whose SYNC0 marks process data. Must
    /// not be empty.
    PhaseLock(const x::telem::Rate rate, const std::vector<Sync0> &syncs):
        period(static_cast<double>(rate.period().nanoseconds())),
        cycle(syncs.front().cycle.nanoseconds()),
        grid(grid_of(rate.period(), syncs)) {
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
