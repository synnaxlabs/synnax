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

#include "x/cpp/xmemory/local_shared.h"

#include "arc/cpp/types/types.h"
#include "bindings.h"
#include "wasmtime.hh" // For Wasmtime C++ API

namespace arc::runtime::wasm {

Bindings::Bindings(state::State *state, wasmtime::Store *store):
    state(state),
    store(store),
    memory(nullptr),
    string_handle_counter(1),
    series_handle_counter(1) {}

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
        auto data = xmemory::make_local_shared<telem::Series>(data_type_const, 1);     \
        data->write(value);                                                            \
        auto time = xmemory::make_local_shared<telem::Series>(telem::TIMESTAMP_T, 1);  \
        time->write(telem::TimeStamp::now());                                          \
        this->state                                                                    \
            ->write_channel(static_cast<types::ChannelKey>(channel_id), data, time);   \
    }

IMPL_CHANNEL_OPS(u8, uint8_t, 0, telem::UINT8_T)
IMPL_CHANNEL_OPS(u16, uint16_t, 0, telem::UINT16_T)
IMPL_CHANNEL_OPS(u32, uint32_t, 0, telem::UINT32_T)
IMPL_CHANNEL_OPS(u64, uint64_t, 0, telem::UINT64_T)
IMPL_CHANNEL_OPS(i8, int8_t, 0, telem::INT8_T)
IMPL_CHANNEL_OPS(i16, int16_t, 0, telem::INT16_T)
IMPL_CHANNEL_OPS(i32, int32_t, 0, telem::INT32_T)
IMPL_CHANNEL_OPS(i64, int64_t, 0, telem::INT64_T)
IMPL_CHANNEL_OPS(f32, float, 0.0f, telem::FLOAT32_T)
IMPL_CHANNEL_OPS(f64, double, 0.0, telem::FLOAT64_T)

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
    const auto it = strings.find(str_handle);
    if (it == strings.end()) return;
    // For STRING_T, capacity is in bytes. Allocate enough for string + newline
    // terminator.
    const size_t byte_cap = it->second.size() + 1;
    auto data = xmemory::make_local_shared<telem::Series>(telem::STRING_T, byte_cap);
    data->write(it->second);
    auto time = xmemory::make_local_shared<telem::Series>(telem::TIMESTAMP_T, 1);
    time->write(telem::TimeStamp::now());
    this->state->write_channel(static_cast<types::ChannelKey>(channel_id), data, time);
}

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
        const uint32_t handle = this->string_handle_counter++;
        this->strings[handle] = it->second;
        return handle;
    }
    if (const auto init_it = this->strings.find(init_handle); init_it != strings.end())
        this->state_string[key] = init_it->second;
    else
        this->state_string[key] = "";
    const uint32_t handle = this->string_handle_counter++;
    this->strings[handle] = this->state_string[key];
    return handle;
}

auto Bindings::state_store_str(
    const uint32_t func_id,
    const uint32_t var_id,
    const uint32_t str_handle
) -> void {
    if (const auto it = strings.find(str_handle); it != strings.end())
        state_string[state_key(func_id, var_id)] = it->second;
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
uint32_t Bindings::string_concat(uint32_t h1, uint32_t h2) {
    const auto it1 = strings.find(h1);
    const auto it2 = strings.find(h2);
    if (it1 == strings.end() || it2 == strings.end()) return 0;
    const uint32_t handle = string_handle_counter++;
    strings[handle] = it1->second + it2->second;
    return handle;
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

uint32_t Bindings::string_create(const std::string &str) {
    const uint32_t handle = string_handle_counter++;
    strings[handle] = str;
    return handle;
}

std::string Bindings::string_get(const uint32_t handle) {
    const auto it = strings.find(handle);
    if (it == strings.end()) return "";
    return it->second;
}

uint64_t Bindings::series_len(uint32_t handle) {
    auto it = series.find(handle);
    if (it == series.end()) return 0;
    return static_cast<uint64_t>(it->second.size());
}

uint32_t Bindings::series_slice(uint32_t handle, uint32_t start, uint32_t end) {
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

template<typename T>
static std::pair<telem::Series, telem::Series>
extend_to_match_length(const telem::Series &a, const telem::Series &b) {
    const auto a_len = a.size();
    const auto b_len = b.size();

    if (a_len == b_len) { return {a.deep_copy(), b.deep_copy()}; }

    const auto max_len = std::max(a_len, b_len);

    auto extend = [](const telem::Series &src, size_t target_len) -> telem::Series {
        if (src.size() >= target_len) return src.deep_copy();
        auto result = telem::Series(src.data_type(), target_len);
        result.resize(target_len);
        const auto density = src.data_type().density();
        std::memcpy(result.data(), src.data(), src.size() * density);
        if (src.size() > 0) {
            auto *data = reinterpret_cast<T *>(result.data());
            const T last_val = data[src.size() - 1];
            for (size_t i = src.size(); i < target_len; i++) {
                data[i] = last_val;
            }
        }
        return result;
    };

    return {extend(a, max_len), extend(b, max_len)};
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
        auto [lhs, rhs] = extend_to_match_length<cpptype>(it_a->second, it_b->second); \
        auto result = lhs op rhs;                                                      \
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
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, gt, >)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, lt, <)                      \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, ge, >=)                     \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, le, <=)                     \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, eq, ==)                     \
    IMPL_SERIES_BINARY_OP(suffix, cpptype, series_compare, ne, !=)                     \
    uint32_t Bindings::series_scalar_compare_gt_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second > v;                                                  \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_scalar_compare_lt_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second < v;                                                  \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_scalar_compare_ge_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second >= v;                                                 \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_scalar_compare_le_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second <= v;                                                 \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_scalar_compare_eq_##suffix(uint32_t handle, cpptype v) { \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        auto result = it->second == v;                                                 \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }                                                                                  \
    uint32_t Bindings::series_scalar_compare_ne_##suffix(uint32_t handle, cpptype v) { \
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

