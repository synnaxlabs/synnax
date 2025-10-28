// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <string>

#include "arc/cpp/runtime/core/node.h"
#include "arc/cpp/runtime/core/types.h"
#include "arc/cpp/runtime/state/state.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"

namespace arc {
namespace interval {

/// @brief Interval node that emits ticks at a fixed period.
///
/// Self-checking interval source node that executes in stratum-0 and
/// emits tick signals when its configured period has elapsed. Matches
/// the Arc language's interval{} construct.
///
/// Example Arc code:
/// @code
/// interval{period: 100ms} -> tick;
/// @endcode
class IntervalNode : public Node {
    std::string id_;                   ///< Node identifier
    State& state_;                     ///< State reference (non-owning)
    ChannelKey output_ch_;             ///< Output channel for tick signal
    uint64_t period_ns_;               ///< Interval period in nanoseconds
    telem::TimeStamp last_execution_;  ///< Last execution timestamp

public:
    /// @brief Construct interval node.
    /// @param id Node identifier.
    /// @param state State reference (non-owning).
    /// @param output_ch Output channel key for tick signal.
    /// @param period_ns Interval period in nanoseconds.
    IntervalNode(std::string id,
                 State* state,
                 ChannelKey output_ch,
                 uint64_t period_ns);

    /// @brief Execute interval check and emit tick if period elapsed.
    ///
    /// Execution logic:
    /// 1. Get current timestamp
    /// 2. Check if period has elapsed since last execution
    /// 3. If yes: write tick signal (u8 = 1) to output channel
    /// 4. If yes: mark output changed via context
    /// 5. If yes: update last_execution timestamp
    /// 6. If no: do nothing (return early)
    ///
    /// @param ctx Node context with callbacks for change tracking.
    /// @return Error status (NIL on success).
    /// @note RT-safe: Simple timestamp comparison, no allocations.
    xerrors::Error execute(NodeContext& ctx) override;

    /// @brief Get node identifier.
    /// @return Node ID string.
    std::string id() const override { return id_; }
};

}  // namespace interval
}  // namespace arc
