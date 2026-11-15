// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstdint>
#include <optional>
#include <string>

#include "wasmtime.hh"

#include "arc/cpp/stl/series/series.h"

namespace arc::stl::series {

namespace {
/// @brief returns a trap when index is outside a series of size samples. A negative
/// index counts from the end.
std::optional<wasmtime::Trap> check_index(const uint32_t index, const size_t size) {
    const auto i = static_cast<int32_t>(index);
    const auto n = static_cast<int64_t>(size);
    if (i >= -n && i < n) return std::nullopt;
    return wasmtime::Trap(
        "index " + std::to_string(i) + " out of bounds for series of size " +
        std::to_string(size)
    );
}

/// @brief returns bound as an offset into a series of size samples. A negative bound
/// counts from the end, and a bound outside the series stops at the nearer end.
size_t slice_bound(const uint32_t bound, const size_t size) {
    const auto n = static_cast<int64_t>(size);
    auto b = static_cast<int64_t>(static_cast<int32_t>(bound));
    if (b < 0) b += n;
    return static_cast<size_t>(std::clamp<int64_t>(b, 0, n));
}

/// @brief returns a trap for an index that does not fit in an i32.
wasmtime::Trap index_overflow(const std::string &index) {
    return wasmtime::Trap("index " + index + " out of bounds for an i32 series index");
}
}

// NOLINTNEXTLINE(readability-function-size): one binding macro per host function.
void Module::bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) {
    auto ss = this->series_state;

// BIND_CREATE_EMPTY registers create_empty_<suffix>, the allocator shared by
// every series element type.
#define BIND_CREATE_EMPTY(suffix, data_type_const)                                     \
    linker                                                                             \
        .func_wrap(                                                                    \
            MODULE_NAME,                                                               \
            "create_empty_" #suffix,                                                   \
            [ss](uint32_t length) -> uint32_t {                                        \
                auto s = x::telem::Series(                                             \
                    data_type_const,                                                   \
                    static_cast<size_t>(length)                                        \
                );                                                                     \
                s.resize(length);                                                      \
                return ss->store(std::move(s));                                        \
            }                                                                          \
        )                                                                              \
        .unwrap();

#define BIND_SERIES_OPS(suffix, cpptype, data_type_const)                              \
    {                                                                                  \
        using W = typename WasmType<cpptype>::type;                                    \
        BIND_CREATE_EMPTY(suffix, data_type_const)                                     \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "set_element_" #suffix,                                                \
                [ss](uint32_t handle, uint32_t index, W value)                         \
                    -> wasmtime::Result<uint32_t, wasmtime::Trap> {                    \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return handle;                                   \
                    if (auto trap = check_index(index, s->size()))                     \
                        return std::move(*trap);                                       \
                    s->set(static_cast<int>(index), static_cast<cpptype>(value));      \
                    return handle;                                                     \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "index_" #suffix,                                                      \
                [ss](uint32_t handle, uint32_t index)                                  \
                    -> wasmtime::Result<W, wasmtime::Trap> {                           \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return W{};                                      \
                    if (auto trap = check_index(index, s->size()))                     \
                        return std::move(*trap);                                       \
                    return static_cast<W>(s->at<cpptype>(static_cast<int>(index)));    \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_add_" #suffix,                                                \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s + static_cast<cpptype>(v);                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_mul_" #suffix,                                                \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s * static_cast<cpptype>(v);                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_sub_" #suffix,                                                \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s - static_cast<cpptype>(v);                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_div_" #suffix,                                                \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    if (static_cast<cpptype>(v) == 0) return 0;                        \
                    auto result = *s / static_cast<cpptype>(v);                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_mod_" #suffix,                                                \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    if (static_cast<cpptype>(v) == 0) return 0;                        \
                    auto result = *s % static_cast<cpptype>(v);                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_rsub_" #suffix,                                               \
                [ss](W v, uint32_t handle) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = static_cast<cpptype>(v) - *s;                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_rdiv_" #suffix,                                               \
                [ss](W v, uint32_t handle) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = static_cast<cpptype>(v) / *s;                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_radd_" #suffix,                                               \
                [ss](W v, uint32_t handle) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = static_cast<cpptype>(v) + *s;                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_rmul_" #suffix,                                               \
                [ss](W v, uint32_t handle) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = static_cast<cpptype>(v) * *s;                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "element_rmod_" #suffix,                                               \
                [ss](W v, uint32_t handle) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = static_cast<cpptype>(v) % *s;                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "series_add_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in series_add_" #suffix \
                        );                                                             \
                    auto result = *sa + *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "series_mul_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in series_mul_" #suffix \
                        );                                                             \
                    auto result = *sa * *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "series_sub_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in series_sub_" #suffix \
                        );                                                             \
                    auto result = *sa - *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "series_div_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in series_div_" #suffix \
                        );                                                             \
                    auto result = *sa / *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "series_mod_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in series_mod_" #suffix \
                        );                                                             \
                    auto result = *sa % *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_gt_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in comparison"          \
                        );                                                             \
                    auto result = *sa > *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_lt_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in comparison"          \
                        );                                                             \
                    auto result = *sa < *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_ge_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in comparison"          \
                        );                                                             \
                    auto result = *sa >= *sb;                                          \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_le_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in comparison"          \
                        );                                                             \
                    auto result = *sa <= *sb;                                          \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_eq_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in comparison"          \
                        );                                                             \
                    auto result = *sa == *sb;                                          \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_ne_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    if (sa->size() != sb->size())                                      \
                        throw std::runtime_error(                                      \
                            "arc panic: series length mismatch in comparison"          \
                        );                                                             \
                    auto result = *sa != *sb;                                          \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_gt_scalar_" #suffix,                                          \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s > static_cast<cpptype>(v);                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_lt_scalar_" #suffix,                                          \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s < static_cast<cpptype>(v);                        \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_ge_scalar_" #suffix,                                          \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s >= static_cast<cpptype>(v);                       \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_le_scalar_" #suffix,                                          \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s <= static_cast<cpptype>(v);                       \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_eq_scalar_" #suffix,                                          \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s == static_cast<cpptype>(v);                       \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "compare_ne_scalar_" #suffix,                                          \
                [ss](uint32_t handle, W v) -> uint32_t {                               \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return 0;                                        \
                    auto result = *s != static_cast<cpptype>(v);                       \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
    }

    BIND_SERIES_OPS(u8, uint8_t, x::telem::UINT8_T)
    BIND_SERIES_OPS(u16, uint16_t, x::telem::UINT16_T)
    BIND_SERIES_OPS(u32, uint32_t, x::telem::UINT32_T)
    BIND_SERIES_OPS(u64, uint64_t, x::telem::UINT64_T)
    BIND_SERIES_OPS(i8, int8_t, x::telem::INT8_T)
    BIND_SERIES_OPS(i16, int16_t, x::telem::INT16_T)
    BIND_SERIES_OPS(i32, int32_t, x::telem::INT32_T)
    BIND_SERIES_OPS(i64, int64_t, x::telem::INT64_T)
    BIND_SERIES_OPS(f32, float, x::telem::FLOAT32_T)
    BIND_SERIES_OPS(f64, double, x::telem::FLOAT64_T)

#undef BIND_SERIES_OPS

    // A str series holds variable-length samples, so it starts with no samples.
    linker
        .func_wrap(
            MODULE_NAME,
            "create_empty_str",
            [ss](uint32_t) -> uint32_t {
                return ss->store(x::telem::Series(x::telem::STRING_T, 0));
            }
        )
        .unwrap();

    // Registers the series operations valid on a bool series: allocation,
    // element access, and indexing. Bool series arise from element-wise
    // comparisons and support no arithmetic.
    BIND_CREATE_EMPTY(bool, x::telem::BOOLEAN_T)
    linker
        .func_wrap(
            MODULE_NAME,
            "set_element_bool",
            [ss](uint32_t handle, uint32_t index, uint32_t value)
                -> wasmtime::Result<uint32_t, wasmtime::Trap> {
                auto *s = ss->get(handle);
                if (s == nullptr) return handle;
                if (auto trap = check_index(index, s->size())) return std::move(*trap);
                s->set(
                    static_cast<int>(index),
                    static_cast<uint8_t>(value != 0 ? 1 : 0)
                );
                return handle;
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "index_bool",
            [ss](uint32_t handle, uint32_t index)
                -> wasmtime::Result<uint32_t, wasmtime::Trap> {
                auto *s = ss->get(handle);
                if (s == nullptr) return uint32_t{0};
                if (auto trap = check_index(index, s->size())) return std::move(*trap);
                return static_cast<uint32_t>(
                    s->at<uint8_t>(static_cast<int>(index)) != 0
                );
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "and",
            [ss](uint32_t a, uint32_t b) -> uint32_t {
                auto *sa = ss->get(a);
                auto *sb = ss->get(b);
                if (sa == nullptr || sb == nullptr) return 0;
                auto result = sa->logical_and(*sb);
                return ss->store(std::move(result));
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "or",
            [ss](uint32_t a, uint32_t b) -> uint32_t {
                auto *sa = ss->get(a);
                auto *sb = ss->get(b);
                if (sa == nullptr || sb == nullptr) return 0;
                auto result = sa->logical_or(*sb);
                return ss->store(std::move(result));
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "and_scalar",
            [ss](uint32_t handle, uint32_t scalar) -> uint32_t {
                auto *s = ss->get(handle);
                if (s == nullptr) return 0;
                auto result = s->logical_and(scalar != 0);
                return ss->store(std::move(result));
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "or_scalar",
            [ss](uint32_t handle, uint32_t scalar) -> uint32_t {
                auto *s = ss->get(handle);
                if (s == nullptr) return 0;
                auto result = s->logical_or(scalar != 0);
                return ss->store(std::move(result));
            }
        )
        .unwrap();

#undef BIND_CREATE_EMPTY

#define BIND_NEGATE(suffix)                                                            \
    linker                                                                             \
        .func_wrap(                                                                    \
            MODULE_NAME,                                                               \
            "negate_" #suffix,                                                         \
            [ss](uint32_t handle) -> uint32_t {                                        \
                auto *s = ss->get(handle);                                             \
                if (s == nullptr) return 0;                                            \
                auto result = -*s;                                                     \
                return ss->store(std::move(result));                                   \
            }                                                                          \
        )                                                                              \
        .unwrap();

    BIND_NEGATE(i8)
    BIND_NEGATE(i16)
    BIND_NEGATE(i32)
    BIND_NEGATE(i64)
    BIND_NEGATE(f32)
    BIND_NEGATE(f64)

#undef BIND_NEGATE

    linker
        .func_wrap(
            MODULE_NAME,
            "not",
            [ss](uint32_t handle) -> uint32_t {
                auto *s = ss->get(handle);
                if (s == nullptr) return 0;
                auto result = s->logical_not();
                return ss->store(std::move(result));
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "len",
            [ss](uint32_t handle) -> int64_t {
                const auto *s = ss->get(handle);
                if (s == nullptr) return 0;
                return static_cast<int64_t>(s->size());
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "index_from_i64",
            [](int64_t index) -> wasmtime::Result<uint32_t, wasmtime::Trap> {
                if (index < INT32_MIN || index > INT32_MAX)
                    return index_overflow(std::to_string(index));
                return static_cast<uint32_t>(index);
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "index_from_u64",
            [](uint64_t index) -> wasmtime::Result<uint32_t, wasmtime::Trap> {
                if (index > static_cast<uint64_t>(INT32_MAX))
                    return index_overflow(std::to_string(index));
                return static_cast<uint32_t>(index);
            }
        )
        .unwrap();
    linker
        .func_wrap(
            MODULE_NAME,
            "slice",
            [ss](uint32_t handle, uint32_t start, uint32_t end) -> uint32_t {
                const auto *src = ss->get(handle);
                if (src == nullptr) return 0;
                const auto first = slice_bound(start, src->size());
                const auto last = slice_bound(end, src->size());
                if (first >= last) return 0;
                const auto slice_len = last - first;
                auto sliced = x::telem::Series(src->data_type(), slice_len);
                const auto density = src->data_type().density();
                std::memcpy(
                    sliced.data(),
                    src->data() + first * density,
                    slice_len * density
                );
                sliced.resize(slice_len);
                return ss->store(std::move(sliced));
            }
        )
        .unwrap();
}

}
