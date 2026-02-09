// Copyright 2026 Synnax Labs, Inc.
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

#include "x/cpp/mem/local_shared.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/types/types.h"
#include "bindings.h"
#include "wasmtime.hh"

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

Bindings::Bindings(
    const std::shared_ptr<state::State> &state,
    wasmtime::Store *store,
    errors::Handler error_handler
):
    state(state),
    store(store),
    memory(nullptr),
    error_handler(std::move(error_handler)) {}

#define IMPL_CHANNEL_OPS(suffix, cpptype, default_val, data_type_const)                \
    cpptype Bindings::channel_read_##suffix(uint32_t channel_id) {                     \
        if (this->state == nullptr) return default_val;                                \
        auto [multi_series, ok] = this->state->read_channel(                           \
            static_cast<types::ChannelKey>(channel_id)                                 \
        );                                                                             \
        if (!ok || multi_series.series.empty()) return default_val;                    \
        const auto &last_series = multi_series.series.back();                          \
        if (last_series.size() == 0) return default_val;                               \
        return last_series.at<cpptype>(-1);                                            \
    }                                                                                  \
    void Bindings::channel_write_##suffix(uint32_t channel_id, cpptype value) {        \
        if (this->state == nullptr) return;                                            \
        auto data = x::mem::make_local_shared<x::telem::Series>(                       \
            value,                                                                     \
            data_type_const                                                            \
        );                                                                             \
        auto time = x::mem::make_local_shared<x::telem::Series>(                       \
            x::telem::TimeStamp::now()                                                 \
        );                                                                             \
        this->state                                                                    \
            ->write_channel(static_cast<types::ChannelKey>(channel_id), data, time);   \
    }

IMPL_CHANNEL_OPS(u8, uint8_t, 0, x::telem::UINT8_T)
IMPL_CHANNEL_OPS(u16, uint16_t, 0, x::telem::UINT16_T)
IMPL_CHANNEL_OPS(u32, uint32_t, 0, x::telem::UINT32_T)
IMPL_CHANNEL_OPS(u64, uint64_t, 0, x::telem::UINT64_T)
IMPL_CHANNEL_OPS(i8, int8_t, 0, x::telem::INT8_T)
IMPL_CHANNEL_OPS(i16, int16_t, 0, x::telem::INT16_T)
IMPL_CHANNEL_OPS(i32, int32_t, 0, x::telem::INT32_T)
IMPL_CHANNEL_OPS(i64, int64_t, 0, x::telem::INT64_T)
IMPL_CHANNEL_OPS(f32, float, 0.0f, x::telem::FLOAT32_T)
IMPL_CHANNEL_OPS(f64, double, 0.0, x::telem::FLOAT64_T)

#undef IMPL_CHANNEL_OPS

uint32_t Bindings::channel_read_str(uint32_t channel_id) {
    if (this->state == nullptr) return 0;
    auto [multi_series, ok] = this->state->read_channel(channel_id);
    if (!ok || multi_series.series.empty()) return 0;
    const auto &last_series = multi_series.series.back();
    if (last_series.size() == 0) return 0;
    std::string str = last_series.at<std::string>(-1);
    return string_create(str);
}

void Bindings::channel_write_str(uint32_t channel_id, uint32_t str_handle) {
    if (this->state == nullptr) return;
    std::string str_value = this->state->string_get(str_handle);
    if (str_value.empty()) return;
    const auto data = x::mem::make_local_shared<x::telem::Series>(str_value);
    const auto time = x::mem::make_local_shared<x::telem::Series>(
        x::telem::TimeStamp::now()
    );
    this->state->write_channel(static_cast<types::ChannelKey>(channel_id), data, time);
}

#define IMPL_STATE_OPS(suffix, cpptype)                                                \
    cpptype Bindings::state_load_##suffix(uint32_t var_id, cpptype init_value) {       \
        if (this->state == nullptr) return init_value;                                 \
        return this->state->var_load_##suffix(var_id, init_value);                     \
    }                                                                                  \
    void Bindings::state_store_##suffix(uint32_t var_id, cpptype value) {              \
        if (this->state == nullptr) return;                                            \
        this->state->var_store_##suffix(var_id, value);                                \
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

