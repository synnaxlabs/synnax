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

/// WasmType maps C++ types to their WASM-compatible equivalents.
/// WASM only has i32, i64, f32, f64 - smaller integer types must be widened.
template<typename T>
struct WasmType {
    using type = T;
};
template<>
struct WasmType<uint8_t> {
    using type = uint32_t;
};
template<>
struct WasmType<uint16_t> {
    using type = uint32_t;
};
template<>
struct WasmType<int8_t> {
    using type = int32_t;
};
template<>
struct WasmType<int16_t> {
    using type = int32_t;
};

/// MethodWrapper provides a callable with WASM-compatible operator() signature,
/// enabling wasmtime::Func::wrap to deduce the function type correctly.
/// Automatically converts between C++ types and WASM types using WasmType trait.
template<typename C, typename R, typename... Args>
struct MethodWrapper {
    C *obj;
    R (C::*fn)(Args...);

    using WasmR = typename WasmType<R>::type;

    WasmR operator()(typename WasmType<Args>::type... args) const {
        return static_cast<WasmR>((obj->*fn)(static_cast<Args>(args)...));
    }
};

template<typename C, typename R, typename... Args>
auto wrap(C *obj, R (C::*fn)(Args...)) {
    return MethodWrapper<C, R, Args...>{obj, fn};
}

Bindings::Bindings(state::State *state, wasmtime::Store *store):
    state(state),
    store(store),
    memory(nullptr),
    string_handle_counter(1),
    series_handle_counter(1) {}

#define IMPL_CHANNEL_OPS(suffix, cpptype, default_val)                                 \
    cpptype Bindings::channel_read_##suffix(uint32_t channel_id) {                     \
        return default_val;                                                            \
    }                                                                                  \
    void Bindings::channel_write_##suffix(uint32_t channel_id, cpptype value) {}

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

uint64_t Bindings::series_len(const uint32_t handle) {
    const auto it = series.find(handle);
    if (it == series.end()) return 0;
    return it->second.size();
}
uint32_t Bindings::series_slice(const uint32_t handle, uint32_t start, uint32_t end) {
    auto it = series.find(handle);
    if (it == series.end()) return 0;
    const auto &src = it->second;
    const auto src_size = src.size();
    if (start >= src_size || end > src_size || start >= end) return 0;
    const auto slice_len = end - start;
    auto sliced = telem::Series(src.data_type(), slice_len);
    const auto density = src.data_type().density();
    std::memcpy(sliced.data(), src.data() + start * density, slice_len * density);
    sliced.resize(slice_len);
    const uint32_t new_handle = series_handle_counter++;
    series.emplace(new_handle, std::move(sliced));
    return new_handle;
}

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

uint32_t Bindings::string_concat(const uint32_t handle1, const uint32_t handle2) {
    const auto it1 = strings.find(handle1);
    const auto it2 = strings.find(handle2);
    if (it1 == strings.end() || it2 == strings.end()) return 0;
    const std::string result = it1->second + it2->second;
    const uint32_t new_handle = string_handle_counter++;
    strings[new_handle] = result;
    return new_handle;
}

uint32_t Bindings::string_equal(const uint32_t handle1, const uint32_t handle2) {
    const auto it1 = strings.find(handle1);
    const auto it2 = strings.find(handle2);
    if (it1 == strings.end() || it2 == strings.end()) return 0;
    return it1->second == it2->second ? 1 : 0;
}

uint32_t Bindings::string_len(const uint32_t handle) {
    const auto it = strings.find(handle);
    if (it == strings.end()) return 0;
    return static_cast<uint32_t>(it->second.length());
}

#define IMPL_SERIES_SCALAR_OP(suffix, cpptype, name, op)                               \
    uint32_t Bindings::series_element_##name##_##suffix(uint32_t handle, cpptype v) {  \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second op v;                                                 \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }

