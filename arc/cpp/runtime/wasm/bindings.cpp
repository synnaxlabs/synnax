// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <chrono>
#include <cmath>
#include <cstdio>
#include <cstring>

#include "bindings.h"
#include "wasmtime.hh" // For Wasmtime C++ API

namespace arc::runtime::wasm {

Bindings::Bindings(state::State *state, wasmtime::Store *store):
    state(state), store(store), memory(nullptr), string_handle_counter(1) {}

// ===== Channel Operations (Stubs) =====

#define IMPL_CHANNEL_OPS(suffix, cpptype, default_val)                                 \
    cpptype Bindings::channel_read_##suffix(uint32_t channel_id) {                     \
        return default_val;                                                            \
    }                                                                                  \
    void Bindings::channel_write_##suffix(uint32_t channel_id, cpptype value) {}       \
    cpptype Bindings::channel_blocking_read_##suffix(uint32_t channel_id) {            \
        return default_val;                                                            \
    }

IMPL_CHANNEL_OPS(u8, uint8_t, 0)
IMPL_CHANNEL_OPS(u16, uint16_t, 0)
IMPL_CHANNEL_OPS(u32, uint32_t, 0)
IMPL_CHANNEL_OPS(u64, uint64_t, 0)
IMPL_CHANNEL_OPS(i8, int8_t, 0)
IMPL_CHANNEL_OPS(i16, int16_t, 0)
IMPL_CHANNEL_OPS(i32, int32_t, 0)
IMPL_CHANNEL_OPS(i64, int64_t, 0)
IMPL_CHANNEL_OPS(f32, float, 0.0f)
IMPL_CHANNEL_OPS(f64, double, 0.0)

#undef IMPL_CHANNEL_OPS

uint32_t Bindings::channel_read_str(uint32_t channel_id) {
    return 0;
}
void Bindings::channel_write_str(uint32_t channel_id, uint32_t str_handle) {}
uint32_t Bindings::channel_blocking_read_str(uint32_t channel_id) {
    return 0;
}

// ===== State Operations =====

#define IMPL_STATE_OPS(suffix, cpptype)                                                \
    cpptype Bindings::state_load_##suffix(                                             \
        uint32_t func_id,                                                              \
        uint32_t var_id,                                                               \
        cpptype init_value                                                             \
    ) {                                                                                \
        auto key = state_key(func_id, var_id);                                         \
        auto it = state_##suffix.find(key);                                            \
        if (it != state_##suffix.end()) return it->second;                             \
        state_##suffix[key] = init_value;                                              \
        return init_value;                                                             \
    }                                                                                  \
    void Bindings::state_store_##suffix(                                               \
        uint32_t func_id,                                                              \
        uint32_t var_id,                                                               \
        cpptype value                                                                  \
    ) {                                                                                \
        state_##suffix[state_key(func_id, var_id)] = value;                            \
    }

IMPL_STATE_OPS(u8, uint8_t)
IMPL_STATE_OPS(u16, uint16_t)
IMPL_STATE_OPS(u32, uint32_t)
IMPL_STATE_OPS(u64, uint64_t)
IMPL_STATE_OPS(i8, int8_t)
IMPL_STATE_OPS(i16, int16_t)
IMPL_STATE_OPS(i32, int32_t)
IMPL_STATE_OPS(i64, int64_t)
IMPL_STATE_OPS(f32, float)
IMPL_STATE_OPS(f64, double)

#undef IMPL_STATE_OPS

uint32_t Bindings::state_load_str(
    const uint32_t func_id,
    const uint32_t var_id,
    const uint32_t init_handle
) {
    const auto key = state_key(func_id, var_id);
    if (const auto it = this->state_string.find(key); it != this->state_string.end()) {
        this->strings[string_handle_counter] = it->second;
        return this->string_handle_counter++;
    }
    if (const auto init_it = this->strings.find(init_handle); init_it != strings.end())
        this->state_string[key] = init_it->second;
    else
        this->state_string[key] = "";
    this->strings[this->string_handle_counter] = this->state_string[key];
    return this->string_handle_counter++;
}

auto Bindings::state_store_str(
    const uint32_t func_id,
    const uint32_t var_id,
    const uint32_t str_handle
) -> void {
    if (const auto it = strings.find(str_handle); it != strings.end())
        state_string[state_key(func_id, var_id)] = it->second;
}

// ===== String Operations =====

