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
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/telem/mono_clock.h"
#include "x/cpp/telem/telem.h"

namespace driver::bus {
/// @brief how long a warning stays on the task status after its cause stops.
const auto WARNING_HOLD = 1 * x::telem::SECOND;

/// @brief the warnings of a task that each hold for WARNING_HOLD after they last
/// occurred, such as a frame that failed to decode. Not safe for concurrent use.
class Warnings {
public:
    /// @param now the steady clock that times the hold.
    explicit Warnings(x::telem::NowFunc now): now(std::move(now)) {}

    /// @brief adds a warning, or restarts its hold when it is already there.
    void add(const std::string &message) {
        const auto at = this->now();
        for (auto &e: this->entries)
            if (e.message == message) {
                e.at = at;
                return;
            }
        this->entries.push_back({message, at});
    }

    /// @brief appends each warning younger than WARNING_HOLD to out, in the order they
    /// first occurred, and forgets the older ones.
    void collect(std::vector<std::string> &out) {
        const auto now = this->now();
        std::erase_if(this->entries, [now](const Entry &e) {
            return now - e.at >= WARNING_HOLD;
        });
        for (const auto &e: this->entries)
            out.push_back(e.message);
    }

    /// @returns the steady time at which the next warning expires, or TimeStamp::max()
    /// when there is none.
    [[nodiscard]] x::telem::TimeStamp expiry() const {
        auto next = x::telem::TimeStamp::max();
        for (const auto &e: this->entries)
            next = std::min(next, e.at + WARNING_HOLD);
        return next;
    }

    /// @brief forgets every warning.
    void clear() { this->entries.clear(); }

private:
    struct Entry {
        std::string message;
        x::telem::TimeStamp at;
    };

    x::telem::NowFunc now;
    std::vector<Entry> entries;
};
}
