// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/nodes/wasm/node.h"

namespace arc {
namespace wasm {

Node::Node(std::string id,
           std::unique_ptr<state::Node> node_state,
           Runtime* runtime,
           wasm_function_inst_t function,
           std::vector<std::string> output_params)
    : id_(std::move(id)),
      node_state_(std::move(node_state)),
      runtime_(*runtime),
      function_(function),
      output_params_(std::move(output_params)) {}

xerrors::Error Node::execute(NodeContext& ctx) {
    // Refresh inputs with temporal alignment
    // Returns false if not ready (waiting for more data)
    if (!node_state_->refresh_inputs()) {
        return xerrors::NIL;  // Not ready, skip execution
    }

    // Set NodeState as user_data so host functions can access it
    runtime_.set_user_data(node_state_.get());

    // Call WASM function (no arguments for now - function uses host functions)
    // Host functions will access NodeState via wasm_runtime_get_user_data()
    std::span<const WasmValue> args(args_.data(), 0);  // Empty args
    std::span<WasmValue> results(results_.data(), 0);  // No results expected

    auto err = runtime_.call_function(function_, args, results);
    if (err) {
        return err;
    }

    // Mark all outputs changed (conservative approach)
    // TODO: WASM compiler should return per-output changed flags
    for (const auto& output_param : output_params_) {
        ctx.mark_changed(output_param);
    }

    return xerrors::NIL;
}

}  // namespace wasm
}  // namespace arc