uint32_t Bindings::string_from_literal(const uint32_t ptr, const uint32_t len) {
    if (!memory || !store) {
        std::fprintf(
            stderr,
            "ERROR: string_from_literal called but no memory or store available\n"
        );
        return 0;
    }

    const auto mem_span = memory->data(*store);
    const uint8_t *mem_data = mem_span.data();

    // Bounds check
    if (const size_t mem_size = mem_span.size(); ptr + len > mem_size) {
        std::fprintf(
            stderr,
            "ERROR: string_from_literal ptr=%u len=%u exceeds memory size=%zu\n",
            ptr,
            len,
            mem_size
        );
        return 0;
    }

    const std::string str(reinterpret_cast<const char *>(mem_data + ptr), len);
    const uint32_t handle = string_handle_counter++;
    strings[handle] = str;
    return handle;
}
uint32_t Bindings::string_concat(uint32_t ptr, uint32_t len) {
    return 0; // Not implemented
}

uint32_t Bindings::string_len(const uint32_t handle) {
    const auto it = strings.find(handle);
    if (it == strings.end()) return 0;
    return static_cast<uint32_t>(it->second.length());
}

uint32_t Bindings::string_equal(const uint32_t handle1, const uint32_t handle2) {
    const auto it1 = strings.find(handle1);
    const auto it2 = strings.find(handle2);
    if (it1 == strings.end() || it2 == strings.end()) return 0;
    return it1->second == it2->second ? 1 : 0;
}

// ===== Series Operations (Stubs) =====
uint64_t Bindings::series_len(uint32_t handle) {
    return 0;
}
uint32_t Bindings::series_slice(uint32_t handle, uint32_t start, uint32_t end) {
    return 0;
}

// Macro to generate all series operation stubs for a given type
#define IMPL_SERIES_OPS(suffix, cpptype, default_val)                                  \
    uint32_t Bindings::series_create_empty_##suffix(uint32_t length) {                 \
        return 0;                                                                      \
    }                                                                                  \
    void Bindings::series_set_element_##suffix(                                        \
        uint32_t handle,                                                               \
        uint32_t index,                                                                \
        cpptype value                                                                  \
    ) {}                                                                               \
    cpptype Bindings::series_index_##suffix(uint32_t handle, uint32_t index) {         \
        return default_val;                                                            \
    }                                                                                  \
    uint32_t Bindings::series_element_add_##suffix(uint32_t handle, cpptype value) {   \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_element_mul_##suffix(uint32_t handle, cpptype value) {   \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_element_sub_##suffix(uint32_t handle, cpptype value) {   \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_element_div_##suffix(uint32_t handle, cpptype value) {   \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_series_add_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_series_mul_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_series_sub_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_series_div_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_compare_gt_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_compare_lt_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_compare_ge_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_compare_le_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_compare_eq_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }                                                                                  \
    uint32_t Bindings::series_compare_ne_##suffix(uint32_t a, uint32_t b) {            \
        return 0;                                                                      \
    }

IMPL_SERIES_OPS(u8, uint8_t, 0)
IMPL_SERIES_OPS(u16, uint16_t, 0)
IMPL_SERIES_OPS(u32, uint32_t, 0)
IMPL_SERIES_OPS(u64, uint64_t, 0)
IMPL_SERIES_OPS(i8, int8_t, 0)
IMPL_SERIES_OPS(i16, int16_t, 0)
IMPL_SERIES_OPS(i32, int32_t, 0)
IMPL_SERIES_OPS(i64, int64_t, 0)
IMPL_SERIES_OPS(f32, float, 0.0f)
IMPL_SERIES_OPS(f64, double, 0.0)

#undef IMPL_SERIES_OPS

// ===== Generic Operations =====

uint64_t Bindings::now() {
    auto now = std::chrono::system_clock::now();
    auto duration = now.time_since_epoch();
    auto micros = std::chrono::duration_cast<std::chrono::microseconds>(duration);
    return static_cast<uint64_t>(micros.count());
}

uint64_t Bindings::len(uint32_t handle) {
    // For now, assume it's a string handle
    return string_len(handle);
}

void Bindings::panic(uint32_t ptr, uint32_t len) {
    if (!memory || !store) {
        std::fprintf(
            stderr,
            "WASM panic: ptr=%u, len=%u (no memory or store available to read message)\n",
            ptr,
            len
        );
        throw std::runtime_error("WASM panic (no memory available)");
    }

    auto mem_span = memory->data(*store);
    const uint8_t *mem_data = mem_span.data();
    size_t mem_size = mem_span.size();

    // Bounds check
    if (ptr + len > mem_size) {
        std::fprintf(
            stderr,
            "WASM panic: ptr=%u, len=%u (out of bounds, memory size=%zu)\n",
            ptr,
            len,
            mem_size
        );
        throw std::runtime_error("WASM panic (out of bounds)");
    }

    std::string message(reinterpret_cast<const char *>(mem_data + ptr), len);
    std::fprintf(stderr, "WASM panic: %s\n", message.c_str());
    throw std::runtime_error("WASM panic: " + message);
}

