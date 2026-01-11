// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>
#include <string>
#include <unordered_map>
#include <vector>

#include "x/cpp/telem/series.h"

#include "arc/cpp/runtime/state/state.h"
#include "wasmtime.hh"

namespace arc::runtime::wasm {

/// Runtime provides the actual implementation of Arc runtime functions.
/// This is the "business logic" layer that the bindings call.
class Bindings {
    [[maybe_unused]] state::State *state;
    wasmtime::Store *store;
    wasmtime::Memory *memory;

    std::unordered_map<uint32_t, std::string> strings;
    uint32_t string_handle_counter;
    std::unordered_map<uint32_t, x::telem::Series> series;
    uint32_t series_handle_counter;

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
    std::unordered_map<uint64_t, x::telem::Series> state_series;

    static uint64_t state_key(const uint32_t func_id, const uint32_t var_id) {
        return static_cast<uint64_t>(func_id) << 32 | static_cast<uint64_t>(var_id);
    }

public:
    Bindings(state::State *state, wasmtime::Store *store);

/// Channel operations use semantic C++ types. The MethodWrapper in bindings.cpp
/// automatically converts to WASM-compatible types (i32, i64, f32, f64) at the
/// binding layer using the WasmType trait.
#define DECLARE_CHANNEL_OPS(suffix, cpptype)                                           \
    cpptype channel_read_##suffix(uint32_t channel_id);                                \
    void channel_write_##suffix(uint32_t channel_id, cpptype value);

    DECLARE_CHANNEL_OPS(u8, uint8_t)
    DECLARE_CHANNEL_OPS(u16, uint16_t)
    DECLARE_CHANNEL_OPS(u32, uint32_t)
    DECLARE_CHANNEL_OPS(u64, uint64_t)
    DECLARE_CHANNEL_OPS(i8, int8_t)
    DECLARE_CHANNEL_OPS(i16, int16_t)
    DECLARE_CHANNEL_OPS(i32, int32_t)
    DECLARE_CHANNEL_OPS(i64, int64_t)
    DECLARE_CHANNEL_OPS(f32, float)
    DECLARE_CHANNEL_OPS(f64, double)

#undef DECLARE_CHANNEL_OPS
    uint32_t channel_read_str(uint32_t channel_id);
    void channel_write_str(uint32_t channel_id, uint32_t str_handle);

#define DECLARE_STATE_OPS(suffix, cpptype)                                             \
    cpptype state_load_##suffix(                                                       \
        uint32_t func_id,                                                              \
        uint32_t var_id,                                                               \
        cpptype init_value                                                             \
    );                                                                                 \
    void state_store_##suffix(uint32_t func_id, uint32_t var_id, cpptype value);

    DECLARE_STATE_OPS(u8, uint8_t)
    DECLARE_STATE_OPS(u16, uint16_t)
    DECLARE_STATE_OPS(u32, uint32_t)
    DECLARE_STATE_OPS(u64, uint64_t)
    DECLARE_STATE_OPS(i8, int8_t)
    DECLARE_STATE_OPS(i16, int16_t)
    DECLARE_STATE_OPS(i32, int32_t)
    DECLARE_STATE_OPS(i64, int64_t)
    DECLARE_STATE_OPS(f32, float)
    DECLARE_STATE_OPS(f64, double)

#undef DECLARE_STATE_OPS

