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
#include "arc/cpp/runtime/nodes/wasm/node.h"

namespace arc {
namespace wasm {

/// @brief Factory for creating WASM nodes from compiled Arc functions.
///
/// Handles node types that correspond to user-defined Arc functions compiled
/// to WebAssembly. Checks if the node type exists in the IR's function list,
/// and if so, creates a WASMNode that executes the compiled function.
///
/// Chain of Responsibility behavior:
/// - Returns {WASMNode, NIL} if node type matches an IR function
/// - Returns {nullptr, NOT_FOUND} if node type is not an IR function
/// - Returns {nullptr, error} if WASM function lookup or node creation fails
///
/// Example Arc code that produces WASM nodes:
/// @code
/// stage calculate_average(input: f64) -> f64 {
///     return input / 2.0
/// }
/// @endcode
/// This creates a node with type="calculate_average" that maps to a WASM function.
class WASMNodeFactory : public NodeFactory {
    Runtime& runtime_;  ///< WASM runtime reference (non-owning)

public:
    /// @brief Construct WASM factory with runtime reference.
    /// @param runtime WASM runtime (must outlive this factory).
    explicit WASMNodeFactory(Runtime& runtime) : runtime_(runtime) {}

    /// @brief Create a WASM node if the type matches an IR function.
    ///
    /// Creation steps:
    /// 1. Check if cfg.ir_node.type exists in cfg.ir->functions
    /// 2. If not found, return NOT_FOUND (not a WASM function)
    /// 3. Verify cfg.runtime is not null (WASM runtime required)
    /// 4. Build NodeState with input edges and output handles
    /// 5. Find WASM function instance via Runtime::find_function()
    /// 6. Create WASMNode with function instance and output parameter names
    ///
    /// @param cfg Factory configuration with IR node and dependencies.
    /// @return {WASMNode, NIL} on success, {nullptr, NOT_FOUND} if not a WASM function,
    ///         {nullptr, error} on failure.
    std::pair<std::unique_ptr<Node>, xerrors::Error>
    create(const NodeFactoryConfig& cfg) override;
};

}  // namespace wasm
}  // namespace arc