#define IMPL_SERIES_BINARY_OP(suffix, cpptype, prefix, name, op)                       \
    uint32_t Bindings::prefix##_##name##_##suffix(uint32_t a, uint32_t b) {            \
        auto it_a = series.find(a);                                                    \
        auto it_b = series.find(b);                                                    \
        if (it_a == series.end() || it_b == series.end()) return 0;                    \
        if (it_a->second.size() != it_b->second.size())                                \
            throw std::runtime_error("arc panic: series length mismatch in " #name);   \
        auto result = it_a->second op it_b->second;                                    \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }

#define IMPL_SERIES_OPS(suffix, cpptype, data_type_const)                              \
    uint32_t Bindings::series_create_empty_##suffix(uint32_t length) {                 \
        auto s = telem::Series(data_type_const, static_cast<size_t>(length));          \
        s.resize(length);                                                              \
        const uint32_t handle = series_handle_counter++;                               \
        series.emplace(handle, std::move(s));                                          \
        return handle;                                                                 \
    }                                                                                  \
    uint32_t Bindings::series_set_element_##suffix(                                    \
        uint32_t handle,                                                               \
        uint32_t index,                                                                \
        cpptype value                                                                  \
    ) {                                                                                \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return handle;                                         \
        it->second.set(static_cast<int>(index), value);                                \
        return handle;                                                                 \
    }                                                                                  \
    cpptype Bindings::series_index_##suffix(uint32_t handle, uint32_t index) {         \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return cpptype{};                                      \
        return it->second.at<cpptype>(static_cast<int>(index));                        \
    }                                                                                  \
    IMPL_SERIES_SCALAR_OP(suffix, cpptype, add, +)                                     \
    IMPL_SERIES_SCALAR_OP(suffix, cpptype, mul, *)                                     \
    IMPL_SERIES_SCALAR_OP(suffix, cpptype, sub, -)                                     \
    uint32_t Bindings::series_element_div_##suffix(uint32_t handle, cpptype value) {   \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        if (value == 0) return 0;                                                      \
        auto result = it->second / value;                                              \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_element_rsub_##suffix(cpptype value, uint32_t handle) {  \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = value - it->second;                                              \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_element_rdiv_##suffix(cpptype value, uint32_t handle) {  \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = value / it->second;                                              \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_element_mod_##suffix(uint32_t handle, cpptype value) {   \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        if (value == 0) return 0;                                                      \
        auto result = it->second % value;                                              \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_series, add, +)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_series, mul, *)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_series, sub, -)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_series, div, /)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_series, mod, %)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, gt, >)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, lt, <)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, ge, >=)                     \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, le, <=)                     \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, eq, ==)                     \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, ne, !=)                     \
    uint32_t Bindings::series_compare_gt_scalar_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second > v;                                                  \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_compare_lt_scalar_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second < v;                                                  \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_compare_ge_scalar_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second >= v;                                                 \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_compare_le_scalar_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second <= v;                                                 \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_compare_eq_scalar_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second == v;                                                 \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_compare_ne_scalar_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second != v;                                                 \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::state_load_series_##suffix(                                     \
        uint32_t func_id,                                                              \
        uint32_t var_id,                                                               \
        uint32_t init_handle                                                           \
    ) {                                                                                \
        const auto key = state_key(func_id, var_id);                                   \
        auto state_it = state_series.find(key);                                        \
        if (state_it != state_series.end()) {                                          \
            auto copy = state_it->second.deep_copy();                                  \
            const uint32_t handle = series_handle_counter++;                           \
            series.emplace(handle, std::move(copy));                                   \
            return handle;                                                             \
        }                                                                              \
        auto init_it = series.find(init_handle);                                       \
        if (init_it != series.end()) {                                                 \
            state_series.emplace(key, init_it->second.deep_copy());                    \
        }                                                                              \
        return init_handle;                                                            \
    }                                                                                  \
    void Bindings::state_store_series_##suffix(                                        \
        uint32_t func_id,                                                              \
        uint32_t var_id,                                                               \
        uint32_t handle                                                                \
    ) {                                                                                \
        auto it = series.find(handle);                                                 \
        if (it != series.end()) {                                                      \
            const auto key = state_key(func_id, var_id);                               \
            auto state_it = state_series.find(key);                                    \
            if (state_it != state_series.end()) { state_series.erase(state_it); }      \
            state_series.emplace(key, it->second.deep_copy());                         \
        }                                                                              \
    }

