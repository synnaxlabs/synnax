// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <map>
#include <memory>
#include <string>
#include <vector>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/core/types.h"
#include "arc/cpp/runtime/scheduler/scheduler.h"
#include "arc/cpp/runtime/scheduler/time_wheel.h"
#include "arc/cpp/runtime/state/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/runtime/wasm/runtime.h"

namespace arc { namespace module {

/// @brief Compiled Arc module (IR + WASM bytecode).
struct Module {
    ir::IR ir; ///< Intermediate representation
    std::vector<uint8_t> wasm; ///< WASM bytecode
    std::map<std::string, uint32_t> output_memory_bases; ///< Multi-output memory layout

    Module() = default;
    Module(ir::IR ir_, std::vector<uint8_t> wasm_):
        ir(std::move(ir_)), wasm(std::move(wasm_)) {}
};

/// @brief Assembled Arc runtime ready for execution.
struct AssembledRuntime {
    /// Default queue capacity (can be configured if needed)
    static constexpr size_t DEFAULT_QUEUE_CAPACITY = 1024;

    std::unique_ptr<Runtime> runtime; ///< WASM runtime
    std::unique_ptr<Scheduler> scheduler; ///< Execution scheduler
    std::unique_ptr<State> state; ///< Runtime state
    std::unique_ptr<TimeWheel> time_wheel; ///< Interval timer (nullptr if no intervals)

    /// I/O queues (capacity set at construction)
    std::unique_ptr<queue::SPSC<ChannelUpdate>> input_queue; ///< I/O → RT
    std::unique_ptr<queue::SPSC<ChannelOutput>> output_queue; ///< RT → I/O

    AssembledRuntime() = default;

    /// @brief Check if runtime is ready to execute.
    bool is_ready() const {
        return runtime && scheduler && state && runtime->is_ready();
    }

    /// @brief Execute one scheduler cycle.
    /// @return Error status (NIL on success).
    xerrors::Error next() {
        if (!scheduler) return xerrors::Error("arc.runtime.not_ready");
        return scheduler->next();
    }

    /// @brief Check if scheduler should execute based on triggers.
    ///
    /// Returns true if either:
    /// - Time trigger: TimeWheel indicates base period has elapsed
    /// - Data trigger: Input queue has new channel data
    ///
    /// @return true if scheduler->next() should be called.
    bool should_execute() const {
        bool time_trigger = time_wheel && time_wheel->should_tick();
        bool data_trigger = input_queue && !input_queue->empty();
        return time_trigger || data_trigger;
    }
};

/// @brief Module loader and runtime assembler.
///
/// Loads compiled Arc modules and assembles them into executable runtimes.
/// Handles the bootstrap sequence:
/// 1. Parse IR from JSON
/// 2. Extract channel metadata from nodes
/// 3. Load WASM bytecode into runtime
/// 4. Register host functions
/// 5. Create WASMNode for each IR node
/// 6. Register nodes with scheduler at correct strata
/// 7. Wire up thread-safe queues
class Loader {
public:
    Loader() = default;

    /// @brief Load a module and assemble the runtime.
    /// @param ir_json JSON-serialized IR.
    /// @param wasm_bytes WASM bytecode (or AOT-compiled binary).
    /// @return Assembled runtime and error status.
    std::pair<AssembledRuntime, xerrors::Error>
    load(const std::string &ir_json, const std::vector<uint8_t> &wasm_bytes);

    /// @brief Load a module from Module structure.
    /// @param module Compiled Arc module.
    /// @return Assembled runtime and error status.
    std::pair<AssembledRuntime, xerrors::Error> load(const Module &module);

    /// @brief Extract all channel keys referenced in IR nodes.
    /// @param ir IR structure.
    /// @return Set of unique channel keys.
    std::vector<ChannelKey> extract_channel_keys(const ir::IR &ir);

    /// @brief Get channel data type from IR node.
    /// @param node IR node.
    /// @param channel_key Channel key to find.
    /// @return Data type or Invalid if not found.
    ir::TypeKind get_channel_type(const ir::Node &node, ChannelKey channel_key);
};

} // namespace module
} // namespace arc