// ===== Math Operations =====

template<typename T>
static T int_pow(T base, T exp) {
    if (exp == 0) return 1;
    T result = 1;
    for (T i = 0; i < exp; ++i) {
        result *= base;
    }
    return result;
}

#define IMPL_MATH_POW_FLOAT(suffix, cpptype)                                           \
    cpptype Bindings::math_pow_##suffix(cpptype base, cpptype exp) {                   \
        return std::pow(base, exp);                                                    \
    }

#define IMPL_MATH_POW_INT(suffix, cpptype)                                             \
    cpptype Bindings::math_pow_##suffix(cpptype base, cpptype exp) {                   \
        return int_pow(base, exp);                                                     \
    }

IMPL_MATH_POW_FLOAT(f32, float)
IMPL_MATH_POW_FLOAT(f64, double)

IMPL_MATH_POW_INT(u8, uint8_t)
IMPL_MATH_POW_INT(u16, uint16_t)
IMPL_MATH_POW_INT(u32, uint32_t)
IMPL_MATH_POW_INT(u64, uint64_t)
IMPL_MATH_POW_INT(i8, int8_t)
IMPL_MATH_POW_INT(i16, int16_t)
IMPL_MATH_POW_INT(i32, int32_t)
IMPL_MATH_POW_INT(i64, int64_t)

#undef IMPL_MATH_POW_FLOAT
#undef IMPL_MATH_POW_INT

// ===== Import Creation =====

std::vector<wasmtime::Extern>
create_imports(wasmtime::Store &store, Bindings *runtime) {
    std::vector<wasmtime::Extern> imports;

// ===== Channel Operations =====
// Order matters! Must match: read, write, blocking_read for each type
#define REGISTER_CHANNEL_OPS(suffix, wasm_type)                                        \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t id) -> wasm_type {              \
            return runtime->channel_read_##suffix(id);                                 \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t id, wasm_type v) {              \
            runtime->channel_write_##suffix(id, v);                                    \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t id) -> wasm_type {              \
            return runtime->channel_blocking_read_##suffix(id);                        \
        })                                                                             \
    );

    REGISTER_CHANNEL_OPS(u8, uint32_t)
    REGISTER_CHANNEL_OPS(u16, uint32_t)
    REGISTER_CHANNEL_OPS(u32, uint32_t)
    REGISTER_CHANNEL_OPS(u64, uint64_t)
    REGISTER_CHANNEL_OPS(i8, int32_t)
    REGISTER_CHANNEL_OPS(i16, int32_t)
    REGISTER_CHANNEL_OPS(i32, int32_t)
    REGISTER_CHANNEL_OPS(i64, int64_t)
    REGISTER_CHANNEL_OPS(f32, float)
    REGISTER_CHANNEL_OPS(f64, double)

#undef REGISTER_CHANNEL_OPS

    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t id) -> uint32_t {
        return runtime->channel_read_str(id);
    }));
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t id, uint32_t v) {
        runtime->channel_write_str(id, v);
    }));
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t id) -> uint32_t {
        return runtime->channel_blocking_read_str(id);
    }));

// ===== Series Operations (Per-Type) =====
// Order: create_empty, set_element, index, element ops (add,mul,sub,div), series ops
// (add,mul,sub,div), comparisons (gt,lt,ge,le,eq,ne)
#define REGISTER_SERIES_OPS(type, wasm_type)                                           \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t len) -> uint32_t {              \
            return runtime->series_create_empty_##type(len);                           \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, uint32_t i, wasm_type v) {   \
            runtime->series_set_element_##type(h, i, v);                               \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, uint32_t i) -> wasm_type {   \
            return runtime->series_index_##type(h, i);                                 \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_element_add_##type(h, v);                           \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_element_mul_##type(h, v);                           \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_element_sub_##type(h, v);                           \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_element_div_##type(h, v);                           \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_series_add_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_series_mul_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_series_sub_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_series_div_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_compare_gt_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_compare_lt_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_compare_ge_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_compare_le_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_compare_eq_##type(a, b);                            \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t a, uint32_t b) -> uint32_t {    \
            return runtime->series_compare_ne_##type(a, b);                            \
        })                                                                             \
    );

    REGISTER_SERIES_OPS(u8, uint32_t)
    REGISTER_SERIES_OPS(u16, uint32_t)
    REGISTER_SERIES_OPS(u32, uint32_t)
    REGISTER_SERIES_OPS(u64, uint64_t)
    REGISTER_SERIES_OPS(i8, int32_t)
    REGISTER_SERIES_OPS(i16, int32_t)
    REGISTER_SERIES_OPS(i32, int32_t)
    REGISTER_SERIES_OPS(i64, int64_t)
    REGISTER_SERIES_OPS(f32, float)
    REGISTER_SERIES_OPS(f64, double)

