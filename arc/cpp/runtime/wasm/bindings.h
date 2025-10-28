// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include "wasm_export.h"

namespace arc {

namespace host {

// ============================================================================
// Channel Read Operations
// ============================================================================

/// @brief Read int32 value from channel.
int32_t channel_read_i32(wasm_exec_env_t exec_env, int32_t channel_id);

/// @brief Read int64 value from channel.
int64_t channel_read_i64(wasm_exec_env_t exec_env, int32_t channel_id);

/// @brief Read float32 value from channel.
float channel_read_f32(wasm_exec_env_t exec_env, int32_t channel_id);

/// @brief Read float64 value from channel.
double channel_read_f64(wasm_exec_env_t exec_env, int32_t channel_id);

// ============================================================================
// Channel Write Operations
// ============================================================================

/// @brief Write int32 value to channel.
void channel_write_i32(wasm_exec_env_t exec_env, int32_t channel_id, int32_t value);

/// @brief Write int64 value to channel.
void channel_write_i64(wasm_exec_env_t exec_env, int32_t channel_id, int64_t value);

/// @brief Write float32 value to channel.
void channel_write_f32(wasm_exec_env_t exec_env, int32_t channel_id, float value);

/// @brief Write float64 value to channel.
void channel_write_f64(wasm_exec_env_t exec_env, int32_t channel_id, double value);

// ============================================================================
// State Variable Load Operations
// ============================================================================

/// @brief Load int32 state variable with initialization value.
int32_t state_load_i32(wasm_exec_env_t exec_env, int32_t var_id, int32_t init_value);

/// @brief Load int64 state variable with initialization value.
int64_t state_load_i64(wasm_exec_env_t exec_env, int32_t var_id, int64_t init_value);

/// @brief Load float32 state variable with initialization value.
float state_load_f32(wasm_exec_env_t exec_env, int32_t var_id, float init_value);

/// @brief Load float64 state variable with initialization value.
double state_load_f64(wasm_exec_env_t exec_env, int32_t var_id, double init_value);

// ============================================================================
// State Variable Store Operations
// ============================================================================

/// @brief Store int32 state variable.
void state_store_i32(wasm_exec_env_t exec_env, int32_t var_id, int32_t value);

/// @brief Store int64 state variable.
void state_store_i64(wasm_exec_env_t exec_env, int32_t var_id, int64_t value);

/// @brief Store float32 state variable.
void state_store_f32(wasm_exec_env_t exec_env, int32_t var_id, float value);

/// @brief Store float64 state variable.
void state_store_f64(wasm_exec_env_t exec_env, int32_t var_id, double value);

// ============================================================================
// Built-in Functions
// ============================================================================

/// @brief Get current timestamp in nanoseconds.
int64_t now(wasm_exec_env_t exec_env);

/// @brief Panic with error message (terminates execution).
void panic(wasm_exec_env_t exec_env, int32_t msg_ptr, int32_t msg_len);

}  // namespace host

/// @brief Register all host functions with WAMR module instance.
/// @param module_inst WAMR module instance.
/// @return true on success, false on failure.
bool register_host_functions(wasm_module_inst_t module_inst);

}  // namespace arc
