// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <unordered_map>

#include "arc/cpp/runtime/state/state.h"
#include "vendor/wamr/include/wasm_export.h"

namespace arc::runtime::wasm::bindings {

/// Runtime provides the actual implementation of Arc runtime functions.
/// This is the "business logic" layer that the bindings call.
class Runtime {
    [[maybe_unused]] state::State *state;
    wasm_module_inst_t module_inst;

    // String storage - handle to string mapping
    std::unordered_map<uint32_t, std::string> strings;
    uint32_t string_handle_counter;

    // State storage for stateful variables
    // Key: (funcID << 32) | varID
    std::unordered_map<uint64_t, uint8_t> state_u8;
    std::unordered_map<uint64_t, uint16_t> state_u16;
    std::unordered_map<uint64_t, uint32_t> state_u32;
    std::unordered_map<uint64_t, uint64_t> state_u64;
    std::unordered_map<uint64_t, int8_t> state_i8;
    std::unordered_map<uint64_t, int16_t> state_i16;
    std::unordered_map<uint64_t, int32_t> state_i32;
    std::unordered_map<uint64_t, int64_t> state_i64;
    std::unordered_map<uint64_t, float> state_f32;
    std::unordered_map<uint64_t, double> state_f64;
    std::unordered_map<uint64_t, std::string> state_string;

    static uint64_t state_key(uint32_t func_id, uint32_t var_id) {
        return (static_cast<uint64_t>(func_id) << 32) | static_cast<uint64_t>(var_id);
    }

public:
    Runtime(state::State *state, wasm_module_inst_t module_inst);

    // Set the module instance (used after instantiation)
    void set_module_inst(wasm_module_inst_t inst) { this->module_inst = inst; }

    // ===== Channel Operations =====
    uint8_t channel_read_u8(uint32_t channel_id);
    void channel_write_u8(uint32_t channel_id, uint8_t value);
    uint8_t channel_blocking_read_u8(uint32_t channel_id);

    uint16_t channel_read_u16(uint32_t channel_id);
    void channel_write_u16(uint32_t channel_id, uint16_t value);
    uint16_t channel_blocking_read_u16(uint32_t channel_id);

    uint32_t channel_read_u32(uint32_t channel_id);
    void channel_write_u32(uint32_t channel_id, uint32_t value);
    uint32_t channel_blocking_read_u32(uint32_t channel_id);

    uint64_t channel_read_u64(uint32_t channel_id);
    void channel_write_u64(uint32_t channel_id, uint64_t value);
    uint64_t channel_blocking_read_u64(uint32_t channel_id);

    int8_t channel_read_i8(uint32_t channel_id);
    void channel_write_i8(uint32_t channel_id, int8_t value);
    int8_t channel_blocking_read_i8(uint32_t channel_id);

    int16_t channel_read_i16(uint32_t channel_id);
    void channel_write_i16(uint32_t channel_id, int16_t value);
    int16_t channel_blocking_read_i16(uint32_t channel_id);

    int32_t channel_read_i32(uint32_t channel_id);
    void channel_write_i32(uint32_t channel_id, int32_t value);
    int32_t channel_blocking_read_i32(uint32_t channel_id);

    int64_t channel_read_i64(uint32_t channel_id);
    void channel_write_i64(uint32_t channel_id, int64_t value);
    int64_t channel_blocking_read_i64(uint32_t channel_id);

    float channel_read_f32(uint32_t channel_id);
    void channel_write_f32(uint32_t channel_id, float value);
    float channel_blocking_read_f32(uint32_t channel_id);

    double channel_read_f64(uint32_t channel_id);
    void channel_write_f64(uint32_t channel_id, double value);
    double channel_blocking_read_f64(uint32_t channel_id);

    uint32_t channel_read_str(uint32_t channel_id);
    void channel_write_str(uint32_t channel_id, uint32_t str_handle);
    uint32_t channel_blocking_read_str(uint32_t channel_id);

