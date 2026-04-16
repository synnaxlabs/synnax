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
#include <functional>

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

namespace arc::runtime::node {

/// @brief Indicates what triggered the current scheduler run.
enum class RunReason {
    /// @brief Scheduler run triggered by timer expiration.
    TimerTick,
    /// @brief Scheduler run triggered by input data arrival.
    ChannelInput,
};

struct Context {
    x::telem::TimeSpan elapsed;
    x::telem::TimeSpan tolerance;
    /// @brief Indicates what triggered this scheduler run.
    /// Time-based nodes should only fire when reason is TimerTick.
    RunReason reason;
    /// @brief records that one of the current node's outputs has a new
    /// value for the current cycle. The ordinal is the output's 0-based
    /// position in the owning ir::Node's outputs slice. Zero hash
    /// lookups on the hot path.
    std::function<void(size_t output_idx)> mark_changed;
    std::function<void()> mark_self_changed;
    std::function<void(x::telem::TimeSpan)> set_deadline;
    std::function<void(const x::errors::Error &)> report_error;
};

class Node {
public:
    virtual ~Node() = default;

    virtual x::errors::Error next(Context &ctx) = 0;

    /// Reset is called when a stage containing this node is activated.
    /// Nodes can override to reset their internal state (e.g., timers, counters).
    /// Default implementation does nothing.
    virtual void reset() {}

    /// @brief reports whether the output at the given 0-based ordinal is
    /// truthy. Used by the scheduler to evaluate conditional edges and
    /// sequential-scope transitions — both fire only when the source
    /// output is truthy (non-zero for numeric types). Nodes that embed
    /// state::Node automatically inherit an implementation that indexes
    /// into the output cache without any string lookup.
    [[nodiscard]] virtual bool is_output_truthy(size_t output_idx) const = 0;
};
}
