// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/wasm_factory.h"

#include "arc/cpp/runtime/node.h"
#include "arc/cpp/runtime/node_state.h"
#include "arc/cpp/runtime/runtime.h"

namespace arc {

std::pair<std::unique_ptr<Node>, xerrors::Error>
WASMNodeFactory::create(const NodeFactoryConfig& cfg) {
    // Check if this node type corresponds to a WASM function in the IR
    const auto* fn = cfg.ir.find_function(cfg.ir_node.type);
    if (!fn) {
        // Not a WASM function - let another factory handle it
        return {nullptr, xerrors::Error("NOT_FOUND")};
    }

    // Get incoming edges for this node (from other node outputs)
    std::vector<Edge> input_edges = cfg.state.incoming_edges(cfg.ir_node.key);

    // Build output handles for this node (this node + each output param)
    std::vector<Handle> output_handles;
    for (const auto& param : cfg.ir_node.outputs.keys) {
        output_handles.push_back(Handle{cfg.ir_node.key, param});
    }

    // Create NodeState with edges and handles
    auto node_state = std::make_unique<NodeState>(
        &cfg.state,
        cfg.ir_node.key,
        input_edges,
        output_handles
    );

    // Find WASM function instance
    auto [wasm_func, func_err] = runtime_.find_function(cfg.ir_node.type);
    if (func_err) {
        return {nullptr, xerrors::Error(
            func_err,
            "Failed to find WASM function '" + cfg.ir_node.type +
            "' for node '" + cfg.ir_node.key + "': " + func_err.data)};
    }

    // Create WASMNode (takes ownership of node_state)
    // Pass output parameter names for change tracking
    auto wasm_node = std::make_unique<WASMNode>(
        cfg.ir_node.key,
        std::move(node_state),
        &runtime_,
        wasm_func,
        cfg.ir_node.outputs.keys  // Output parameter names
    );

    return {std::move(wasm_node), xerrors::NIL};
}

}  // namespace arc