// Series unary operations - negate for signed types
#define IMPL_SERIES_NEGATE(suffix, cpptype, data_type_const)                           \
    uint32_t Bindings::series_negate_##suffix(uint32_t handle) {                       \
        auto it = series.find(handle);                                                 \
        if (it == series.end()) return 0;                                              \
        const auto &src = it->second;                                                  \
        const size_t len = src.size();                                                 \
        auto result = telem::Series(data_type_const, len);                             \
        result.resize(len);                                                            \
        for (size_t i = 0; i < len; i++) {                                             \
            result.set(static_cast<int>(i), -src.at<cpptype>(static_cast<int>(i)));    \
        }                                                                              \
        const uint32_t new_handle = series_handle_counter++;                           \
        series.emplace(new_handle, std::move(result));                                 \
        return new_handle;                                                             \
    }

IMPL_SERIES_NEGATE(f64, double, telem::FLOAT64_T)
IMPL_SERIES_NEGATE(f32, float, telem::FLOAT32_T)
IMPL_SERIES_NEGATE(i64, int64_t, telem::INT64_T)
IMPL_SERIES_NEGATE(i32, int32_t, telem::INT32_T)
IMPL_SERIES_NEGATE(i16, int16_t, telem::INT16_T)
IMPL_SERIES_NEGATE(i8, int8_t, telem::INT8_T)

#undef IMPL_SERIES_NEGATE

// Logical NOT for boolean series (u8)
uint32_t Bindings::series_not_u8(uint32_t handle) {
    auto it = series.find(handle);
    if (it == series.end()) return 0;
    const auto &src = it->second;
    const size_t len = src.size();
    auto result = telem::Series(telem::UINT8_T, len);
    result.resize(len);
    for (size_t i = 0; i < len; i++) {
        result.set(
            static_cast<int>(i),
            static_cast<uint8_t>(src.at<uint8_t>(static_cast<int>(i)) == 0 ? 1 : 0)
        );
    }
    const uint32_t new_handle = series_handle_counter++;
    series.emplace(new_handle, std::move(result));
    return new_handle;
}

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

// ===== Channel Operations =====
// Order matters! Must match: read, write for each type
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

// ===== Series Operations (Per-Type) =====
// Order must match Go compiler: create_empty, set_element, index,
// element ops (add,mul,sub,div,mod,rsub,rdiv), series ops (add,mul,sub,div,mod),
// comparisons (gt,lt,ge,le,eq,ne), scalar comparisons (gt,lt,ge,le,eq,ne),
// state ops (load,store)
#define REGISTER_SERIES_OPS(type, wasm_type)                                           \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t len) -> uint32_t {              \
            return runtime->series_create_empty_##type(len);                           \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            [runtime](uint32_t h, uint32_t i, wasm_type v) -> uint32_t {               \
                return runtime->series_set_element_##type(h, i, v);                    \
            }                                                                          \
        )                                                                              \
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
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_element_mod_##type(h, v);                           \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](wasm_type v, uint32_t h) -> uint32_t {   \
            return runtime->series_element_rsub_##type(v, h);                          \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](wasm_type v, uint32_t h) -> uint32_t {   \
            return runtime->series_element_rdiv_##type(v, h);                          \
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
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_scalar_compare_gt_##type(h, v);                     \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_scalar_compare_lt_##type(h, v);                     \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_scalar_compare_ge_##type(h, v);                     \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_scalar_compare_le_##type(h, v);                     \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_scalar_compare_eq_##type(h, v);                     \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(store, [runtime](uint32_t h, wasm_type v) -> uint32_t {   \
            return runtime->series_scalar_compare_ne_##type(h, v);                     \
        })                                                                             \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            [runtime](uint32_t fid, uint32_t vid, uint32_t init) -> uint32_t {         \
                return runtime->state_load_series_##type(fid, vid, init);              \
            }                                                                          \
        )                                                                              \
    );                                                                                 \
    imports.push_back(                                                                 \
        wasmtime::Func::wrap(                                                          \
            store,                                                                     \
            [runtime](uint32_t fid, uint32_t vid, uint32_t h) {                        \
                runtime->state_store_series_##type(fid, vid, h);                       \
            }                                                                          \
        )                                                                              \
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

    // ===== Series Unary Operations =====
    // Order: negate for signed types (f64, f32, i64, i32, i16, i8), then not for u8
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->series_negate_f64(h);
    }));
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->series_negate_f32(h);
    }));
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->series_negate_i64(h);
    }));
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->series_negate_i32(h);
    }));
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->series_negate_i16(h);
    }));
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->series_negate_i8(h);
    }));
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->series_not_u8(h);
    }));

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

    // string_concat
    imports.push_back(
        wasmtime::Func::wrap(store, [runtime](uint32_t h1, uint32_t h2) -> uint32_t {
            return runtime->string_concat(h1, h2);
        })
    );

    // string_equal
    imports.push_back(
        wasmtime::Func::wrap(store, [runtime](uint32_t h1, uint32_t h2) -> uint32_t {
            return runtime->string_equal(h1, h2);
        })
    );

    // string_len
    imports.push_back(wasmtime::Func::wrap(store, [runtime](uint32_t h) -> uint32_t {
        return runtime->string_len(h);
    }));

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
