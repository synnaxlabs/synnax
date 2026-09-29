// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <optional>
#include <vector>

#include "x/cpp/telem/telem.h"

namespace driver::bus {
/// @brief Schedule tracks when each message a bus controller sends is next due. Not
/// safe for concurrent use.
class Schedule {
public:
    Schedule() = default;

    /// @param periods the period of each message. A message with no period is never
    /// due.
    /// @param start the time every periodic message is first due.
    Schedule(
        const std::vector<std::optional<x::telem::TimeSpan>> &periods,
        x::telem::TimeStamp start
    );

    /// @returns the earliest time a message is due, or nullopt when no message has
    /// a period.
    [[nodiscard]] std::optional<x::telem::TimeStamp> next() const;

    /// @brief appends each message due at now to out and moves it to its next due
    /// time. A message more than one period late skips the sends it missed.
    void due(x::telem::TimeStamp now, std::vector<std::size_t> &out);

private:
    /// @brief Entry is one periodic message.
    struct Entry {
        /// @brief message is the index of the message.
        std::size_t message = 0;
        /// @brief period is the message's period.
        x::telem::TimeSpan period;
        /// @brief at is when the message is next due.
        x::telem::TimeStamp at;
    };

    /// @brief entries holds the periodic messages.
    std::vector<Entry> entries;
};
}
