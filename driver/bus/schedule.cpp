// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/bus/schedule.h"

namespace driver::bus {
Schedule::Schedule(
    const std::vector<std::optional<x::telem::TimeSpan>> &periods,
    const x::telem::TimeStamp start
) {
    for (std::size_t i = 0; i < periods.size(); i++)
        if (periods[i].has_value())
            this->entries.push_back({.message = i, .period = *periods[i], .at = start});
}

std::optional<x::telem::TimeStamp> Schedule::next() const {
    std::optional<x::telem::TimeStamp> next;
    for (const auto &e: this->entries)
        if (!next.has_value() || e.at < *next) next = e.at;
    return next;
}

void Schedule::due(const x::telem::TimeStamp now, std::vector<std::size_t> &out) {
    for (auto &e: this->entries) {
        if (e.at > now) continue;
        out.push_back(e.message);
        e.at = e.at + e.period;
        if (e.at <= now) e.at = now + e.period;
    }
}
}