    // ===== State Operations =====
    uint8_t state_load_u8(uint32_t func_id, uint32_t var_id, uint8_t init_value);
    void state_store_u8(uint32_t func_id, uint32_t var_id, uint8_t value);

    uint16_t state_load_u16(uint32_t func_id, uint32_t var_id, uint16_t init_value);
    void state_store_u16(uint32_t func_id, uint32_t var_id, uint16_t value);

    uint32_t state_load_u32(uint32_t func_id, uint32_t var_id, uint32_t init_value);
    void state_store_u32(uint32_t func_id, uint32_t var_id, uint32_t value);

    uint64_t state_load_u64(uint32_t func_id, uint32_t var_id, uint64_t init_value);
    void state_store_u64(uint32_t func_id, uint32_t var_id, uint64_t value);

    int8_t state_load_i8(uint32_t func_id, uint32_t var_id, int8_t init_value);
    void state_store_i8(uint32_t func_id, uint32_t var_id, int8_t value);

    int16_t state_load_i16(uint32_t func_id, uint32_t var_id, int16_t init_value);
    void state_store_i16(uint32_t func_id, uint32_t var_id, int16_t value);

    int32_t state_load_i32(uint32_t func_id, uint32_t var_id, int32_t init_value);
    void state_store_i32(uint32_t func_id, uint32_t var_id, int32_t value);

    int64_t state_load_i64(uint32_t func_id, uint32_t var_id, int64_t init_value);
    void state_store_i64(uint32_t func_id, uint32_t var_id, int64_t value);

    float state_load_f32(uint32_t func_id, uint32_t var_id, float init_value);
    void state_store_f32(uint32_t func_id, uint32_t var_id, float value);

    double state_load_f64(uint32_t func_id, uint32_t var_id, double init_value);
    void state_store_f64(uint32_t func_id, uint32_t var_id, double value);

    uint32_t state_load_str(uint32_t func_id, uint32_t var_id, uint32_t init_handle);
    void state_store_str(uint32_t func_id, uint32_t var_id, uint32_t str_handle);

    // ===== Series Operations =====
    // For now, these are stubs that panic - series operations are complex
    uint32_t series_create_empty(uint32_t length, uint32_t element_size);
    uint64_t series_len(uint32_t handle);
    uint32_t series_slice(uint32_t handle, uint32_t start, uint32_t end);

    // ===== String Operations =====
    uint32_t string_from_literal(uint32_t ptr, uint32_t len);
    uint32_t string_len(uint32_t handle);
    uint32_t string_equal(uint32_t handle1, uint32_t handle2);

    // ===== Generic Operations =====
    uint64_t now();
    uint64_t len(uint32_t handle);
    void panic(uint32_t ptr, uint32_t len);

    // ===== Math Operations =====
    float math_pow_f32(float base, float exp);
    double math_pow_f64(double base, double exp);

    uint8_t math_int_pow_u8(uint8_t base, uint8_t exp);
    uint16_t math_int_pow_u16(uint16_t base, uint16_t exp);
    uint32_t math_int_pow_u32(uint32_t base, uint32_t exp);
    uint64_t math_int_pow_u64(uint64_t base, uint64_t exp);

    int8_t math_int_pow_i8(int8_t base, int8_t exp);
    int16_t math_int_pow_i16(int16_t base, int16_t exp);
    int32_t math_int_pow_i32(int32_t base, int32_t exp);
    int64_t math_int_pow_i64(int64_t base, int64_t exp);
};

/// Register all native functions with the WAMR runtime.
/// This must be called before wasm_runtime_instantiate().
void register_natives(Runtime *runtime);

/// Get the Runtime instance from the execution environment.
/// Used by native function wrappers to access the Runtime.
Runtime *get_runtime(wasm_exec_env_t exec_env);

/// Set the Runtime instance for the module instance.
/// Called after instantiation.
void set_runtime(wasm_module_inst_t module_inst, Runtime *runtime);

}