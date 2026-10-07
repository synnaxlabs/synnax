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
#include <atomic>
#include <cstdint>
#include <cstdlib>
#include <iomanip>
#include <ostream>
#include <thread>
#include <utility>
#include <vector>

#include "x/cpp/telem/telem.h"

namespace arc::runtime::testutil {
/// @brief Statistics of a set of timing errors. An error is the time from a deadline
/// to its event, so a negative error is an early event. The set must not be empty.
class Spread {
    /// @brief the errors in nanoseconds, in ascending order.
    std::vector<std::int64_t> sorted_ns;

public:
    explicit Spread(std::vector<std::int64_t> errors_ns):
        sorted_ns(std::move(errors_ns)) {
        std::sort(this->sorted_ns.begin(), this->sorted_ns.end());
    }

    /// @brief returns the error at percent of the sorted set, in nanoseconds.
    [[nodiscard]] std::int64_t at(const size_t percent) const {
        return this->sorted_ns[(this->sorted_ns.size() - 1) * percent / 100];
    }

    /// @brief returns how many errors are further than bound from zero.
    [[nodiscard]] size_t over(const x::telem::TimeSpan bound) const {
        size_t count = 0;
        for (const auto error_ns: this->sorted_ns)
            if (std::abs(error_ns) > bound.nanoseconds()) count++;
        return count;
    }

    /// @brief prints the percentiles and the range in microseconds, then how many
    /// errors are over 0.1, 0.5, and 1 ms.
    friend std::ostream &operator<<(std::ostream &os, const Spread &s) {
        os << std::fixed << std::setprecision(1) << "min " << s.at(0) / 1e3 << ", p50 "
           << s.at(50) / 1e3 << ", p90 " << s.at(90) / 1e3 << ", p99 " << s.at(99) / 1e3
           << ", max " << s.at(100) / 1e3 << " us, spread "
           << (s.at(100) - s.at(0)) / 1e3
           << " us, over 0.1/0.5/1 ms: " << s.over(100 * x::telem::MICROSECOND) << "/"
           << s.over(500 * x::telem::MICROSECOND) << "/"
           << s.over(x::telem::MILLISECOND);
        return os;
    }
};

/// @brief Spins one thread on each core until destroyed.
class Load {
    std::atomic<bool> running{true};
    std::vector<std::thread> threads;

public:
    Load() {
        const auto count = std::max(1u, std::thread::hardware_concurrency());
        for (unsigned int i = 0; i < count; i++)
            this->threads.emplace_back([this] {
                while (this->running.load(std::memory_order_relaxed))
                    continue;
            });
    }

    ~Load() {
        this->running.store(false);
        for (auto &thread: this->threads)
            thread.join();
    }
};
}
