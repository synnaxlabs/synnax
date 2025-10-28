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

#include "arc/cpp/runtime/core/context.h"
#include "x/cpp/xerrors/errors.h"

namespace arc {

/// @brief Abstract node interface for executable units in the scheduler.
///
/// Nodes represent compiled Arc stages (WASM functions, operators, constants, etc.)
/// that can be executed by the scheduler. Each node implements reactive execution
/// logic and reports output changes via the NodeContext.
///
/// Implementations include:
/// - WASMNode: Executes compiled user functions
/// - IntervalNode: Emits periodic ticks
/// - OperatorNode: Arithmetic/logical operations (future)
/// - ConstantNode: Constant values (future)
/// - TelemNode: Channel I/O (future)
class Node {
public:
    virtual ~Node() = default;

    /// @brief Execute this node.
    ///
    /// Execution is reactive: nodes check if they have new input data and
    /// skip execution if not. When execution produces output, the node calls
    /// ctx.mark_changed() to trigger downstream re-execution.
    ///
    /// @param ctx Node context with callbacks for change tracking and error reporting.
    /// @return Error status (NIL on success).
    /// @note Must be RT-safe if used in RT thread.
    virtual xerrors::Error execute(NodeContext& ctx) = 0;

    /// @brief Get node identifier.
    /// @return Unique node ID string.
    virtual std::string id() const = 0;
};

}  // namespace arc
