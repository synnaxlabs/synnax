// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <array>
#include <memory>
#include <string>
#include <vector>

#include "arc/cpp/runtime/core/node.h"
#include "arc/cpp/runtime/state/node_state.h"
#include "arc/cpp/runtime/wasm/runtime.h"
#include "x/cpp/xerrors/errors.h"

namespace arc {
namespace wasm {

/// @brief WASM node that executes compiled Arc stage functions.
///
/// Implements the arc::Node interface by calling WASM functions via the Runtime.
/// Each Node corresponds to one Arc stage (function) and owns its
/// state::Node for scoped access to channels and state variables.
///
/// Example Arc code:
/// @code
/// stage calculate_average(input: f64) -> f64 {
///     return input / 2.0
/// }
/// @endcode
class Node : public arc::Node {
    std::string id_;                           ///< Node identifier
    std::unique_ptr<state::Node> node_state_;  ///< Per-node state (owned)
    Runtime& runtime_;                         ///< WASM runtime reference (non-owning)
    wasm_function_inst_t function_;            ///< WASM function to execute
    std::vector<std::string> output_params_;   ///< Output parameter names

    // Pre-allocated buffers for function calls (RT-safe)
    static constexpr size_t MAX_ARGS = 16;
    static constexpr size_t MAX_RESULTS = 16;
    std::array<WasmValue, MAX_ARGS> args_;
    std::array<WasmValue, MAX_RESULTS> results_;

public:
    /// @brief Construct WASM node.
    /// @param id Node identifier.
    /// @param node_state NodeState for this node (ownership transferred).
    /// @param runtime WASM runtime (must outlive this node).
    /// @param function WASM function instance to execute.
    /// @param output_params Output parameter names (for change tracking).
    Node(std::string id,
             std::unique_ptr<NodeState> node_state,
             Runtime* runtime,
             wasm_function_inst_t function,
             std::vector<std::string> output_params);

    /// @brief Execute this node's WASM function.
    ///
    /// Execution logic:
    /// 1. Check if input data is available (via NodeState)
    /// 2. If no data, return NIL (skip execution)
    /// 3. Call WASM function via Runtime
    /// 4. Mark outputs changed via context callback
    /// 5. Handle any errors
    ///
    /// @param ctx Node context with callbacks for change tracking.
    /// @return Error status (NIL on success).
    /// @note RT-safe: No allocations, calls AOT-compiled WASM.
    xerrors::Error execute(NodeContext& ctx) override;

    /// @brief Get node identifier.
    /// @return Node ID string.
    std::string id() const override { return id_; }

    /// @brief Get NodeState reference.
    /// @return Reference to this node's state.
    NodeState& state() { return *node_state_; }
    const NodeState& state() const { return *node_state_; }

    /// @brief Get WASM function instance.
    /// @return Function instance pointer.
    wasm_function_inst_t function() const { return function_; }
};

}  // namespace wasm
}  // namespace arc