#undef REGISTER_SERIES_OPS

// ===== State Operations =====
#define REGISTER_STATE_OPS(suffix, wasm_type)                                          \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            [runtime](uint32_t fid, uint32_t vid, wasm_type init) -> wasm_type {       \
                return runtime->state_load_##suffix(fid, vid, init);                   \
            }                                                                          \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            [runtime](uint32_t fid, uint32_t vid, wasm_type v) {                       \
                runtime->state_store_##suffix(fid, vid, v);                            \
            }                                                                          \
        )                                                                              \
    );

    REGISTER_STATE_OPS(u8, uint32_t)
    REGISTER_STATE_OPS(u16, uint32_t)
    REGISTER_STATE_OPS(u32, uint32_t)
    REGISTER_STATE_OPS(u64, uint64_t)
    REGISTER_STATE_OPS(i8, int32_t)
    REGISTER_STATE_OPS(i16, int32_t)
    REGISTER_STATE_OPS(i32, int32_t)
    REGISTER_STATE_OPS(i64, int64_t)
    REGISTER_STATE_OPS(f32, float)
    REGISTER_STATE_OPS(f64, double)

#undef REGISTER_STATE_OPS

    imports.push_back(
        wasmtime::Func::wrap(
            store,
            [runtime](uint32_t fid, uint32_t vid, uint32_t init) -> uint32_t {
                return runtime->state_load_str(fid, vid, init);
            }
        )
    );
    imports.push_back(
        wasmtime::Func::wrap(store, [runtime](uint32_t fid, uint32_t vid, uint32_t v) {
            runtime->state_store_str(fid, vid, v);
        })
    );

    // ===== Generic Operations =====
    // series_len
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint64_t {
        return runtime->series_len(h);
    }));

    // series_slice
    imports.push_back(
        wasmtime::Func::wrap(
            store,
            [runtime](uint32_t h, uint32_t s, uint32_t e) -> uint32_t {
                return runtime->series_slice(h, s, e);
            }
        )
    );

    // string_from_literal
    imports.push_back(
        wasmtime::Func::wrap(store, [runtime](uint32_t ptr, uint32_t len) -> uint32_t {
            return runtime->string_from_literal(ptr, len);
        })
    );

    // string_len
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->string_len(h);
    }));

    // string_equal
    imports.push_back(
        wasmtime::Func::wrap(store, [runtime](uint32_t h1, uint32_t h2) -> uint32_t {
            return runtime->string_equal(h1, h2);
        })
    );

    // now
    imports.push_back(wasmtime::Func::wrap(store, [runtime]() -> uint64_t {
        return runtime->now();
    }));

    // len (generic)
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint64_t {
        return runtime->len(h);
    }));

// ===== Math Operations =====
#define REGISTER_MATH_POW(suffix, wasm_type)                                           \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](wasm_type b, wasm_type e) -> wasm_type { \
            return runtime->math_pow_##suffix(b, e);                                   \
        })                                                                             \
    );

    REGISTER_MATH_POW(f32, float)
    REGISTER_MATH_POW(f64, double)
    REGISTER_MATH_POW(u8, uint32_t)
    REGISTER_MATH_POW(u16, uint32_t)
    REGISTER_MATH_POW(u32, uint32_t)
    REGISTER_MATH_POW(u64, uint64_t)
    REGISTER_MATH_POW(i8, int32_t)
    REGISTER_MATH_POW(i16, int32_t)
    REGISTER_MATH_POW(i32, int32_t)
    REGISTER_MATH_POW(i64, int64_t)

#undef REGISTER_MATH_POW

    // panic
    imports.push_back(
        wasmtime::Func::wrap(store, [runtime](uint32_t ptr, uint32_t len) {
            runtime->panic(ptr, len);
        })
    );

    std::printf("Created %zu host function imports\n", imports.size());
    return imports;
}

}