uint32_t Bindings::state_load_str(const uint32_t var_id, const uint32_t init_handle) {
    if (this->state == nullptr) return init_handle;
    return this->state->var_load_str(var_id, init_handle);
}

void Bindings::state_store_str(const uint32_t var_id, const uint32_t str_handle) {
    if (this->state == nullptr) return;
    this->state->var_store_str(var_id, str_handle);
}

uint64_t Bindings::series_len(const uint32_t handle) {
    if (this->state == nullptr) return 0;
    const auto *s = this->state->series_get(handle);
    if (s == nullptr) return 0;
    return s->size();
}

uint32_t Bindings::series_slice(
    const uint32_t handle,
    const uint32_t start,
    const uint32_t end
) {
    if (this->state == nullptr) return 0;
    const auto *src = this->state->series_get(handle);
    if (src == nullptr) return 0;
    const auto src_size = src->size();
    if (start >= src_size || end > src_size || start >= end) return 0;
    const auto slice_len = end - start;
    auto sliced = x::telem::Series(src->data_type(), slice_len);
    const auto density = src->data_type().density();
    std::memcpy(sliced.data(), src->data() + start * density, slice_len * density);
    sliced.resize(slice_len);
    return this->state->series_store(std::move(sliced));
}

uint32_t Bindings::string_from_literal(const uint32_t ptr, const uint32_t len) {
    if (!memory || !store) {
        std::fprintf(
            stderr,
            "ERROR: string_from_literal called but no memory or store available\n"
        );
        return 0;
    }
    if (this->state == nullptr) return 0;

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

    return this->state->string_from_memory(mem_data + ptr, len);
}

uint32_t Bindings::string_concat(const uint32_t handle1, const uint32_t handle2) {
    if (this->state == nullptr) return 0;
    const std::string s1 = this->state->string_get(handle1);
    const std::string s2 = this->state->string_get(handle2);
    if (s1.empty() && s2.empty()) return 0;
    return this->state->string_create(s1 + s2);
}

uint32_t Bindings::string_equal(const uint32_t handle1, const uint32_t handle2) {
    if (this->state == nullptr) return 0;
    if (!this->state->string_exists(handle1) || !this->state->string_exists(handle2))
        return 0;
    const std::string s1 = this->state->string_get(handle1);
    const std::string s2 = this->state->string_get(handle2);
    return s1 == s2 ? 1 : 0;
}

uint32_t Bindings::string_len(const uint32_t handle) {
    if (this->state == nullptr) return 0;
    const std::string s = this->state->string_get(handle);
    return static_cast<uint32_t>(s.length());
}

uint32_t Bindings::string_create(const std::string &str) {
    if (this->state == nullptr) return 0;
    return this->state->string_create(str);
}

std::string Bindings::string_get(const uint32_t handle) const {
    if (this->state == nullptr) return "";
    return this->state->string_get(handle);
}

#define IMPL_SERIES_SCALAR_OP(suffix, cpptype, name, op)                               \
    uint32_t Bindings::series_element_##name##_##suffix(uint32_t handle, cpptype v) {  \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = *s op v;                                                         \
        return this->state->series_store(std::move(result));                           \
    }

#define IMPL_SERIES_BINARY_OP(suffix, cpptype, prefix, name, op)                       \
    uint32_t Bindings::prefix##_##name##_##suffix(uint32_t a, uint32_t b) {            \
        if (this->state == nullptr) return 0;                                          \
        auto *sa = this->state->series_get(a);                                         \
        auto *sb = this->state->series_get(b);                                         \
        if (sa == nullptr || sb == nullptr) return 0;                                  \
        if (sa->size() != sb->size())                                                  \
            throw std::runtime_error("arc panic: series length mismatch in " #name);   \
        auto result = *sa op * sb;                                                     \
        return this->state->series_store(std::move(result));                           \
    }

