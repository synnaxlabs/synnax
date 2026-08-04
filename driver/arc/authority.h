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
#include <unordered_map>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/control/control.h"

#include "driver/common/status.h"
#include "driver/control/state.h"

// Mirrors core/pkg/service/arc/task/control.go. Keep the warning message in sync.
namespace driver::arc::authority {

/// @brief a write channel currently held by a subject other than this task.
struct Conflict {
    synnax::channel::Channel channel;
    x::control::Subject holder;
};

/// @brief returns the write channels held by a subject other than self, sorted by
/// channel key for stable output.
inline std::vector<Conflict> evaluate_conflicts(
    const driver::control::States &states,
    const std::vector<synnax::channel::Channel> &writes,
    const x::control::Subject &self
) {
    std::vector<Conflict> conflicts;
    for (const auto &ch: writes) {
        auto held = states.holder(ch.key);
        if (!held.has_value() || held->subject == self) continue;
        conflicts.push_back({ch, held->subject});
    }
    std::sort(
        conflicts.begin(),
        conflicts.end(),
        [](const Conflict &a, const Conflict &b) {
            return a.channel.key < b.channel.key;
        }
    );
    return conflicts;
}

/// @brief returns the holder's name, falling back to its key.
inline std::string holder_name(const x::control::Subject &subject) {
    return subject.name.empty() ? subject.key : subject.name;
}

/// @brief builds the summary message and holder-grouped description for a conflict set.
/// conflicts must be sorted by channel key so the grouping is deterministic.
inline std::pair<std::string, std::string>
build_warning(const std::vector<Conflict> &conflicts) {
    const std::string channel_noun = conflicts.size() == 1 ? "channel" : "channels";
    std::vector<x::control::Subject> holders;
    std::unordered_map<std::string, std::vector<std::string>> channels_by_holder;
    for (const auto &c: conflicts) {
        if (channels_by_holder.find(c.holder.key) == channels_by_holder.end())
            holders.push_back(c.holder);
        channels_by_holder[c.holder.key].push_back(c.channel.name);
    }
    const std::string writer_phrase = holders.size() > 1 ? "other writers"
                                                         : "another writer";
    const std::string message = "Authority held on " +
                                std::to_string(conflicts.size()) + " " + channel_noun +
                                " by " + writer_phrase;
    std::string description;
    for (size_t i = 0; i < holders.size(); i++) {
        if (i > 0) description += "\n";
        description += holder_name(holders[i]) + ": ";
        const auto &names = channels_by_holder[holders[i].key];
        for (size_t j = 0; j < names.size(); j++) {
            if (j > 0) description += ", ";
            description += names[j];
        }
    }
    return {message, description};
}

/// @brief warns on control-authority conflicts over a task's write channels, clearing
/// once none remain. Mirrors controlWarner in core/pkg/service/arc/task/control.go.
class Warner {
    /// @brief live per-channel holder map. Null when the context provides none.
    std::shared_ptr<driver::control::States> states;
    /// @brief non-index write channels named in the warning.
    std::vector<synnax::channel::Channel> writes;
    /// @brief this task's control subject, excluded from its own conflicts.
    x::control::Subject self;

public:
    Warner(
        std::shared_ptr<driver::control::States> states,
        std::vector<synnax::channel::Channel> writes,
        x::control::Subject self
    ):
        states(std::move(states)), writes(std::move(writes)), self(std::move(self)) {}

    /// @brief warns when another writer out-ranks this task, clearing once no
    /// conflicts remain. Inert when no states or write channels are present.
    void report(common::StatusHandler &state) const {
        if (this->states == nullptr || this->writes.empty()) return;
        const auto conflicts = evaluate_conflicts(
            *this->states,
            this->writes,
            this->self
        );
        if (conflicts.empty()) return state.clear_warning();
        state.send_warning(build_warning(conflicts).first);
    }
};
}
