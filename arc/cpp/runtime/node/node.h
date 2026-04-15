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
#include <string>
#include <vector>

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
    /// position in the list returned by Node::outputs(). Zero hash
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

    /// @brief Returns the output param names in canonical order. The
    /// position of a name in this list is its ordinal for mark_changed
    /// and is_output_truthy calls. The scheduler pre-populates its
    /// per-output propagation tables from this list so that ordinals
    /// are stable and known statically to the node implementation.
    [[nodiscard]] virtual std::vector<std::string> outputs() const { return {}; }

    /// @brief Checks if the output at the given param name is truthy.
    /// Used by the scheduler to evaluate one-shot edges - edges only fire
    /// when the source output is truthy.
    /// @param param The name of the output parameter to check.
    /// @returns true if the output exists and its last value is non-zero, false
    /// otherwise.
    [[nodiscard]] virtual bool is_output_truthy(const std::string &param) const = 0;
};
}
