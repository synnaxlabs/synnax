// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "bindings.h"

#include <cmath>
#include <cstdio>
#include <cstring>
#include <chrono>

#include "wasmtime.h"  // For Wasmtime-specific APIs

namespace arc::runtime::wasm::bindings {

// Wasmtime uses Store context data for Runtime association

Runtime::Runtime(state::State *state, wasmtime_store_t *store)
    : state(state),
      store(store),
      string_handle_counter(1) {}

// ===== Channel Operations (Stubs) =====

uint8_t Runtime::channel_read_u8(uint32_t channel_id) {
    // TODO: Implement actual channel reading from state
    return 0;
}

void Runtime::channel_write_u8(uint32_t channel_id, uint8_t value) {
    // TODO: Implement actual channel writing to state
}

uint8_t Runtime::channel_blocking_read_u8(uint32_t channel_id) {
    return channel_read_u8(channel_id);
}

uint16_t Runtime::channel_read_u16(uint32_t channel_id) { return 0; }
void Runtime::channel_write_u16(uint32_t channel_id, uint16_t value) {}
uint16_t Runtime::channel_blocking_read_u16(uint32_t channel_id) {
    return channel_read_u16(channel_id);
}

uint32_t Runtime::channel_read_u32(uint32_t channel_id) { return 0; }
void Runtime::channel_write_u32(uint32_t channel_id, uint32_t value) {}
uint32_t Runtime::channel_blocking_read_u32(uint32_t channel_id) {
    return channel_read_u32(channel_id);
}

uint64_t Runtime::channel_read_u64(uint32_t channel_id) { return 0; }
void Runtime::channel_write_u64(uint32_t channel_id, uint64_t value) {}
uint64_t Runtime::channel_blocking_read_u64(uint32_t channel_id) {
    return channel_read_u64(channel_id);
}

int8_t Runtime::channel_read_i8(uint32_t channel_id) { return 0; }
void Runtime::channel_write_i8(uint32_t channel_id, int8_t value) {}
int8_t Runtime::channel_blocking_read_i8(uint32_t channel_id) {
    return channel_read_i8(channel_id);
}

int16_t Runtime::channel_read_i16(uint32_t channel_id) { return 0; }
void Runtime::channel_write_i16(uint32_t channel_id, int16_t value) {}
int16_t Runtime::channel_blocking_read_i16(uint32_t channel_id) {
    return channel_read_i16(channel_id);
}

int32_t Runtime::channel_read_i32(uint32_t channel_id) { return 0; }
void Runtime::channel_write_i32(uint32_t channel_id, int32_t value) {}
int32_t Runtime::channel_blocking_read_i32(uint32_t channel_id) {
    return channel_read_i32(channel_id);
}

int64_t Runtime::channel_read_i64(uint32_t channel_id) { return 0; }
void Runtime::channel_write_i64(uint32_t channel_id, int64_t value) {}
int64_t Runtime::channel_blocking_read_i64(uint32_t channel_id) {
    return channel_read_i64(channel_id);
}

float Runtime::channel_read_f32(uint32_t channel_id) { return 0.0f; }
void Runtime::channel_write_f32(uint32_t channel_id, float value) {}
float Runtime::channel_blocking_read_f32(uint32_t channel_id) {
    return channel_read_f32(channel_id);
}

double Runtime::channel_read_f64(uint32_t channel_id) { return 0.0; }
void Runtime::channel_write_f64(uint32_t channel_id, double value) {}
double Runtime::channel_blocking_read_f64(uint32_t channel_id) {
    return channel_read_f64(channel_id);
}

uint32_t Runtime::channel_read_str(uint32_t channel_id) { return 0; }
void Runtime::channel_write_str(uint32_t channel_id, uint32_t str_handle) {}
uint32_t Runtime::channel_blocking_read_str(uint32_t channel_id) {
    return channel_read_str(channel_id);
}

// ===== State Operations =====

uint8_t Runtime::state_load_u8(uint32_t func_id, uint32_t var_id, uint8_t init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_u8.find(key);
    if (it != state_u8.end()) return it->second;
    state_u8[key] = init_value;
    return init_value;
}

void Runtime::state_store_u8(uint32_t func_id, uint32_t var_id, uint8_t value) {
    state_u8[state_key(func_id, var_id)] = value;
}

uint16_t Runtime::state_load_u16(uint32_t func_id, uint32_t var_id, uint16_t init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_u16.find(key);
    if (it != state_u16.end()) return it->second;
    state_u16[key] = init_value;
    return init_value;
}

void Runtime::state_store_u16(uint32_t func_id, uint32_t var_id, uint16_t value) {
    state_u16[state_key(func_id, var_id)] = value;
}

uint32_t Runtime::state_load_u32(uint32_t func_id, uint32_t var_id, uint32_t init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_u32.find(key);
    if (it != state_u32.end()) return it->second;
    state_u32[key] = init_value;
    return init_value;
}

void Runtime::state_store_u32(uint32_t func_id, uint32_t var_id, uint32_t value) {
    state_u32[state_key(func_id, var_id)] = value;
}

uint64_t Runtime::state_load_u64(uint32_t func_id, uint32_t var_id, uint64_t init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_u64.find(key);
    if (it != state_u64.end()) return it->second;
    state_u64[key] = init_value;
    return init_value;
}

void Runtime::state_store_u64(uint32_t func_id, uint32_t var_id, uint64_t value) {
    state_u64[state_key(func_id, var_id)] = value;
}

int8_t Runtime::state_load_i8(uint32_t func_id, uint32_t var_id, int8_t init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_i8.find(key);
    if (it != state_i8.end()) return it->second;
    state_i8[key] = init_value;
    return init_value;
}

void Runtime::state_store_i8(uint32_t func_id, uint32_t var_id, int8_t value) {
    state_i8[state_key(func_id, var_id)] = value;
}

int16_t Runtime::state_load_i16(uint32_t func_id, uint32_t var_id, int16_t init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_i16.find(key);
    if (it != state_i16.end()) return it->second;
    state_i16[key] = init_value;
    return init_value;
}

void Runtime::state_store_i16(uint32_t func_id, uint32_t var_id, int16_t value) {
    state_i16[state_key(func_id, var_id)] = value;
}

int32_t Runtime::state_load_i32(uint32_t func_id, uint32_t var_id, int32_t init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_i32.find(key);
    if (it != state_i32.end()) return it->second;
    state_i32[key] = init_value;
    return init_value;
}

void Runtime::state_store_i32(uint32_t func_id, uint32_t var_id, int32_t value) {
    state_i32[state_key(func_id, var_id)] = value;
}

int64_t Runtime::state_load_i64(uint32_t func_id, uint32_t var_id, int64_t init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_i64.find(key);
    if (it != state_i64.end()) return it->second;
    state_i64[key] = init_value;
    return init_value;
}

void Runtime::state_store_i64(uint32_t func_id, uint32_t var_id, int64_t value) {
    state_i64[state_key(func_id, var_id)] = value;
}

float Runtime::state_load_f32(uint32_t func_id, uint32_t var_id, float init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_f32.find(key);
    if (it != state_f32.end()) return it->second;
    state_f32[key] = init_value;
    return init_value;
}

void Runtime::state_store_f32(uint32_t func_id, uint32_t var_id, float value) {
    state_f32[state_key(func_id, var_id)] = value;
}

double Runtime::state_load_f64(uint32_t func_id, uint32_t var_id, double init_value) {
    auto key = state_key(func_id, var_id);
    auto it = state_f64.find(key);
    if (it != state_f64.end()) return it->second;
    state_f64[key] = init_value;
    return init_value;
}

void Runtime::state_store_f64(uint32_t func_id, uint32_t var_id, double value) {
    state_f64[state_key(func_id, var_id)] = value;
}

uint32_t Runtime::state_load_str(uint32_t func_id, uint32_t var_id, uint32_t init_handle) {
    auto key = state_key(func_id, var_id);
    auto it = state_string.find(key);
    if (it != state_string.end()) {
        // Return a handle to the stored string
        strings[string_handle_counter] = it->second;
        return string_handle_counter++;
    }
    auto init_it = strings.find(init_handle);
    if (init_it != strings.end()) {
        state_string[key] = init_it->second;
    } else {
        state_string[key] = "";
    }
    strings[string_handle_counter] = state_string[key];
    return string_handle_counter++;
}

void Runtime::state_store_str(uint32_t func_id, uint32_t var_id, uint32_t str_handle) {
    auto it = strings.find(str_handle);
    if (it != strings.end()) {
        state_string[state_key(func_id, var_id)] = it->second;
    }
}

// ===== Series Operations (Stubs) =====

uint32_t Runtime::series_create_empty(uint32_t length, uint32_t element_size) {
    // TODO: Implement series creation
    return 0;
}

uint64_t Runtime::series_len(uint32_t handle) {
    // TODO: Implement series length
    return 0;
}

uint32_t Runtime::series_slice(uint32_t handle, uint32_t start, uint32_t end) {
    // TODO: Implement series slicing
    return 0;
}

// ===== String Operations =====

uint32_t Runtime::string_from_literal(uint32_t ptr, uint32_t len) {
    // TODO: Rewrite for Wasmtime - need memory access via store/instance
    // void *mem_data = wasm_memory_data(memory);
    // std::string str(static_cast<const char*>(mem_data + ptr), len);
    std::fprintf(stderr, "WARNING: string_from_literal not yet implemented for Wasmtime\n");
    return 0;
}

uint32_t Runtime::string_len(uint32_t handle) {
    auto it = strings.find(handle);
    if (it == strings.end()) return 0;
    return static_cast<uint32_t>(it->second.length());
}

uint32_t Runtime::string_equal(uint32_t handle1, uint32_t handle2) {
    auto it1 = strings.find(handle1);
    auto it2 = strings.find(handle2);

    if (it1 == strings.end() || it2 == strings.end()) return 0;
    return it1->second == it2->second ? 1 : 0;
}

// ===== Generic Operations =====

uint64_t Runtime::now() {
    auto now = std::chrono::system_clock::now();
    auto duration = now.time_since_epoch();
    auto micros = std::chrono::duration_cast<std::chrono::microseconds>(duration);
    return static_cast<uint64_t>(micros.count());
}

uint64_t Runtime::len(uint32_t handle) {
    // For now, assume it's a string handle
    return string_len(handle);
}

void Runtime::panic(uint32_t ptr, uint32_t len) {
    // TODO: Rewrite for Wasmtime - need memory access
    std::fprintf(stderr, "WASM panic: ptr=%u, len=%u (message not yet readable in Wasmtime)\n", ptr, len);
    std::abort();
}

// ===== Math Operations =====

float Runtime::math_pow_f32(float base, float exp) {
    return std::pow(base, exp);
}

double Runtime::math_pow_f64(double base, double exp) {
    return std::pow(base, exp);
}

template<typename T>
static T int_pow(T base, T exp) {
    if (exp == 0) return 1;
    T result = 1;
    for (T i = 0; i < exp; ++i) {
        result *= base;
    }
    return result;
}

uint8_t Runtime::math_int_pow_u8(uint8_t base, uint8_t exp) {
    return int_pow(base, exp);
}

uint16_t Runtime::math_int_pow_u16(uint16_t base, uint16_t exp) {
    return int_pow(base, exp);
}

uint32_t Runtime::math_int_pow_u32(uint32_t base, uint32_t exp) {
    return int_pow(base, exp);
}

uint64_t Runtime::math_int_pow_u64(uint64_t base, uint64_t exp) {
    return int_pow(base, exp);
}

int8_t Runtime::math_int_pow_i8(int8_t base, int8_t exp) {
    return int_pow(base, exp);
}

int16_t Runtime::math_int_pow_i16(int16_t base, int16_t exp) {
    return int_pow(base, exp);
}

int32_t Runtime::math_int_pow_i32(int32_t base, int32_t exp) {
    return int_pow(base, exp);
}

int64_t Runtime::math_int_pow_i64(int64_t base, int64_t exp) {
    return int_pow(base, exp);
}

/*
// ===== WAMR Native Function Wrappers (COMMENTED OUT FOR WASMTIME MIGRATION) =====
// These are the actual C functions that WAMR will call
// TODO: Rewrite these to use Wasmtime's wasm_func_new() API

#define GET_RUNTIME() \
    auto module_inst = wasm_runtime_get_module_inst(exec_env); \
    auto it = runtime_map.find(module_inst); \
    if (it == runtime_map.end()) { \
        std::fprintf(stderr, "Runtime not found for module instance\n"); \
        return; \
    } \
    Runtime *runtime = it->second;

#define GET_RUNTIME_RET(ret_val) \
    auto module_inst = wasm_runtime_get_module_inst(exec_env); \
    auto it = runtime_map.find(module_inst); \
    if (it == runtime_map.end()) { \
        std::fprintf(stderr, "Runtime not found for module instance\n"); \
        return ret_val; \
    } \
    Runtime *runtime = it->second;

// Channel operations
static uint32_t native_channel_read_u8(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_read_u8(channel_id);
}

static void native_channel_write_u8(wasm_exec_env_t exec_env, uint32_t channel_id, uint32_t value) {
    GET_RUNTIME()
    runtime->channel_write_u8(channel_id, static_cast<uint8_t>(value));
}

static uint32_t native_channel_blocking_read_u8(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_blocking_read_u8(channel_id);
}

static uint32_t native_channel_read_u16(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_read_u16(channel_id);
}

static void native_channel_write_u16(wasm_exec_env_t exec_env, uint32_t channel_id, uint32_t value) {
    GET_RUNTIME()
    runtime->channel_write_u16(channel_id, static_cast<uint16_t>(value));
}

static uint32_t native_channel_blocking_read_u16(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_blocking_read_u16(channel_id);
}

static uint32_t native_channel_read_u32(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_read_u32(channel_id);
}

static void native_channel_write_u32(wasm_exec_env_t exec_env, uint32_t channel_id, uint32_t value) {
    GET_RUNTIME()
    runtime->channel_write_u32(channel_id, value);
}

static uint32_t native_channel_blocking_read_u32(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_blocking_read_u32(channel_id);
}

static uint64_t native_channel_read_u64(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_read_u64(channel_id);
}

static void native_channel_write_u64(wasm_exec_env_t exec_env, uint32_t channel_id, uint64_t value) {
    GET_RUNTIME()
    runtime->channel_write_u64(channel_id, value);
}

static uint64_t native_channel_blocking_read_u64(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_blocking_read_u64(channel_id);
}

static uint32_t native_channel_read_i8(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(static_cast<uint8_t>(runtime->channel_read_i8(channel_id)));
}

static void native_channel_write_i8(wasm_exec_env_t exec_env, uint32_t channel_id, uint32_t value) {
    GET_RUNTIME()
    runtime->channel_write_i8(channel_id, static_cast<int8_t>(value));
}

static uint32_t native_channel_blocking_read_i8(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(static_cast<uint8_t>(runtime->channel_blocking_read_i8(channel_id)));
}

static uint32_t native_channel_read_i16(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(static_cast<uint16_t>(runtime->channel_read_i16(channel_id)));
}

static void native_channel_write_i16(wasm_exec_env_t exec_env, uint32_t channel_id, uint32_t value) {
    GET_RUNTIME()
    runtime->channel_write_i16(channel_id, static_cast<int16_t>(value));
}

static uint32_t native_channel_blocking_read_i16(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(static_cast<uint16_t>(runtime->channel_blocking_read_i16(channel_id)));
}

static uint32_t native_channel_read_i32(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(runtime->channel_read_i32(channel_id));
}

static void native_channel_write_i32(wasm_exec_env_t exec_env, uint32_t channel_id, uint32_t value) {
    GET_RUNTIME()
    runtime->channel_write_i32(channel_id, static_cast<int32_t>(value));
}

static uint32_t native_channel_blocking_read_i32(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(runtime->channel_blocking_read_i32(channel_id));
}

static uint64_t native_channel_read_i64(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return static_cast<uint64_t>(runtime->channel_read_i64(channel_id));
}

static void native_channel_write_i64(wasm_exec_env_t exec_env, uint32_t channel_id, uint64_t value) {
    GET_RUNTIME()
    runtime->channel_write_i64(channel_id, static_cast<int64_t>(value));
}

static uint64_t native_channel_blocking_read_i64(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return static_cast<uint64_t>(runtime->channel_blocking_read_i64(channel_id));
}

static float native_channel_read_f32(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0.0f)
    return runtime->channel_read_f32(channel_id);
}

static void native_channel_write_f32(wasm_exec_env_t exec_env, uint32_t channel_id, float value) {
    GET_RUNTIME()
    runtime->channel_write_f32(channel_id, value);
}

static float native_channel_blocking_read_f32(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0.0f)
    return runtime->channel_blocking_read_f32(channel_id);
}

static double native_channel_read_f64(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0.0)
    return runtime->channel_read_f64(channel_id);
}

static void native_channel_write_f64(wasm_exec_env_t exec_env, uint32_t channel_id, double value) {
    GET_RUNTIME()
    runtime->channel_write_f64(channel_id, value);
}

static double native_channel_blocking_read_f64(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0.0)
    return runtime->channel_blocking_read_f64(channel_id);
}

static uint32_t native_channel_read_str(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_read_str(channel_id);
}

static void native_channel_write_str(wasm_exec_env_t exec_env, uint32_t channel_id, uint32_t str_handle) {
    GET_RUNTIME()
    runtime->channel_write_str(channel_id, str_handle);
}

static uint32_t native_channel_blocking_read_str(wasm_exec_env_t exec_env, uint32_t channel_id) {
    GET_RUNTIME_RET(0)
    return runtime->channel_blocking_read_str(channel_id);
}

// State operations
static uint32_t native_state_load_u8(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t init_value) {
    GET_RUNTIME_RET(0)
    return runtime->state_load_u8(func_id, var_id, static_cast<uint8_t>(init_value));
}

static void native_state_store_u8(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t value) {
    GET_RUNTIME()
    runtime->state_store_u8(func_id, var_id, static_cast<uint8_t>(value));
}

static uint32_t native_state_load_u16(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t init_value) {
    GET_RUNTIME_RET(0)
    return runtime->state_load_u16(func_id, var_id, static_cast<uint16_t>(init_value));
}

static void native_state_store_u16(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t value) {
    GET_RUNTIME()
    runtime->state_store_u16(func_id, var_id, static_cast<uint16_t>(value));
}

static uint32_t native_state_load_u32(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t init_value) {
    GET_RUNTIME_RET(0)
    return runtime->state_load_u32(func_id, var_id, init_value);
}

static void native_state_store_u32(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t value) {
    GET_RUNTIME()
    runtime->state_store_u32(func_id, var_id, value);
}

static uint64_t native_state_load_u64(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint64_t init_value) {
    GET_RUNTIME_RET(0)
    return runtime->state_load_u64(func_id, var_id, init_value);
}

static void native_state_store_u64(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint64_t value) {
    GET_RUNTIME()
    runtime->state_store_u64(func_id, var_id, value);
}

static uint32_t native_state_load_i8(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t init_value) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(static_cast<uint8_t>(runtime->state_load_i8(func_id, var_id, static_cast<int8_t>(init_value))));
}

static void native_state_store_i8(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t value) {
    GET_RUNTIME()
    runtime->state_store_i8(func_id, var_id, static_cast<int8_t>(value));
}

static uint32_t native_state_load_i16(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t init_value) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(static_cast<uint16_t>(runtime->state_load_i16(func_id, var_id, static_cast<int16_t>(init_value))));
}

static void native_state_store_i16(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t value) {
    GET_RUNTIME()
    runtime->state_store_i16(func_id, var_id, static_cast<int16_t>(value));
}

static uint32_t native_state_load_i32(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t init_value) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(runtime->state_load_i32(func_id, var_id, static_cast<int32_t>(init_value)));
}

static void native_state_store_i32(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t value) {
    GET_RUNTIME()
    runtime->state_store_i32(func_id, var_id, static_cast<int32_t>(value));
}

static uint64_t native_state_load_i64(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint64_t init_value) {
    GET_RUNTIME_RET(0)
    return static_cast<uint64_t>(runtime->state_load_i64(func_id, var_id, static_cast<int64_t>(init_value)));
}

static void native_state_store_i64(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint64_t value) {
    GET_RUNTIME()
    runtime->state_store_i64(func_id, var_id, static_cast<int64_t>(value));
}

static float native_state_load_f32(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, float init_value) {
    GET_RUNTIME_RET(0.0f)
    return runtime->state_load_f32(func_id, var_id, init_value);
}

static void native_state_store_f32(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, float value) {
    GET_RUNTIME()
    runtime->state_store_f32(func_id, var_id, value);
}

static double native_state_load_f64(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, double init_value) {
    GET_RUNTIME_RET(0.0)
    return runtime->state_load_f64(func_id, var_id, init_value);
}

static void native_state_store_f64(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, double value) {
    GET_RUNTIME()
    runtime->state_store_f64(func_id, var_id, value);
}

static uint32_t native_state_load_str(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t init_handle) {
    GET_RUNTIME_RET(0)
    return runtime->state_load_str(func_id, var_id, init_handle);
}

static void native_state_store_str(wasm_exec_env_t exec_env, uint32_t func_id, uint32_t var_id, uint32_t str_handle) {
    GET_RUNTIME()
    runtime->state_store_str(func_id, var_id, str_handle);
}

// String operations
static uint32_t native_string_from_literal(wasm_exec_env_t exec_env, uint32_t ptr, uint32_t len) {
    GET_RUNTIME_RET(0)
    return runtime->string_from_literal(ptr, len);
}

static uint32_t native_string_len(wasm_exec_env_t exec_env, uint32_t handle) {
    GET_RUNTIME_RET(0)
    return runtime->string_len(handle);
}

static uint32_t native_string_equal(wasm_exec_env_t exec_env, uint32_t handle1, uint32_t handle2) {
    GET_RUNTIME_RET(0)
    return runtime->string_equal(handle1, handle2);
}

// Generic operations
static uint64_t native_now(wasm_exec_env_t exec_env) {
    GET_RUNTIME_RET(0)
    return runtime->now();
}

static uint64_t native_len(wasm_exec_env_t exec_env, uint32_t handle) {
    GET_RUNTIME_RET(0)
    return runtime->len(handle);
}

static void native_panic(wasm_exec_env_t exec_env, uint32_t ptr, uint32_t len) {
    GET_RUNTIME()
    runtime->panic(ptr, len);
}

// Math operations
static float native_math_pow_f32(wasm_exec_env_t exec_env, float base, float exp) {
    GET_RUNTIME_RET(0.0f)
    return runtime->math_pow_f32(base, exp);
}

static double native_math_pow_f64(wasm_exec_env_t exec_env, double base, double exp) {
    GET_RUNTIME_RET(0.0)
    return runtime->math_pow_f64(base, exp);
}

static uint32_t native_math_int_pow_u8(wasm_exec_env_t exec_env, uint32_t base, uint32_t exp) {
    GET_RUNTIME_RET(0)
    return runtime->math_int_pow_u8(static_cast<uint8_t>(base), static_cast<uint8_t>(exp));
}

static uint32_t native_math_int_pow_u16(wasm_exec_env_t exec_env, uint32_t base, uint32_t exp) {
    GET_RUNTIME_RET(0)
    return runtime->math_int_pow_u16(static_cast<uint16_t>(base), static_cast<uint16_t>(exp));
}

static uint32_t native_math_int_pow_u32(wasm_exec_env_t exec_env, uint32_t base, uint32_t exp) {
    GET_RUNTIME_RET(0)
    return runtime->math_int_pow_u32(base, exp);
}

static uint64_t native_math_int_pow_u64(wasm_exec_env_t exec_env, uint64_t base, uint64_t exp) {
    GET_RUNTIME_RET(0)
    return runtime->math_int_pow_u64(base, exp);
}

static uint32_t native_math_int_pow_i8(wasm_exec_env_t exec_env, uint32_t base, uint32_t exp) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(static_cast<uint8_t>(runtime->math_int_pow_i8(static_cast<int8_t>(base), static_cast<int8_t>(exp))));
}

static uint32_t native_math_int_pow_i16(wasm_exec_env_t exec_env, uint32_t base, uint32_t exp) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(static_cast<uint16_t>(runtime->math_int_pow_i16(static_cast<int16_t>(base), static_cast<int16_t>(exp))));
}

static uint32_t native_math_int_pow_i32(wasm_exec_env_t exec_env, uint32_t base, uint32_t exp) {
    GET_RUNTIME_RET(0)
    return static_cast<uint32_t>(runtime->math_int_pow_i32(static_cast<int32_t>(base), static_cast<int32_t>(exp)));
}

static uint64_t native_math_int_pow_i64(wasm_exec_env_t exec_env, uint64_t base, uint64_t exp) {
    GET_RUNTIME_RET(0)
    return static_cast<uint64_t>(runtime->math_int_pow_i64(static_cast<int64_t>(base), static_cast<int64_t>(exp)));
}

// Stub series operations - these just return 0 for now
static uint32_t native_series_create_empty_u8(wasm_exec_env_t exec_env, uint32_t length) { return 0; }
static void native_series_set_element_u8(wasm_exec_env_t exec_env, uint32_t handle, uint32_t index, uint32_t value) {}
static uint32_t native_series_index_u8(wasm_exec_env_t exec_env, uint32_t handle, uint32_t index) { return 0; }
static uint32_t native_series_element_add_u8(wasm_exec_env_t exec_env, uint32_t handle, uint32_t value) { return 0; }
static uint32_t native_series_element_mul_u8(wasm_exec_env_t exec_env, uint32_t handle, uint32_t value) { return 0; }
static uint32_t native_series_element_sub_u8(wasm_exec_env_t exec_env, uint32_t handle, uint32_t value) { return 0; }
static uint32_t native_series_element_div_u8(wasm_exec_env_t exec_env, uint32_t handle, uint32_t value) { return 0; }
static uint32_t native_series_series_add_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_series_mul_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_series_sub_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_series_div_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_compare_gt_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_compare_lt_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_compare_ge_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_compare_le_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_compare_eq_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }
static uint32_t native_series_compare_ne_u8(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }

// Repeat for all other series types (u16, u32, u64, i8, i16, i32, i64, f32, f64)
// For brevity, creating stubs for each
#define SERIES_STUBS(TYPE, SUFFIX) \
    static uint32_t native_series_create_empty_##SUFFIX(wasm_exec_env_t exec_env, uint32_t length) { return 0; } \
    static void native_series_set_element_##SUFFIX(wasm_exec_env_t exec_env, uint32_t handle, uint32_t index, TYPE value) {} \
    static TYPE native_series_index_##SUFFIX(wasm_exec_env_t exec_env, uint32_t handle, uint32_t index) { return 0; } \
    static uint32_t native_series_element_add_##SUFFIX(wasm_exec_env_t exec_env, uint32_t handle, TYPE value) { return 0; } \
    static uint32_t native_series_element_mul_##SUFFIX(wasm_exec_env_t exec_env, uint32_t handle, TYPE value) { return 0; } \
    static uint32_t native_series_element_sub_##SUFFIX(wasm_exec_env_t exec_env, uint32_t handle, TYPE value) { return 0; } \
    static uint32_t native_series_element_div_##SUFFIX(wasm_exec_env_t exec_env, uint32_t handle, TYPE value) { return 0; } \
    static uint32_t native_series_series_add_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_series_mul_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_series_sub_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_series_div_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_compare_gt_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_compare_lt_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_compare_ge_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_compare_le_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_compare_eq_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; } \
    static uint32_t native_series_compare_ne_##SUFFIX(wasm_exec_env_t exec_env, uint32_t h1, uint32_t h2) { return 0; }

SERIES_STUBS(uint32_t, u16)
SERIES_STUBS(uint32_t, u32)
SERIES_STUBS(uint64_t, u64)
SERIES_STUBS(uint32_t, i8)
SERIES_STUBS(uint32_t, i16)
SERIES_STUBS(uint32_t, i32)
SERIES_STUBS(uint64_t, i64)
SERIES_STUBS(float, f32)
SERIES_STUBS(double, f64)

static uint64_t native_series_len(wasm_exec_env_t exec_env, uint32_t handle) { return 0; }
static uint32_t native_series_slice(wasm_exec_env_t exec_env, uint32_t handle, uint32_t start, uint32_t end) { return 0; }

// ===== Registration =====

// WAMR signature strings:
// i = i32, I = i64, f = f32, F = f64
// (params)result format

static NativeSymbol native_symbols[] = {
    // Channel operations
    {"channel_read_u8", (void*)native_channel_read_u8, "(i)i", NULL},
    {"channel_write_u8", (void*)native_channel_write_u8, "(ii)", NULL},
    {"channel_blocking_read_u8", (void*)native_channel_blocking_read_u8, "(i)i", NULL},
    {"channel_read_u16", (void*)native_channel_read_u16, "(i)i", NULL},
    {"channel_write_u16", (void*)native_channel_write_u16, "(ii)", NULL},
    {"channel_blocking_read_u16", (void*)native_channel_blocking_read_u16, "(i)i", NULL},
    {"channel_read_u32", (void*)native_channel_read_u32, "(i)i", NULL},
    {"channel_write_u32", (void*)native_channel_write_u32, "(ii)", NULL},
    {"channel_blocking_read_u32", (void*)native_channel_blocking_read_u32, "(i)i", NULL},
    {"channel_read_u64", (void*)native_channel_read_u64, "(i)I", NULL},
    {"channel_write_u64", (void*)native_channel_write_u64, "(iI)", NULL},
    {"channel_blocking_read_u64", (void*)native_channel_blocking_read_u64, "(i)I", NULL},
    {"channel_read_i8", (void*)native_channel_read_i8, "(i)i", NULL},
    {"channel_write_i8", (void*)native_channel_write_i8, "(ii)", NULL},
    {"channel_blocking_read_i8", (void*)native_channel_blocking_read_i8, "(i)i", NULL},
    {"channel_read_i16", (void*)native_channel_read_i16, "(i)i", NULL},
    {"channel_write_i16", (void*)native_channel_write_i16, "(ii)", NULL},
    {"channel_blocking_read_i16", (void*)native_channel_blocking_read_i16, "(i)i", NULL},
    {"channel_read_i32", (void*)native_channel_read_i32, "(i)i", NULL},
    {"channel_write_i32", (void*)native_channel_write_i32, "(ii)", NULL},
    {"channel_blocking_read_i32", (void*)native_channel_blocking_read_i32, "(i)i", NULL},
    {"channel_read_i64", (void*)native_channel_read_i64, "(i)I", NULL},
    {"channel_write_i64", (void*)native_channel_write_i64, "(iI)", NULL},
    {"channel_blocking_read_i64", (void*)native_channel_blocking_read_i64, "(i)I", NULL},
    {"channel_read_f32", (void*)native_channel_read_f32, "(i)f", NULL},
    {"channel_write_f32", (void*)native_channel_write_f32, "(if)", NULL},
    {"channel_blocking_read_f32", (void*)native_channel_blocking_read_f32, "(i)f", NULL},
    {"channel_read_f64", (void*)native_channel_read_f64, "(i)F", NULL},
    {"channel_write_f64", (void*)native_channel_write_f64, "(iF)", NULL},
    {"channel_blocking_read_f64", (void*)native_channel_blocking_read_f64, "(i)F", NULL},
    {"channel_read_str", (void*)native_channel_read_str, "(i)i", NULL},
    {"channel_write_str", (void*)native_channel_write_str, "(ii)", NULL},
    {"channel_blocking_read_str", (void*)native_channel_blocking_read_str, "(i)i", NULL},

    // State operations
    {"state_load_u8", (void*)native_state_load_u8, "(iii)i", NULL},
    {"state_store_u8", (void*)native_state_store_u8, "(iii)", NULL},
    {"state_load_u16", (void*)native_state_load_u16, "(iii)i", NULL},
    {"state_store_u16", (void*)native_state_store_u16, "(iii)", NULL},
    {"state_load_u32", (void*)native_state_load_u32, "(iii)i", NULL},
    {"state_store_u32", (void*)native_state_store_u32, "(iii)", NULL},
    {"state_load_u64", (void*)native_state_load_u64, "(iiI)I", NULL},
    {"state_store_u64", (void*)native_state_store_u64, "(iiI)", NULL},
    {"state_load_i8", (void*)native_state_load_i8, "(iii)i", NULL},
    {"state_store_i8", (void*)native_state_store_i8, "(iii)", NULL},
    {"state_load_i16", (void*)native_state_load_i16, "(iii)i", NULL},
    {"state_store_i16", (void*)native_state_store_i16, "(iii)", NULL},
    {"state_load_i32", (void*)native_state_load_i32, "(iii)i", NULL},
    {"state_store_i32", (void*)native_state_store_i32, "(iii)", NULL},
    {"state_load_i64", (void*)native_state_load_i64, "(iiI)I", NULL},
    {"state_store_i64", (void*)native_state_store_i64, "(iiI)", NULL},
    {"state_load_f32", (void*)native_state_load_f32, "(iif)f", NULL},
    {"state_store_f32", (void*)native_state_store_f32, "(iif)", NULL},
    {"state_load_f64", (void*)native_state_load_f64, "(iiF)F", NULL},
    {"state_store_f64", (void*)native_state_store_f64, "(iiF)", NULL},
    {"state_load_str", (void*)native_state_load_str, "(iii)i", NULL},
    {"state_store_str", (void*)native_state_store_str, "(iii)", NULL},

    // Series operations (stubs)
    {"series_create_empty_u8", (void*)native_series_create_empty_u8, "(i)i", NULL},
    {"series_set_element_u8", (void*)native_series_set_element_u8, "(iii)", NULL},
    {"series_index_u8", (void*)native_series_index_u8, "(ii)i", NULL},
    {"series_element_add_u8", (void*)native_series_element_add_u8, "(ii)i", NULL},
    {"series_element_mul_u8", (void*)native_series_element_mul_u8, "(ii)i", NULL},
    {"series_element_sub_u8", (void*)native_series_element_sub_u8, "(ii)i", NULL},
    {"series_element_div_u8", (void*)native_series_element_div_u8, "(ii)i", NULL},
    {"series_series_add_u8", (void*)native_series_series_add_u8, "(ii)i", NULL},
    {"series_series_mul_u8", (void*)native_series_series_mul_u8, "(ii)i", NULL},
    {"series_series_sub_u8", (void*)native_series_series_sub_u8, "(ii)i", NULL},
    {"series_series_div_u8", (void*)native_series_series_div_u8, "(ii)i", NULL},
    {"series_compare_gt_u8", (void*)native_series_compare_gt_u8, "(ii)i", NULL},
    {"series_compare_lt_u8", (void*)native_series_compare_lt_u8, "(ii)i", NULL},
    {"series_compare_ge_u8", (void*)native_series_compare_ge_u8, "(ii)i", NULL},
    {"series_compare_le_u8", (void*)native_series_compare_le_u8, "(ii)i", NULL},
    {"series_compare_eq_u8", (void*)native_series_compare_eq_u8, "(ii)i", NULL},
    {"series_compare_ne_u8", (void*)native_series_compare_ne_u8, "(ii)i", NULL},

    {"series_create_empty_u16", (void*)native_series_create_empty_u16, "(i)i", NULL},
    {"series_set_element_u16", (void*)native_series_set_element_u16, "(iii)", NULL},
    {"series_index_u16", (void*)native_series_index_u16, "(ii)i", NULL},
    {"series_element_add_u16", (void*)native_series_element_add_u16, "(ii)i", NULL},
    {"series_element_mul_u16", (void*)native_series_element_mul_u16, "(ii)i", NULL},
    {"series_element_sub_u16", (void*)native_series_element_sub_u16, "(ii)i", NULL},
    {"series_element_div_u16", (void*)native_series_element_div_u16, "(ii)i", NULL},
    {"series_series_add_u16", (void*)native_series_series_add_u16, "(ii)i", NULL},
    {"series_series_mul_u16", (void*)native_series_series_mul_u16, "(ii)i", NULL},
    {"series_series_sub_u16", (void*)native_series_series_sub_u16, "(ii)i", NULL},
    {"series_series_div_u16", (void*)native_series_series_div_u16, "(ii)i", NULL},
    {"series_compare_gt_u16", (void*)native_series_compare_gt_u16, "(ii)i", NULL},
    {"series_compare_lt_u16", (void*)native_series_compare_lt_u16, "(ii)i", NULL},
    {"series_compare_ge_u16", (void*)native_series_compare_ge_u16, "(ii)i", NULL},
    {"series_compare_le_u16", (void*)native_series_compare_le_u16, "(ii)i", NULL},
    {"series_compare_eq_u16", (void*)native_series_compare_eq_u16, "(ii)i", NULL},
    {"series_compare_ne_u16", (void*)native_series_compare_ne_u16, "(ii)i", NULL},

    {"series_create_empty_u32", (void*)native_series_create_empty_u32, "(i)i", NULL},
    {"series_set_element_u32", (void*)native_series_set_element_u32, "(iii)", NULL},
    {"series_index_u32", (void*)native_series_index_u32, "(ii)i", NULL},
    {"series_element_add_u32", (void*)native_series_element_add_u32, "(ii)i", NULL},
    {"series_element_mul_u32", (void*)native_series_element_mul_u32, "(ii)i", NULL},
    {"series_element_sub_u32", (void*)native_series_element_sub_u32, "(ii)i", NULL},
    {"series_element_div_u32", (void*)native_series_element_div_u32, "(ii)i", NULL},
    {"series_series_add_u32", (void*)native_series_series_add_u32, "(ii)i", NULL},
    {"series_series_mul_u32", (void*)native_series_series_mul_u32, "(ii)i", NULL},
    {"series_series_sub_u32", (void*)native_series_series_sub_u32, "(ii)i", NULL},
    {"series_series_div_u32", (void*)native_series_series_div_u32, "(ii)i", NULL},
    {"series_compare_gt_u32", (void*)native_series_compare_gt_u32, "(ii)i", NULL},
    {"series_compare_lt_u32", (void*)native_series_compare_lt_u32, "(ii)i", NULL},
    {"series_compare_ge_u32", (void*)native_series_compare_ge_u32, "(ii)i", NULL},
    {"series_compare_le_u32", (void*)native_series_compare_le_u32, "(ii)i", NULL},
    {"series_compare_eq_u32", (void*)native_series_compare_eq_u32, "(ii)i", NULL},
    {"series_compare_ne_u32", (void*)native_series_compare_ne_u32, "(ii)i", NULL},

    {"series_create_empty_u64", (void*)native_series_create_empty_u64, "(i)i", NULL},
    {"series_set_element_u64", (void*)native_series_set_element_u64, "(iiI)", NULL},
    {"series_index_u64", (void*)native_series_index_u64, "(ii)I", NULL},
    {"series_element_add_u64", (void*)native_series_element_add_u64, "(iI)i", NULL},
    {"series_element_mul_u64", (void*)native_series_element_mul_u64, "(iI)i", NULL},
    {"series_element_sub_u64", (void*)native_series_element_sub_u64, "(iI)i", NULL},
    {"series_element_div_u64", (void*)native_series_element_div_u64, "(iI)i", NULL},
    {"series_series_add_u64", (void*)native_series_series_add_u64, "(ii)i", NULL},
    {"series_series_mul_u64", (void*)native_series_series_mul_u64, "(ii)i", NULL},
    {"series_series_sub_u64", (void*)native_series_series_sub_u64, "(ii)i", NULL},
    {"series_series_div_u64", (void*)native_series_series_div_u64, "(ii)i", NULL},
    {"series_compare_gt_u64", (void*)native_series_compare_gt_u64, "(ii)i", NULL},
    {"series_compare_lt_u64", (void*)native_series_compare_lt_u64, "(ii)i", NULL},
    {"series_compare_ge_u64", (void*)native_series_compare_ge_u64, "(ii)i", NULL},
    {"series_compare_le_u64", (void*)native_series_compare_le_u64, "(ii)i", NULL},
    {"series_compare_eq_u64", (void*)native_series_compare_eq_u64, "(ii)i", NULL},
    {"series_compare_ne_u64", (void*)native_series_compare_ne_u64, "(ii)i", NULL},

    {"series_create_empty_i8", (void*)native_series_create_empty_i8, "(i)i", NULL},
    {"series_set_element_i8", (void*)native_series_set_element_i8, "(iii)", NULL},
    {"series_index_i8", (void*)native_series_index_i8, "(ii)i", NULL},
    {"series_element_add_i8", (void*)native_series_element_add_i8, "(ii)i", NULL},
    {"series_element_mul_i8", (void*)native_series_element_mul_i8, "(ii)i", NULL},
    {"series_element_sub_i8", (void*)native_series_element_sub_i8, "(ii)i", NULL},
    {"series_element_div_i8", (void*)native_series_element_div_i8, "(ii)i", NULL},
    {"series_series_add_i8", (void*)native_series_series_add_i8, "(ii)i", NULL},
    {"series_series_mul_i8", (void*)native_series_series_mul_i8, "(ii)i", NULL},
    {"series_series_sub_i8", (void*)native_series_series_sub_i8, "(ii)i", NULL},
    {"series_series_div_i8", (void*)native_series_series_div_i8, "(ii)i", NULL},
    {"series_compare_gt_i8", (void*)native_series_compare_gt_i8, "(ii)i", NULL},
    {"series_compare_lt_i8", (void*)native_series_compare_lt_i8, "(ii)i", NULL},
    {"series_compare_ge_i8", (void*)native_series_compare_ge_i8, "(ii)i", NULL},
    {"series_compare_le_i8", (void*)native_series_compare_le_i8, "(ii)i", NULL},
    {"series_compare_eq_i8", (void*)native_series_compare_eq_i8, "(ii)i", NULL},
    {"series_compare_ne_i8", (void*)native_series_compare_ne_i8, "(ii)i", NULL},

    {"series_create_empty_i16", (void*)native_series_create_empty_i16, "(i)i", NULL},
    {"series_set_element_i16", (void*)native_series_set_element_i16, "(iii)", NULL},
    {"series_index_i16", (void*)native_series_index_i16, "(ii)i", NULL},
    {"series_element_add_i16", (void*)native_series_element_add_i16, "(ii)i", NULL},
    {"series_element_mul_i16", (void*)native_series_element_mul_i16, "(ii)i", NULL},
    {"series_element_sub_i16", (void*)native_series_element_sub_i16, "(ii)i", NULL},
    {"series_element_div_i16", (void*)native_series_element_div_i16, "(ii)i", NULL},
    {"series_series_add_i16", (void*)native_series_series_add_i16, "(ii)i", NULL},
    {"series_series_mul_i16", (void*)native_series_series_mul_i16, "(ii)i", NULL},
    {"series_series_sub_i16", (void*)native_series_series_sub_i16, "(ii)i", NULL},
    {"series_series_div_i16", (void*)native_series_series_div_i16, "(ii)i", NULL},
    {"series_compare_gt_i16", (void*)native_series_compare_gt_i16, "(ii)i", NULL},
    {"series_compare_lt_i16", (void*)native_series_compare_lt_i16, "(ii)i", NULL},
    {"series_compare_ge_i16", (void*)native_series_compare_ge_i16, "(ii)i", NULL},
    {"series_compare_le_i16", (void*)native_series_compare_le_i16, "(ii)i", NULL},
    {"series_compare_eq_i16", (void*)native_series_compare_eq_i16, "(ii)i", NULL},
    {"series_compare_ne_i16", (void*)native_series_compare_ne_i16, "(ii)i", NULL},

    {"series_create_empty_i32", (void*)native_series_create_empty_i32, "(i)i", NULL},
    {"series_set_element_i32", (void*)native_series_set_element_i32, "(iii)", NULL},
    {"series_index_i32", (void*)native_series_index_i32, "(ii)i", NULL},
    {"series_element_add_i32", (void*)native_series_element_add_i32, "(ii)i", NULL},
    {"series_element_mul_i32", (void*)native_series_element_mul_i32, "(ii)i", NULL},
    {"series_element_sub_i32", (void*)native_series_element_sub_i32, "(ii)i", NULL},
    {"series_element_div_i32", (void*)native_series_element_div_i32, "(ii)i", NULL},
    {"series_series_add_i32", (void*)native_series_series_add_i32, "(ii)i", NULL},
    {"series_series_mul_i32", (void*)native_series_series_mul_i32, "(ii)i", NULL},
    {"series_series_sub_i32", (void*)native_series_series_sub_i32, "(ii)i", NULL},
    {"series_series_div_i32", (void*)native_series_series_div_i32, "(ii)i", NULL},
    {"series_compare_gt_i32", (void*)native_series_compare_gt_i32, "(ii)i", NULL},
    {"series_compare_lt_i32", (void*)native_series_compare_lt_i32, "(ii)i", NULL},
    {"series_compare_ge_i32", (void*)native_series_compare_ge_i32, "(ii)i", NULL},
    {"series_compare_le_i32", (void*)native_series_compare_le_i32, "(ii)i", NULL},
    {"series_compare_eq_i32", (void*)native_series_compare_eq_i32, "(ii)i", NULL},
    {"series_compare_ne_i32", (void*)native_series_compare_ne_i32, "(ii)i", NULL},

    {"series_create_empty_i64", (void*)native_series_create_empty_i64, "(i)i", NULL},
    {"series_set_element_i64", (void*)native_series_set_element_i64, "(iiI)", NULL},
    {"series_index_i64", (void*)native_series_index_i64, "(ii)I", NULL},
    {"series_element_add_i64", (void*)native_series_element_add_i64, "(iI)i", NULL},
    {"series_element_mul_i64", (void*)native_series_element_mul_i64, "(iI)i", NULL},
    {"series_element_sub_i64", (void*)native_series_element_sub_i64, "(iI)i", NULL},
    {"series_element_div_i64", (void*)native_series_element_div_i64, "(iI)i", NULL},
    {"series_series_add_i64", (void*)native_series_series_add_i64, "(ii)i", NULL},
    {"series_series_mul_i64", (void*)native_series_series_mul_i64, "(ii)i", NULL},
    {"series_series_sub_i64", (void*)native_series_series_sub_i64, "(ii)i", NULL},
    {"series_series_div_i64", (void*)native_series_series_div_i64, "(ii)i", NULL},
    {"series_compare_gt_i64", (void*)native_series_compare_gt_i64, "(ii)i", NULL},
    {"series_compare_lt_i64", (void*)native_series_compare_lt_i64, "(ii)i", NULL},
    {"series_compare_ge_i64", (void*)native_series_compare_ge_i64, "(ii)i", NULL},
    {"series_compare_le_i64", (void*)native_series_compare_le_i64, "(ii)i", NULL},
    {"series_compare_eq_i64", (void*)native_series_compare_eq_i64, "(ii)i", NULL},
    {"series_compare_ne_i64", (void*)native_series_compare_ne_i64, "(ii)i", NULL},

    {"series_create_empty_f32", (void*)native_series_create_empty_f32, "(i)i", NULL},
    {"series_set_element_f32", (void*)native_series_set_element_f32, "(iif)", NULL},
    {"series_index_f32", (void*)native_series_index_f32, "(ii)f", NULL},
    {"series_element_add_f32", (void*)native_series_element_add_f32, "(if)i", NULL},
    {"series_element_mul_f32", (void*)native_series_element_mul_f32, "(if)i", NULL},
    {"series_element_sub_f32", (void*)native_series_element_sub_f32, "(if)i", NULL},
    {"series_element_div_f32", (void*)native_series_element_div_f32, "(if)i", NULL},
    {"series_series_add_f32", (void*)native_series_series_add_f32, "(ii)i", NULL},
    {"series_series_mul_f32", (void*)native_series_series_mul_f32, "(ii)i", NULL},
    {"series_series_sub_f32", (void*)native_series_series_sub_f32, "(ii)i", NULL},
    {"series_series_div_f32", (void*)native_series_series_div_f32, "(ii)i", NULL},
    {"series_compare_gt_f32", (void*)native_series_compare_gt_f32, "(ii)i", NULL},
    {"series_compare_lt_f32", (void*)native_series_compare_lt_f32, "(ii)i", NULL},
    {"series_compare_ge_f32", (void*)native_series_compare_ge_f32, "(ii)i", NULL},
    {"series_compare_le_f32", (void*)native_series_compare_le_f32, "(ii)i", NULL},
    {"series_compare_eq_f32", (void*)native_series_compare_eq_f32, "(ii)i", NULL},
    {"series_compare_ne_f32", (void*)native_series_compare_ne_f32, "(ii)i", NULL},

    {"series_create_empty_f64", (void*)native_series_create_empty_f64, "(i)i", NULL},
    {"series_set_element_f64", (void*)native_series_set_element_f64, "(iiF)", NULL},
    {"series_index_f64", (void*)native_series_index_f64, "(ii)F", NULL},
    {"series_element_add_f64", (void*)native_series_element_add_f64, "(iF)i", NULL},
    {"series_element_mul_f64", (void*)native_series_element_mul_f64, "(iF)i", NULL},
    {"series_element_sub_f64", (void*)native_series_element_sub_f64, "(iF)i", NULL},
    {"series_element_div_f64", (void*)native_series_element_div_f64, "(iF)i", NULL},
    {"series_series_add_f64", (void*)native_series_series_add_f64, "(ii)i", NULL},
    {"series_series_mul_f64", (void*)native_series_series_mul_f64, "(ii)i", NULL},
    {"series_series_sub_f64", (void*)native_series_series_sub_f64, "(ii)i", NULL},
    {"series_series_div_f64", (void*)native_series_series_div_f64, "(ii)i", NULL},
    {"series_compare_gt_f64", (void*)native_series_compare_gt_f64, "(ii)i", NULL},
    {"series_compare_lt_f64", (void*)native_series_compare_lt_f64, "(ii)i", NULL},
    {"series_compare_ge_f64", (void*)native_series_compare_ge_f64, "(ii)i", NULL},
    {"series_compare_le_f64", (void*)native_series_compare_le_f64, "(ii)i", NULL},
    {"series_compare_eq_f64", (void*)native_series_compare_eq_f64, "(ii)i", NULL},
    {"series_compare_ne_f64", (void*)native_series_compare_ne_f64, "(ii)i", NULL},

    {"series_len", (void*)native_series_len, "(i)I", NULL},
    {"series_slice", (void*)native_series_slice, "(iii)i", NULL},

    // String operations
    {"string_from_literal", (void*)native_string_from_literal, "(ii)i", NULL},
    {"string_len", (void*)native_string_len, "(i)i", NULL},
    {"string_equal", (void*)native_string_equal, "(ii)i", NULL},

    // Generic operations
    {"now", (void*)native_now, "()I", NULL},
    {"len", (void*)native_len, "(i)I", NULL},
    {"panic", (void*)native_panic, "(ii)", NULL},

    // Math operations
    {"math_pow_f32", (void*)native_math_pow_f32, "(ff)f", NULL},
    {"math_pow_f64", (void*)native_math_pow_f64, "(FF)F", NULL},
    {"math_int_pow_u8", (void*)native_math_int_pow_u8, "(ii)i", NULL},
    {"math_int_pow_u16", (void*)native_math_int_pow_u16, "(ii)i", NULL},
    {"math_int_pow_u32", (void*)native_math_int_pow_u32, "(ii)i", NULL},
    {"math_int_pow_u64", (void*)native_math_int_pow_u64, "(II)I", NULL},
    {"math_int_pow_i8", (void*)native_math_int_pow_i8, "(ii)i", NULL},
    {"math_int_pow_i16", (void*)native_math_int_pow_i16, "(ii)i", NULL},
    {"math_int_pow_i32", (void*)native_math_int_pow_i32, "(ii)i", NULL},
    {"math_int_pow_i64", (void*)native_math_int_pow_i64, "(II)I", NULL},
};
*/

// ===== Wasmtime Host Function Implementation =====

// Helper: Get Runtime from store data using Wasmtime-specific API
Runtime *get_runtime_from_store(wasmtime_store_t *store) {
    wasmtime_context_t *context = wasmtime_store_context(store);
    void *data = wasmtime_context_get_data(context);
    return static_cast<Runtime*>(data);
}

// Helper: Set Runtime in store data using Wasmtime-specific API
void set_runtime_in_store(wasmtime_store_t *store, Runtime *runtime) {
    wasmtime_context_t *context = wasmtime_store_context(store);
    wasmtime_context_set_data(context, runtime);
    if (runtime) {
        runtime->set_store(store);
    }
}

// ===== Wasmtime Host Function Callbacks =====
// Pattern: Each callback extracts Runtime from env (which is the store)

// Example: state_load_u32
static wasm_trap_t* host_state_load_u32(
    void *env,
    const wasm_val_vec_t *args,
    wasm_val_vec_t *results
) {
    auto *store = static_cast<wasmtime_store_t*>(env);
    Runtime *runtime = get_runtime_from_store(store);
    if (!runtime) {
        return wasmtime_trap_new("Runtime not found in store", 23);
    }

    // Extract args: func_id, var_id, init_value
    uint32_t func_id = args->data[0].of.i32;
    uint32_t var_id = args->data[1].of.i32;
    uint32_t init_value = args->data[2].of.i32;

    // Call runtime method
    uint32_t result = runtime->state_load_u32(func_id, var_id, init_value);

    // Set result
    results->data[0].kind = WASM_I32;
    results->data[0].of.i32 = result;

    return nullptr;  // No trap
}

// Example: state_store_u32
static wasm_trap_t* host_state_store_u32(
    void *env,
    const wasm_val_vec_t *args,
    wasm_val_vec_t *results
) {
    auto *store = static_cast<wasmtime_store_t*>(env);
    Runtime *runtime = get_runtime_from_store(store);
    if (!runtime) {
        return wasmtime_trap_new("Runtime not found in store", 23);
    }

    uint32_t func_id = args->data[0].of.i32;
    uint32_t var_id = args->data[1].of.i32;
    uint32_t value = args->data[2].of.i32;

    runtime->state_store_u32(func_id, var_id, value);

    return nullptr;
}

// Example: math_pow_f32
static wasm_trap_t* host_math_pow_f32(
    void *env,
    const wasm_val_vec_t *args,
    wasm_val_vec_t *results
) {
    auto *store = static_cast<wasmtime_store_t*>(env);
    Runtime *runtime = get_runtime_from_store(store);
    if (!runtime) {
        return wasmtime_trap_new("Runtime not found in store", 23);
    }

    float base = args->data[0].of.f32;
    float exp = args->data[1].of.f32;
    float result = runtime->math_pow_f32(base, exp);

    results->data[0].kind = WASM_F32;
    results->data[0].of.f32 = result;

    return nullptr;
}

// Example: now (returns current timestamp)
static wasm_trap_t* host_now(
    void *env,
    const wasm_val_vec_t *args,
    wasm_val_vec_t *results
) {
    auto *store = static_cast<wasmtime_store_t*>(env);
    Runtime *runtime = get_runtime_from_store(store);
    if (!runtime) {
        return wasmtime_trap_new("Runtime not found in store", 23);
    }

    uint64_t now = runtime->now();

    results->data[0].kind = WASM_I64;
    results->data[0].of.i64 = now;

    return nullptr;
}

// ===== Import Creation =====

wasm_extern_vec_t create_imports(wasmtime_store_t *store, Runtime *runtime) {
    std::vector<wasm_extern_t*> imports;

    // Helper lambda to create function type
    auto make_func_type = [](const std::vector<wasm_valkind_t> &params,
                              const std::vector<wasm_valkind_t> &results) -> wasm_functype_t* {
        wasm_valtype_vec_t param_types;
        wasm_valtype_vec_new_uninitialized(&param_types, params.size());
        for (size_t i = 0; i < params.size(); i++) {
            param_types.data[i] = wasm_valtype_new(params[i]);
        }

        wasm_valtype_vec_t result_types;
        wasm_valtype_vec_new_uninitialized(&result_types, results.size());
        for (size_t i = 0; i < results.size(); i++) {
            result_types.data[i] = wasm_valtype_new(results[i]);
        }

        return wasm_functype_new(&param_types, &result_types);
    };

    // Cast wasmtime_store_t to wasm_store_t for standard API compatibility
    wasm_store_t *wasm_store = reinterpret_cast<wasm_store_t*>(store);

    // Create function: state_load_u32(i32, i32, i32) -> i32
    wasm_functype_t *state_load_u32_type = make_func_type(
        {WASM_I32, WASM_I32, WASM_I32},  // func_id, var_id, init_value
        {WASM_I32}                        // return value
    );
    wasm_func_t *state_load_u32_func = wasm_func_new_with_env(
        wasm_store,
        state_load_u32_type,
        host_state_load_u32,
        store,  // Pass wasmtime_store as env for callbacks
        nullptr  // No finalizer
    );
    wasm_functype_delete(state_load_u32_type);
    imports.push_back(wasm_func_as_extern(state_load_u32_func));

    // Create function: state_store_u32(i32, i32, i32) -> void
    wasm_functype_t *state_store_u32_type = make_func_type(
        {WASM_I32, WASM_I32, WASM_I32},
        {}  // No return
    );
    wasm_func_t *state_store_u32_func = wasm_func_new_with_env(
        wasm_store,
        state_store_u32_type,
        host_state_store_u32,
        store,
        nullptr
    );
    wasm_functype_delete(state_store_u32_type);
    imports.push_back(wasm_func_as_extern(state_store_u32_func));

    // Create function: math_pow_f32(f32, f32) -> f32
    wasm_functype_t *math_pow_f32_type = make_func_type(
        {WASM_F32, WASM_F32},
        {WASM_F32}
    );
    wasm_func_t *math_pow_f32_func = wasm_func_new_with_env(
        wasm_store,
        math_pow_f32_type,
        host_math_pow_f32,
        store,
        nullptr
    );
    wasm_functype_delete(math_pow_f32_type);
    imports.push_back(wasm_func_as_extern(math_pow_f32_func));

    // Create function: now() -> i64
    wasm_functype_t *now_type = make_func_type({}, {WASM_I64});
    wasm_func_t *now_func = wasm_func_new_with_env(
        wasm_store,
        now_type,
        host_now,
        store,
        nullptr
    );
    wasm_functype_delete(now_type);
    imports.push_back(wasm_func_as_extern(now_func));

    // TODO: Add remaining 266+ host functions following this pattern

    // Convert to wasm_extern_vec_t
    wasm_extern_vec_t import_vec;
    wasm_extern_vec_new_uninitialized(&import_vec, imports.size());
    for (size_t i = 0; i < imports.size(); i++) {
        import_vec.data[i] = imports[i];
    }

    std::printf("Created %zu host function imports\n", imports.size());
    return import_vec;
}

void delete_imports(wasm_extern_vec_t *imports) {
    // Wasmtime manages memory - don't delete individual funcs
    // Just delete the vector itself
    wasm_extern_vec_delete(imports);
}

}