#define IMPL_SERIES_OPS(suffix, cpptype, data_type_const)                              \
    uint32_t Bindings::series_create_empty_##suffix(uint32_t length) {                 \
        if (this->state == nullptr) return 0;                                          \
        auto s = x::telem::Series(data_type_const, static_cast<size_t>(length));       \
        s.resize(length);                                                              \
        return this->state->series_store(std::move(s));                                \
    }                                                                                  \
    uint32_t Bindings::series_set_element_##suffix(                                    \
        uint32_t handle,                                                               \
        uint32_t index,                                                                \
        cpptype value                                                                  \
    ) {                                                                                \
        if (this->state == nullptr) return handle;                                     \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return handle;                                               \
        s->set(static_cast<int>(index), value);                                        \
        return handle;                                                                 \
    }                                                                                  \
    cpptype Bindings::series_index_##suffix(uint32_t handle, uint32_t index) {         \
        if (this->state == nullptr) return cpptype{};                                  \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return cpptype{};                                            \
        return s->at<cpptype>(static_cast<int>(index));                                \
    }                                                                                  \
    IMPL_SERIES_SCALAR_OP(suffix, cpptype, add, +)                                     \
    IMPL_SERIES_SCALAR_OP(suffix, cpptype, mul, *)                                     \
    IMPL_SERIES_SCALAR_OP(suffix, cpptype, sub, -)                                     \
    uint32_t Bindings::series_element_div_##suffix(uint32_t handle, cpptype value) {   \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        if (value == 0) return 0;                                                      \
        auto result = *s / value;                                                      \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_element_rsub_##suffix(cpptype value, uint32_t handle) {  \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = value - *s;                                                      \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_element_rdiv_##suffix(cpptype value, uint32_t handle) {  \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = value / *s;                                                      \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_element_radd_##suffix(cpptype value, uint32_t handle) {  \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = value + *s;                                                      \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_element_rmul_##suffix(cpptype value, uint32_t handle) {  \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = value * *s;                                                      \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_element_rmod_##suffix(cpptype value, uint32_t handle) {  \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = value % *s;                                                      \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_element_mod_##suffix(uint32_t handle, cpptype value) {   \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        if (value == 0) return 0;                                                      \
        auto result = *s % value;                                                      \
        return this->state->series_store(std::move(result));                           \
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
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = *s > v;                                                          \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_compare_lt_scalar_##suffix(uint32_t handle, cpptype v) { \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = *s < v;                                                          \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_compare_ge_scalar_##suffix(uint32_t handle, cpptype v) { \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = *s >= v;                                                         \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_compare_le_scalar_##suffix(uint32_t handle, cpptype v) { \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = *s <= v;                                                         \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_compare_eq_scalar_##suffix(uint32_t handle, cpptype v) { \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = *s == v;                                                         \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::series_compare_ne_scalar_##suffix(uint32_t handle, cpptype v) { \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = *s != v;                                                         \
        return this->state->series_store(std::move(result));                           \
    }                                                                                  \
    uint32_t Bindings::state_load_series_##suffix(                                     \
        uint32_t var_id,                                                               \
        uint32_t init_handle                                                           \
    ) {                                                                                \
        if (this->state == nullptr) return init_handle;                                \
        return this->state->var_load_series(var_id, init_handle);                      \
    }                                                                                  \
    void Bindings::state_store_series_##suffix(uint32_t var_id, uint32_t handle) {     \
        if (this->state == nullptr) return;                                            \
        this->state->var_store_series(var_id, handle);                                 \
    }

IMPL_SERIES_OPS(u8, uint8_t, x::telem::UINT8_T)
IMPL_SERIES_OPS(u16, uint16_t, x::telem::UINT16_T)
IMPL_SERIES_OPS(u32, uint32_t, x::telem::UINT32_T)
IMPL_SERIES_OPS(u64, uint64_t, x::telem::UINT64_T)
IMPL_SERIES_OPS(i8, int8_t, x::telem::INT8_T)
IMPL_SERIES_OPS(i16, int16_t, x::telem::INT16_T)
IMPL_SERIES_OPS(i32, int32_t, x::telem::INT32_T)
IMPL_SERIES_OPS(i64, int64_t, x::telem::INT64_T)
IMPL_SERIES_OPS(f32, float, x::telem::FLOAT32_T)
IMPL_SERIES_OPS(f64, double, x::telem::FLOAT64_T)

#undef IMPL_SERIES_OPS
#undef IMPL_SERIES_SCALAR_OP
#undef IMPL_SERIES_BINARY_OP

