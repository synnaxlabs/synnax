// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include <set>

#include <nlohmann/json.hpp>

#include "x/cpp/telem/telem.h"
#include "x/cpp/xjson/xjson.h"

#include "arc/cpp/runtime/core/node.h"
#include "arc/cpp/runtime/factory/factory.h"
#include "arc/cpp/runtime/module/loader.h"
#include "arc/cpp/runtime/nodes/interval/factory.h"
#include "arc/cpp/runtime/nodes/wasm/factory.h"
#include "arc/cpp/runtime/wasm/bindings.h"

namespace arc { namespace module {

std::pair<AssembledRuntime, xerrors::Error> Loader::load(const Module &module) {
    AssembledRuntime assembled;

    // 1. Create queues with runtime-configured capacity
    assembled.input_queue = std::make_unique<queue::SPSC<ChannelUpdate>>(
        AssembledRuntime::DEFAULT_QUEUE_CAPACITY
    );
    assembled.output_queue = std::make_unique<queue::SPSC<ChannelOutput>>(
        AssembledRuntime::DEFAULT_QUEUE_CAPACITY
    );

    // 2. Create state
    assembled.state = std::make_unique<State>(
        assembled.input_queue.get(),
        assembled.output_queue.get()
    );

    // 3. Extract and register channels from IR
    auto channel_keys = extract_channel_keys(module.ir);
    for (auto channel_key: channel_keys) {
        // Find the type for this channel by scanning nodes
        TypeKind type_kind = TypeKind::Invalid;

        for (const auto &node: module.ir.nodes) {
            // Check read channels
            for (const auto &[key, param]: node.channels.read) {
                if (key == channel_key) {
                    // Find type from inputs
                    auto *param_type = node.inputs.get(param);
                    if (param_type) {
                        type_kind = param_type->kind;
                        break;
                    }
                }
            }

            // Check write channels
            for (const auto &[param, key]: node.channels.write) {
                if (key == channel_key) {
                    // Find type from outputs
                    auto *param_type = node.outputs.get(param);
                    if (param_type) {
                        type_kind = param_type->kind;
                        break;
                    }
                }
            }

            if (type_kind != TypeKind::Invalid) break;
        }

        // Convert TypeKind to DataType
        telem::DataType dt = telem::UNKNOWN_T;
        switch (type_kind) {
            case TypeKind::I32:
                dt = telem::INT32_T;
                break;
            case TypeKind::I64:
                dt = telem::INT64_T;
                break;
            case TypeKind::F32:
                dt = telem::FLOAT32_T;
                break;
            case TypeKind::F64:
                dt = telem::FLOAT64_T;
                break;
            case TypeKind::TimeStamp:
                dt = telem::TIMESTAMP_T;
                break;
            // Add more type mappings as needed
            default:
                dt = telem::FLOAT64_T; // Default fallback
        }

        assembled.state->register_channel(channel_key, dt);
    }

    // 4. Register nodes in state (for metadata)
    for (const auto &ir_node: module.ir.nodes) {
        NodeMetadata meta;
        meta.key = ir_node.key;
        meta.type = ir_node.type;
        meta.input_params = ir_node.inputs.keys;
        meta.output_params = ir_node.outputs.keys;

        // Extract channel keys
        for (const auto &[chan_key, param]: ir_node.channels.read) {
            meta.read_channels.push_back(chan_key);
        }

        for (const auto &[param, chan_key]: ir_node.channels.write) {
            meta.write_channels.push_back(chan_key);
        }

        assembled.state->register_node(meta);
    }

    // 5. Register edges from IR (both in state and scheduler)
    // State needs edges for temporal alignment
    // Scheduler needs edges for per-output change propagation
    for (const auto &ir_edge: module.ir.edges) {
        Edge edge{
            Handle{ir_edge.source.node, ir_edge.source.param},
            Handle{ir_edge.target.node, ir_edge.target.param}
        };
        assembled.state->add_edge(edge);
    }

    // 6. Initialize runtime
    assembled.runtime = std::make_unique<Runtime>();

    auto init_err = Runtime::initialize_runtime();
    if (init_err) { return {AssembledRuntime{}, init_err}; }

    // 7. Load WASM module (if bytecode provided)
    if (!module.wasm.empty()) {
        auto load_err = assembled.runtime->load_aot_module(module.wasm);
        if (load_err) { return {AssembledRuntime{}, load_err}; }

        // 8. Instantiate WASM module with fixed memory
        auto inst_err = assembled.runtime->instantiate(64 * 1024, 0);
        if (inst_err) { return {AssembledRuntime{}, inst_err}; }
    }

    // 9. Register host functions
    // TODO: This should be done before instantiation in real WAMR
    // For now, we'll register them globally

    // 10. Create scheduler
    assembled.scheduler = std::make_unique<Scheduler>(assembled.state.get());

    // 10a. Register edges in scheduler for per-output change propagation
    for (const auto &ir_edge: module.ir.edges) {
        assembled.scheduler->register_outgoing_edge(
            ir_edge.source.node,
            ir_edge.source.param,
            ir_edge.target.node
        );
    }

    // 10b. Scan for interval nodes and create TimeWheel if any exist
    std::vector<uint64_t> interval_periods;
    for (const auto &ir_node: module.ir.nodes) {
        if (ir_node.type == "interval") {
            // Extract period from config_values
            if (ir_node.config_values.count("period") > 0) {
                try {
                    uint64_t period_ns = ir_node.config_values.at("period")
                                             .get<uint64_t>();
                    interval_periods.push_back(period_ns);
                } catch (const std::exception &e) {
                    return {
                        AssembledRuntime{},
                        xerrors::Error("arc.module.invalid_interval_period", e.what())
                    };
                }
            }
        }
    }

    // Create TimeWheel if intervals exist
    if (!interval_periods.empty()) {
        uint64_t base_period = TimeWheel::calculate_base_period(interval_periods);
        assembled.time_wheel = std::make_unique<TimeWheel>(base_period);
    }

    // 11. Create node factory with all registered node types
    MultiFactory factory;
    factory.add(std::make_unique<interval::Factory>());
    if (assembled.runtime) {
        factory.add(std::make_unique<wasm::Factory>(*assembled.runtime));
    }

    // 12. Create nodes using factory pattern
    for (const auto &ir_node: module.ir.nodes) {
        // Find stratum for this node
        size_t stratum = 0;
        for (size_t i = 0; i < module.ir.strata.size(); i++) {
            for (const auto &node_key: module.ir.strata[i]) {
                if (node_key == ir_node.key) {
                    stratum = i;
                    break;
                }
            }
        }

        // Create node using factory
        NodeFactoryConfig cfg{ir_node, *assembled.state, module.ir};

        auto [node, create_err] = factory.create(cfg);
        if (create_err) {
            // Skip nodes with no matching factory (forward compatibility)
            if (create_err.type == "NOT_FOUND") { continue; }
            // Real error - fail the load
            return {
                AssembledRuntime{},
                xerrors::Error(
                    create_err,
                    "Failed to create node '" + ir_node.key + "' of type '" +
                        ir_node.type + "': " + create_err.data
                )
            };
        }

        // Register with scheduler (scheduler takes ownership of node)
        auto reg_err = assembled.scheduler
                           ->register_node(ir_node.key, std::move(node), stratum);
        if (reg_err) { return {AssembledRuntime{}, reg_err}; }
    }

    return {std::move(assembled), xerrors::NIL};
}

std::vector<ChannelKey> Loader::extract_channel_keys(const IR &ir) {
    std::set<ChannelKey> keys;

    for (const auto &node: ir.nodes) {
        // Collect read channels
        for (const auto &[key, param]: node.channels.read) {
            keys.insert(key);
        }

        // Collect write channels
        for (const auto &[param, key]: node.channels.write) {
            keys.insert(key);
        }
    }

    return std::vector<ChannelKey>(keys.begin(), keys.end());
}

TypeKind Loader::get_channel_type(const Node &node, ChannelKey channel_key) {
    // Check read channels
    for (const auto &[key, param]: node.channels.read) {
        if (key == channel_key) {
            auto *type = node.inputs.get(param);
            if (type) return type->kind;
        }
    }

    // Check write channels
    for (const auto &[param, key]: node.channels.write) {
        if (key == channel_key) {
            auto *type = node.outputs.get(param);
            if (type) return type->kind;
        }
    }

    return TypeKind::Invalid;
}

} // namespace module
} // namespace arc