    uint32_t state_load_str(uint32_t func_id, uint32_t var_id, uint32_t init_handle);
    void state_store_str(uint32_t func_id, uint32_t var_id, uint32_t str_handle);

// Series element operations (per type) - using macro with proper C++ types
#define DECLARE_SERIES_OPS(suffix, cpptype)                                            \
    uint32_t series_create_empty_##suffix(uint32_t length);                            \
    uint32_t series_set_element_##suffix(                                              \
        uint32_t handle,                                                               \
        uint32_t index,                                                                \
        cpptype value                                                                  \
    );                                                                                 \
    cpptype series_index_##suffix(uint32_t handle, uint32_t index);                    \
    uint32_t series_element_add_##suffix(uint32_t handle, cpptype value);              \
    uint32_t series_element_mul_##suffix(uint32_t handle, cpptype value);              \
    uint32_t series_element_sub_##suffix(uint32_t handle, cpptype value);              \
    uint32_t series_element_div_##suffix(uint32_t handle, cpptype value);              \
    uint32_t series_element_rsub_##suffix(cpptype value, uint32_t handle);             \
    uint32_t series_element_rdiv_##suffix(cpptype value, uint32_t handle);             \
    uint32_t series_element_mod_##suffix(uint32_t handle, cpptype value);              \
    uint32_t series_series_add_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_series_mul_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_series_sub_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_series_div_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_series_mod_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_compare_gt_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_compare_lt_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_compare_ge_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_compare_le_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_compare_eq_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_compare_ne_##suffix(uint32_t a, uint32_t b);                       \
    uint32_t series_compare_gt_scalar_##suffix(uint32_t handle, cpptype value);        \
    uint32_t series_compare_lt_scalar_##suffix(uint32_t handle, cpptype value);        \
    uint32_t series_compare_ge_scalar_##suffix(uint32_t handle, cpptype value);        \
    uint32_t series_compare_le_scalar_##suffix(uint32_t handle, cpptype value);        \
    uint32_t series_compare_eq_scalar_##suffix(uint32_t handle, cpptype value);        \
    uint32_t series_compare_ne_scalar_##suffix(uint32_t handle, cpptype value);        \
    uint32_t state_load_series_##suffix(                                               \
        uint32_t func_id,                                                              \
        uint32_t var_id,                                                               \
        uint32_t init_handle                                                           \
    );                                                                                 \
    void state_store_series_##suffix(                                                  \
        uint32_t func_id,                                                              \
        uint32_t var_id,                                                               \
        uint32_t handle                                                                \
    );

    DECLARE_SERIES_OPS(u8, uint8_t)
    DECLARE_SERIES_OPS(u16, uint16_t)
    DECLARE_SERIES_OPS(u32, uint32_t)
    DECLARE_SERIES_OPS(u64, uint64_t)
    DECLARE_SERIES_OPS(i8, int8_t)
    DECLARE_SERIES_OPS(i16, int16_t)
    DECLARE_SERIES_OPS(i32, int32_t)
    DECLARE_SERIES_OPS(i64, int64_t)
    DECLARE_SERIES_OPS(f32, float)
    DECLARE_SERIES_OPS(f64, double)

#undef DECLARE_SERIES_OPS

#define DECLARE_SERIES_NEGATE(suffix) uint32_t series_negate_##suffix(uint32_t handle);
    DECLARE_SERIES_NEGATE(i8)
    DECLARE_SERIES_NEGATE(i16)
    DECLARE_SERIES_NEGATE(i32)
    DECLARE_SERIES_NEGATE(i64)
    DECLARE_SERIES_NEGATE(f32)
    DECLARE_SERIES_NEGATE(f64)
#undef DECLARE_SERIES_NEGATE

    uint32_t series_not_u8(uint32_t handle);

    void set_memory(wasmtime::Memory *mem) { this->memory = mem; }

    static uint64_t now();
    uint64_t len(uint32_t handle);
    void panic(uint32_t ptr, uint32_t len);
#define DECLARE_MATH_POW_OP(suffix, cpptype)                                           \
    cpptype math_pow_##suffix(cpptype base, cpptype exp);
    DECLARE_MATH_POW_OP(u8, uint8_t)
    DECLARE_MATH_POW_OP(u16, uint16_t)
    DECLARE_MATH_POW_OP(u32, uint32_t)
    DECLARE_MATH_POW_OP(u64, uint64_t)
    DECLARE_MATH_POW_OP(i8, int8_t)
    DECLARE_MATH_POW_OP(i16, int16_t)
    DECLARE_MATH_POW_OP(i32, int32_t)
    DECLARE_MATH_POW_OP(i64, int64_t)
    DECLARE_MATH_POW_OP(f32, float)
    DECLARE_MATH_POW_OP(f64, double)
#undef DECLARE_MATH_POW_OP

    uint64_t series_len(uint32_t handle);
    uint32_t series_slice(uint32_t handle, uint32_t start, uint32_t end);

    uint32_t string_from_literal(uint32_t ptr, uint32_t len);
    uint32_t string_concat(uint32_t handle1, uint32_t handle2);
    uint32_t string_equal(uint32_t handle1, uint32_t handle2);
    uint32_t string_len(uint32_t handle);

    /// @brief Creates a string handle from a C++ string (for testing)
    uint32_t string_create(const std::string &str);
    /// @brief Gets the string value for a handle (for testing)
    std::string string_get(uint32_t handle);

    /// @brief Clears transient string and series handles at the end of each execution
    /// cycle. This resets the handle counters and clears the temporary storage maps.
    /// Stateful variables (state_* maps) are NOT cleared as they must persist across
    /// cycles.
    void clear_transient_handles();
};

/// @brief create import vector with all registered host functions for Wasmtime.
/// Must be called before instance creation.
/// Returns vector of Extern objects that should be passed to Instance::create().
std::vector<wasmtime::Extern> create_imports(wasmtime::Store &store, Bindings *runtime);

}