// Unary negate operations (signed types only)
#define IMPL_SERIES_NEGATE(suffix)                                                     \
    uint32_t Bindings::series_negate_##suffix(uint32_t handle) {                       \
        if (this->state == nullptr) return 0;                                          \
        auto *s = this->state->series_get(handle);                                     \
        if (s == nullptr) return 0;                                                    \
        auto result = -*s;                                                             \
        return this->state->series_store(std::move(result));                           \
    }

IMPL_SERIES_NEGATE(i8)
IMPL_SERIES_NEGATE(i16)
IMPL_SERIES_NEGATE(i32)
IMPL_SERIES_NEGATE(i64)
IMPL_SERIES_NEGATE(f32)
IMPL_SERIES_NEGATE(f64)

#undef IMPL_SERIES_NEGATE

// Logical NOT (U8 only - for boolean negation: 0 -> 1, non-zero -> 0)
uint32_t Bindings::series_not_u8(uint32_t handle) {
    if (this->state == nullptr) return 0;
    auto *s = this->state->series_get(handle);
    if (s == nullptr) return 0;
    auto result = s->logical_not();
    return this->state->series_store(std::move(result));
}

uint64_t Bindings::now() {
    return static_cast<uint64_t>(x::telem::TimeStamp::now().nanoseconds());
}

uint64_t Bindings::len(const uint32_t handle) {
    return string_len(handle);
}

void Bindings::panic(const uint32_t ptr, const uint32_t len) {
    std::string message;
    if (!memory || !store) {
        message = "no memory available";
    } else {
        const auto mem_span = memory->data(*store);
        const uint8_t *mem_data = mem_span.data();
        const size_t mem_size = mem_span.size();
        if (ptr + len > mem_size)
            message = "out of bounds";
        else
            message = std::string(reinterpret_cast<const char *>(mem_data + ptr), len);
    }
    std::fprintf(stderr, "WASM panic: %s\n", message.c_str());
    this->error_handler(x::errors::Error(errors::WASM_PANIC, message));
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

std::vector<wasmtime::Extern>
create_imports(wasmtime::Store &store, std::shared_ptr<Bindings> runtime) {
    std::vector<wasmtime::Extern> imports;

/// Channel ops use wrap() which auto-converts C++ types to WASM types via WasmType
/// trait
#define REGISTER_CHANNEL_OPS(suffix)                                                   \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::channel_read_##suffix)                      \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::channel_write_##suffix)                     \
        )                                                                              \
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
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::channel_read_str))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::channel_write_str))
    );

#define REGISTER_SERIES_OPS(suffix)                                                    \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_create_empty_##suffix)               \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_set_element_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_index_##suffix)                      \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_add_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_mul_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_sub_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_div_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_mod_##suffix)                \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_rsub_##suffix)               \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_rdiv_##suffix)               \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_radd_##suffix)               \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_rmul_##suffix)               \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_element_rmod_##suffix)               \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_series_add_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_series_mul_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_series_sub_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_series_div_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_series_mod_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_gt_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_lt_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_ge_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_le_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_eq_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_ne_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_gt_scalar_##suffix)          \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_lt_scalar_##suffix)          \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_ge_scalar_##suffix)          \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_le_scalar_##suffix)          \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_eq_scalar_##suffix)          \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::series_compare_ne_scalar_##suffix)          \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::state_load_series_##suffix)                 \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::state_store_series_##suffix)                \
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
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_negate_f64))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_negate_f32))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_negate_i64))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_negate_i32))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_negate_i16))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_negate_i8))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_not_u8))
    );

#define REGISTER_STATE_OPS(suffix)                                                     \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::state_load_##suffix)                        \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            wrap(runtime.get(), &Bindings::state_store_##suffix)                       \
        )                                                                              \
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
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::state_load_str))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::state_store_str))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_len))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::series_slice))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::string_from_literal))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::string_concat))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::string_equal))
    );
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::string_len))
    );
    imports.push_back(wasmtime::Func::wrap(store, &Bindings::now));
    imports.push_back(wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::len)));

#define REGISTER_MATH_POW(suffix)                                                      \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::math_pow_##suffix)) \
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
    imports.push_back(
        wasmtime::Func::wrap(store, wrap(runtime.get(), &Bindings::panic))
    );
    return imports;
}
}