IMPL_SERIES_OPS(u8, uint8_t, telem::UINT8_T)
IMPL_SERIES_OPS(u16, uint16_t, telem::UINT16_T)
IMPL_SERIES_OPS(u32, uint32_t, telem::UINT32_T)
IMPL_SERIES_OPS(u64, uint64_t, telem::UINT64_T)
IMPL_SERIES_OPS(i8, int8_t, telem::INT8_T)
IMPL_SERIES_OPS(i16, int16_t, telem::INT16_T)
IMPL_SERIES_OPS(i32, int32_t, telem::INT32_T)
IMPL_SERIES_OPS(i64, int64_t, telem::INT64_T)
IMPL_SERIES_OPS(f32, float, telem::FLOAT32_T)
IMPL_SERIES_OPS(f64, double, telem::FLOAT64_T)

#undef IMPL_SERIES_OPS
#undef IMPL_SERIES_SCALAR_OP
#undef IMPL_SERIES_BINARY_OP

// Unary negate operations (signed types only)
#define IMPL_SERIES_NEGATE(suffix)                                                     \
    uint32_t Bindings::series_negate_##suffix(uint32_t handle) {                       \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = -it->second;                                                     \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }

IMPL_SERIES_NEGATE(i8)
IMPL_SERIES_NEGATE(i16)
IMPL_SERIES_NEGATE(i32)
IMPL_SERIES_NEGATE(i64)
IMPL_SERIES_NEGATE(f32)
IMPL_SERIES_NEGATE(f64)

#undef IMPL_SERIES_NEGATE

// Boolean NOT (U8 only - for logical negation)
uint32_t Bindings::series_not_u8(uint32_t handle) {
    auto it = series.find(handle);
    if (it == series.end()) return 0;
    auto result = ~it->second;
    const uint32_t new_handle = series_handle_counter++;
    series.emplace(new_handle, std::move(result));
    return new_handle;
}

uint64_t Bindings::now() {
    return static_cast<uint64_t>(telem::TimeStamp::now().nanoseconds());
}

uint64_t Bindings::len(const uint32_t handle) {
    return string_len(handle);
}

void Bindings::panic(const uint32_t ptr, const uint32_t len) {
    if (!memory || !store) {
        std::fprintf(
            stderr,
            "WASM panic: ptr=%u, len=%u (no memory or store available to read message)\n",
            ptr,
            len
        );
        throw std::runtime_error("WASM panic (no memory available)");
    }

    const auto mem_span = memory->data(*store);
    const uint8_t *mem_data = mem_span.data();
    const size_t mem_size = mem_span.size();

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

    const std::string message(reinterpret_cast<const char *>(mem_data + ptr), len);
    std::fprintf(stderr, "WASM panic: %s\n", message.c_str());
    throw std::runtime_error("WASM panic: " + message);
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

void Bindings::clear_transient_handles() {
    // Clear transient series storage and reset counter.
    // state_series is NOT cleared as it holds stateful variables.
    this->series.clear();
    this->series_handle_counter = 1;

    // Clear transient string storage and reset counter.
    // state_string is NOT cleared as it holds stateful variables.
    this->strings.clear();
    this->string_handle_counter = 1;
}

// ===== Import Creation =====

std::vector<wasmtime::Extern>
create_imports(wasmtime::Store &store, Bindings *runtime) {
    std::vector<wasmtime::Extern> imports;

/// Channel ops use wrap() which auto-converts C++ types to WASM types via WasmType
/// trait
#define REGISTER_CHANNEL_OPS(suffix)                                                   \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::channel_read_##suffix))   \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::channel_write_##suffix))  \
    );

    REGISTER_CHANNEL_OPS(u8)
    REGISTER_CHANNEL_OPS(u16)
    REGISTER_CHANNEL_OPS(u32)
    REGISTER_CHANNEL_OPS(u64)
    REGISTER_CHANNEL_OPS(i8)
    REGISTER_CHANNEL_OPS(i16)
    REGISTER_CHANNEL_OPS(i32)
    REGISTER_CHANNEL_OPS(i64)
    REGISTER_CHANNEL_OPS(f32)
    REGISTER_CHANNEL_OPS(f64)

#undef REGISTER_CHANNEL_OPS

    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::channel_read_str))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::channel_write_str))
    );

