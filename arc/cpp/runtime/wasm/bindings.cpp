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

namespace arc::runtime::wasm::bindings {

// Module-local storage for Runtime instance per module
static std::unordered_map<wasm_module_inst_t, Runtime *> runtime_map;

Runtime::Runtime(state::State *state, wasm_module_inst_t module_inst)
    : state(state),
      module_inst(module_inst),
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
    if (module_inst == nullptr) return 0;

    void *native_ptr = wasm_runtime_addr_app_to_native(module_inst, ptr);
    if (native_ptr == nullptr) return 0;

    std::string str(static_cast<const char *>(native_ptr), len);
    uint32_t handle = string_handle_counter++;
    strings[handle] = str;
    return handle;
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
    if (module_inst == nullptr) {
        std::fprintf(stderr, "WASM panic: (unable to read message)\n");
        std::abort();
    }

    void *native_ptr = wasm_runtime_addr_app_to_native(module_inst, ptr);
    if (native_ptr == nullptr) {
        std::fprintf(stderr, "WASM panic: (invalid pointer)\n");
        std::abort();
    }

    std::string msg(static_cast<const char *>(native_ptr), len);
    std::fprintf(stderr, "WASM panic: %s\n", msg.c_str());
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

// ===== WAMR Native Function Wrappers =====
// These are the actual C functions that WAMR will call

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

void register_natives(Runtime *runtime) {
    uint32_t count = sizeof(native_symbols) / sizeof(NativeSymbol);
    std::printf("Registering %u native functions\n", count);

    bool result = wasm_runtime_register_natives(
        "env",
        native_symbols,
        count
    );

    std::printf("Registration result: %s\n", result ? "SUCCESS" : "FAILED");
}

Runtime *get_runtime(wasm_exec_env_t exec_env) {
    auto module_inst = wasm_runtime_get_module_inst(exec_env);
    auto it = runtime_map.find(module_inst);
    if (it == runtime_map.end()) return nullptr;
    return it->second;
}

void set_runtime(wasm_module_inst_t module_inst, Runtime *runtime) {
    runtime_map[module_inst] = runtime;
    if (runtime) {
        runtime->set_module_inst(module_inst);
    }
}

}
