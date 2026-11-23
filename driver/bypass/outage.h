// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <atomic>
#include <cstddef>
#include <cstdint>
#include <ostream>
#include <string>

#include "x/cpp/log/log.h"
#include "x/cpp/telem/telem.h"

namespace driver::bypass {
/// @brief how the Driver holds frames for the Core while the Core is slow or
/// unreachable.
struct OutageConfig {
    /// @brief the bytes that frames waiting for the Core may hold, for all tasks of the
    /// Driver together.
    size_t queue_size = size_t{256} * 1024 * 1024;
    /// @brief the longest time that a stopping task sends the frames that wait for the
    /// Core. It drops the rest.
    x::telem::TimeSpan drain_timeout = 5 * x::telem::SECOND;
    /// @brief the shortest time between two task status reports of a new drop count.
    x::telem::TimeSpan drop_report_interval = 5 * x::telem::SECOND;
    /// @brief the wait before a task first tries again to reach an unreachable Core.
    /// Each later wait is 1.1 times longer.
    x::telem::TimeSpan reopen_interval = x::telem::SECOND;
    /// @brief the longest wait between two tries to reach an unreachable Core.
    x::telem::TimeSpan max_reopen_interval = x::telem::MINUTE;

    template<typename Parser>
    void override(Parser &p) {
        this->queue_size = p.field("queue_size", this->queue_size);
        const auto span = [&p](const std::string &name, const x::telem::TimeSpan &def) {
            const auto s = p.field(name, def.seconds());
            return x::telem::TimeSpan(static_cast<int64_t>(s * 1e9));
        };
        this->drain_timeout = span("drain_timeout", this->drain_timeout);
        this->drop_report_interval = span(
            "drop_report_interval",
            this->drop_report_interval
        );
        this->reopen_interval = span("reopen_interval", this->reopen_interval);
        this->max_reopen_interval = span(
            "max_reopen_interval",
            this->max_reopen_interval
        );
    }

    friend std::ostream &operator<<(std::ostream &os, const OutageConfig &cfg) {
        const auto field = [&os](const std::string &name) -> std::ostream & {
            return os << "  " << x::log::SHALE() << name << x::log::RESET() << ": ";
        };
        field("queue size") << cfg.queue_size / (size_t{1024} * 1024) << " MB\n";
        field("drain timeout") << cfg.drain_timeout.seconds() << "s\n";
        field("drop report interval") << cfg.drop_report_interval.seconds() << "s\n";
        field("reopen interval") << cfg.reopen_interval.seconds() << "s\n";
        field("max reopen interval") << cfg.max_reopen_interval.seconds() << "s";
        return os;
    }
};

/// @brief a count of bytes that many queues share, up to a fixed size. Safe for
/// concurrent use, and lock free.
class Budget {
    const size_t cap;
    std::atomic<size_t> held = 0;

public:
    explicit Budget(const size_t size): cap(size) {}

    Budget(const Budget &) = delete;
    Budget &operator=(const Budget &) = delete;

    /// @brief takes bytes from the budget.
    /// @param force takes them even when they do not fit.
    /// @returns false, and takes nothing, when the bytes do not fit.
    bool reserve(const size_t bytes, const bool force = false) {
        if (force) {
            this->held.fetch_add(bytes, std::memory_order_relaxed);
            return true;
        }
        auto current = this->held.load(std::memory_order_relaxed);
        do {
            if (current + bytes > this->cap) return false;
        } while (!this->held.compare_exchange_weak(
            current,
            current + bytes,
            std::memory_order_relaxed
        ));
        return true;
    }

    /// @brief gives back bytes that reserve took.
    void release(const size_t bytes) {
        this->held.fetch_sub(bytes, std::memory_order_relaxed);
    }

    /// @returns the bytes that all reservations hold.
    [[nodiscard]] size_t used() const {
        return this->held.load(std::memory_order_relaxed);
    }

    /// @returns the bytes that the budget holds at most, except for forced
    /// reservations.
    [[nodiscard]] size_t size() const { return this->cap; }
};
}