#define REGISTER_SERIES_OPS(suffix)                                                    \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_create_empty_##suffix)                     \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_set_element_##suffix)                      \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_index_##suffix))   \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_element_add_##suffix)                      \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_element_mul_##suffix)                      \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_element_sub_##suffix)                      \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_element_div_##suffix)                      \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_element_mod_##suffix)                      \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_element_rsub_##suffix)                     \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_element_rdiv_##suffix)                     \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_series_add_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_series_mul_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_series_sub_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_series_div_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_series_mod_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_gt_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_lt_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_ge_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_le_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_eq_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_ne_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_gt_scalar_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_lt_scalar_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_ge_scalar_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_le_scalar_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_eq_scalar_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::series_compare_ne_scalar_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::state_load_series_##suffix)                       \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime, &Bindings::state_store_series_##suffix)                      \
        )                                                                              \
    );

    REGISTER_SERIES_OPS(u8)
    REGISTER_SERIES_OPS(u16)
    REGISTER_SERIES_OPS(u32)
    REGISTER_SERIES_OPS(u64)
    REGISTER_SERIES_OPS(i8)
    REGISTER_SERIES_OPS(i16)
    REGISTER_SERIES_OPS(i32)
    REGISTER_SERIES_OPS(i64)
    REGISTER_SERIES_OPS(f32)
    REGISTER_SERIES_OPS(f64)

#undef REGISTER_SERIES_OPS

    // Register unary operations (negate for signed types, NOT for boolean)
    // Order matches Go: F64, F32, I64, I32, I16, I8 for negate, then U8 for NOT
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_negate_f64))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_negate_f32))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_negate_i64))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_negate_i32))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_negate_i16))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_negate_i8))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_not_u8))
    );

#define REGISTER_STATE_OPS(suffix)                                                     \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::state_load_##suffix))     \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::state_store_##suffix))    \
    );

    REGISTER_STATE_OPS(u8)
    REGISTER_STATE_OPS(u16)
    REGISTER_STATE_OPS(u32)
    REGISTER_STATE_OPS(u64)
    REGISTER_STATE_OPS(i8)
    REGISTER_STATE_OPS(i16)
    REGISTER_STATE_OPS(i32)
    REGISTER_STATE_OPS(i64)
    REGISTER_STATE_OPS(f32)
    REGISTER_STATE_OPS(f64)

#undef REGISTER_STATE_OPS
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::state_load_str))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::state_store_str))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_len))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::series_slice))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::string_from_literal))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::string_concat))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::string_equal))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::string_len))
    );
    imports.push_back(wasmtime::Func::wrap(store, &Bindings::now));
    imports.push_back(wasmtime::Func::wrap(store, wrap(runtime, &Bindings::len)));

#define REGISTER_MATH_POW(suffix)                                                      \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, wrap(runtime, &Bindings::math_pow_##suffix))       \
    );

    REGISTER_MATH_POW(f32)
    REGISTER_MATH_POW(f64)
    REGISTER_MATH_POW(u8)
    REGISTER_MATH_POW(u16)
    REGISTER_MATH_POW(u32)
    REGISTER_MATH_POW(u64)
    REGISTER_MATH_POW(i8)
    REGISTER_MATH_POW(i16)
    REGISTER_MATH_POW(i32)
    REGISTER_MATH_POW(i64)

#undef REGISTER_MATH_POW
    imports.push_back(wasmtime::Func::wrap(store, wrap(runtime, &Bindings::panic)));
    std::printf("Created %zu host function imports\n", imports.size());
    return imports;
}
}
