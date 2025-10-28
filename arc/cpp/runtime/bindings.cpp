// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/bindings.h"

#include "arc/cpp/runtime/node_state.h"

namespace arc {

namespace host {

// Helper to get NodeState from execution environment
static NodeState *get_node_state(wasm_exec_env_t exec_env) {
    return static_cast<NodeState *>(wasm_runtime_get_user_data(exec_env));
}

// ============================================================================
// Channel Read Operations
// ============================================================================

int32_t channel_read_i32(wasm_exec_env_t exec_env, int32_t channel_id) {
    auto *node_state = get_node_state(exec_env);
    auto [value, err] = node_state->read_channel(channel_id);
    if (err) return 0;  // Default on error
    if (std::holds_alternative<int32_t>(value)) {
        return std::get<int32_t>(value);
    }
    return 0;  // Type mismatch
}

int64_t channel_read_i64(wasm_exec_env_t exec_env, int32_t channel_id) {
    auto *node_state = get_node_state(exec_env);
    auto [value, err] = node_state->read_channel(channel_id);
    if (err) return 0;
    if (std::holds_alternative<int64_t>(value)) {
        return std::get<int64_t>(value);
    }
    return 0;
}

float channel_read_f32(wasm_exec_env_t exec_env, int32_t channel_id) {
    auto *node_state = get_node_state(exec_env);
    auto [value, err] = node_state->read_channel(channel_id);
    if (err) return 0.0f;
    if (std::holds_alternative<float>(value)) {
        return std::get<float>(value);
    }
    return 0.0f;
}

double channel_read_f64(wasm_exec_env_t exec_env, int32_t channel_id) {
    auto *node_state = get_node_state(exec_env);
    auto [value, err] = node_state->read_channel(channel_id);
    if (err) return 0.0;
    if (std::holds_alternative<double>(value)) {
        return std::get<double>(value);
    }
    return 0.0;
}

// ============================================================================
// Channel Write Operations
// ============================================================================

void channel_write_i32(wasm_exec_env_t exec_env, int32_t channel_id, int32_t value) {
    auto *node_state = get_node_state(exec_env);
    node_state->write_channel(channel_id, value);
}

void channel_write_i64(wasm_exec_env_t exec_env, int32_t channel_id, int64_t value) {
    auto *node_state = get_node_state(exec_env);
    node_state->write_channel(channel_id, value);
}

void channel_write_f32(wasm_exec_env_t exec_env, int32_t channel_id, float value) {
    auto *node_state = get_node_state(exec_env);
    node_state->write_channel(channel_id, value);
}

void channel_write_f64(wasm_exec_env_t exec_env, int32_t channel_id, double value) {
    auto *node_state = get_node_state(exec_env);
    node_state->write_channel(channel_id, value);
}

// ============================================================================
// State Variable Load Operations
// ============================================================================

int32_t state_load_i32(wasm_exec_env_t exec_env, int32_t var_id, int32_t init_value) {
    auto *node_state = get_node_state(exec_env);
    return node_state->load_state_var(static_cast<uint32_t>(var_id), init_value);
}

int64_t state_load_i64(wasm_exec_env_t exec_env, int32_t var_id, int64_t init_value) {
    auto *node_state = get_node_state(exec_env);
    return node_state->load_state_var(static_cast<uint32_t>(var_id), init_value);
}

float state_load_f32(wasm_exec_env_t exec_env, int32_t var_id, float init_value) {
    auto *node_state = get_node_state(exec_env);
    return node_state->load_state_var(static_cast<uint32_t>(var_id), init_value);
}

double state_load_f64(wasm_exec_env_t exec_env, int32_t var_id, double init_value) {
    auto *node_state = get_node_state(exec_env);
    return node_state->load_state_var(static_cast<uint32_t>(var_id), init_value);
}

// ============================================================================
// State Variable Store Operations
// ============================================================================

void state_store_i32(wasm_exec_env_t exec_env, int32_t var_id, int32_t value) {
    auto *node_state = get_node_state(exec_env);
    node_state->store_state_var(static_cast<uint32_t>(var_id), value);
}

void state_store_i64(wasm_exec_env_t exec_env, int32_t var_id, int64_t value) {
    auto *node_state = get_node_state(exec_env);
    node_state->store_state_var(static_cast<uint32_t>(var_id), value);
}

void state_store_f32(wasm_exec_env_t exec_env, int32_t var_id, float value) {
    auto *node_state = get_node_state(exec_env);
    node_state->store_state_var(static_cast<uint32_t>(var_id), value);
}

void state_store_f64(wasm_exec_env_t exec_env, int32_t var_id, double value) {
    auto *node_state = get_node_state(exec_env);
    node_state->store_state_var(static_cast<uint32_t>(var_id), value);
}

// ============================================================================
// Built-in Functions
// ============================================================================

int64_t now(wasm_exec_env_t exec_env) {
    (void)exec_env;  // Unused
    return telem::TimeStamp::now().nanoseconds();
}

void panic(wasm_exec_env_t exec_env, int32_t msg_ptr, int32_t msg_len) {
    auto *module_inst = wasm_runtime_get_module_inst(exec_env);

    // Validate memory access
    if (!wasm_runtime_validate_app_addr(module_inst, msg_ptr, msg_len)) {
        wasm_runtime_set_exception(module_inst, "panic: invalid memory access");
        return;
    }

    // Get message from WASM linear memory
    const char *msg = reinterpret_cast<const char *>(
        wasm_runtime_addr_app_to_native(module_inst, msg_ptr)
    );

    // Set exception with message
    std::string error_msg = "Arc panic: ";
    error_msg.append(msg, msg_len);
    wasm_runtime_set_exception(module_inst, error_msg.c_str());
}

}  // namespace host

// ============================================================================
// Host Function Registration
// ============================================================================

bool register_host_functions(wasm_module_inst_t module_inst) {
    // Define native symbols for WAMR
    static NativeSymbol native_symbols[] = {
        // Channel read operations (signature: (i)i/I/f/F)
        {"channel_read_i32", reinterpret_cast<void *>(host::channel_read_i32), "(i)i",
         nullptr},
        {"channel_read_i64", reinterpret_cast<void *>(host::channel_read_i64), "(i)I",
         nullptr},
        {"channel_read_f32", reinterpret_cast<void *>(host::channel_read_f32), "(i)f",
         nullptr},
        {"channel_read_f64", reinterpret_cast<void *>(host::channel_read_f64), "(i)F",
         nullptr},

        // Channel write operations (signature: (ii)v, (iI)v, (if)v, (iF)v)
        {"channel_write_i32", reinterpret_cast<void *>(host::channel_write_i32), "(ii)v",
         nullptr},
        {"channel_write_i64", reinterpret_cast<void *>(host::channel_write_i64), "(iI)v",
         nullptr},
        {"channel_write_f32", reinterpret_cast<void *>(host::channel_write_f32), "(if)v",
         nullptr},
        {"channel_write_f64", reinterpret_cast<void *>(host::channel_write_f64), "(iF)v",
         nullptr},

        // State load operations (signature: (ii)i/I, (if)f, (iF)F)
        {"state_load_i32", reinterpret_cast<void *>(host::state_load_i32), "(ii)i",
         nullptr},
        {"state_load_i64", reinterpret_cast<void *>(host::state_load_i64), "(iI)I",
         nullptr},
        {"state_load_f32", reinterpret_cast<void *>(host::state_load_f32), "(if)f",
         nullptr},
        {"state_load_f64", reinterpret_cast<void *>(host::state_load_f64), "(iF)F",
         nullptr},

        // State store operations (signature: (ii)v, (iI)v, (if)v, (iF)v)
        {"state_store_i32", reinterpret_cast<void *>(host::state_store_i32), "(ii)v",
         nullptr},
        {"state_store_i64", reinterpret_cast<void *>(host::state_store_i64), "(iI)v",
         nullptr},
        {"state_store_f32", reinterpret_cast<void *>(host::state_store_f32), "(if)v",
         nullptr},
        {"state_store_f64", reinterpret_cast<void *>(host::state_store_f64), "(iF)v",
         nullptr},

        // Built-in functions
        {"now", reinterpret_cast<void *>(host::now), "()I", nullptr},
        {"panic", reinterpret_cast<void *>(host::panic), "(ii)v", nullptr},
    };

    constexpr uint32_t num_symbols = sizeof(native_symbols) / sizeof(NativeSymbol);

    // Register native symbols with module instance
    if (!wasm_runtime_register_natives("env",  // Import module name
                                       native_symbols, num_symbols)) {
        return false;
    }

    return true;
}

}  // namespace arc
