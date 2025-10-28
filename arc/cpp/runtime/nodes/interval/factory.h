// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include "arc/cpp/runtime/factory/factory.h"
#include "arc/cpp/runtime/nodes/interval/node.h"

namespace arc {
namespace interval {

/// @brief Factory for creating interval nodes that emit periodic ticks.
///
/// Handles the special "interval" node type from Arc's interval{} construct.
/// These nodes execute in stratum-0 and emit tick signals when their configured
/// period has elapsed.
///
/// Chain of Responsibility behavior:
/// - Returns {Node, NIL} if node type is "interval"
/// - Returns {nullptr, NOT_FOUND} if node type is not "interval"
/// - Returns {nullptr, error} if configuration is invalid (missing period/output)
///
/// Example Arc code that produces interval nodes:
/// @code
/// interval{period: 100ms} -> tick;
/// @endcode
/// This creates a node with type="interval" and config_values["period"] = 100000000 (ns).
class Factory : public NodeFactory {
public:
    /// @brief Create an interval node if the type is "interval".
    ///
    /// Creation steps:
    /// 1. Check if cfg.ir_node.type == "interval"
    /// 2. If not, return NOT_FOUND (not an interval node)
    /// 3. Extract period from config_values["period"] (uint64_t in nanoseconds)
    /// 4. Extract output channel key from channels.write["output"]
    /// 5. Create Node with period and output channel
    ///
    /// Required IR node structure:
    /// - type: "interval"
    /// - config_values["period"]: uint64_t (nanoseconds)
    /// - channels.write["output"]: ChannelKey
    ///
    /// @param cfg Factory configuration with IR node and dependencies.
    /// @return {Node, NIL} on success, {nullptr, NOT_FOUND} if not "interval",
    ///         {nullptr, error} if configuration is invalid.
    std::pair<std::unique_ptr<arc::Node>, xerrors::Error>
    create(const NodeFactoryConfig& cfg) override;
};

}  // namespace interval
}  // namespace arc
