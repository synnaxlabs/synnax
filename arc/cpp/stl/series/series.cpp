// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "wasmtime.hh"

#include "arc/cpp/stl/series/series.h"

namespace arc::stl::series {

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
                [ss](uint32_t handle, uint32_t index, W value) -> uint32_t {           \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return handle;                                   \
                    s->set(static_cast<int>(index), static_cast<cpptype>(value));      \
                    return handle;                                                     \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                MODULE_NAME,                                                           \
                "index_" #suffix,                                                      \
                [ss](uint32_t handle, uint32_t index) -> W {                           \
                    auto *s = ss->get(handle);                                         \
                    if (s == nullptr) return W{};                                      \
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

// BIND_BITWISE_OPS registers the bitwise host functions for one integer
// element type. The commutative reverse forms reuse the forward operators;
// only the shifts compute (scalar op element).
#define BIND_BITWISE_ELEMENT(suffix, cpptype, name, op)                                \
    linker                                                                             \
        .func_wrap(                                                                    \
            MODULE_NAME,                                                               \
            "element_" #name "_" #suffix,                                              \
            [ss](uint32_t handle, typename WasmType<cpptype>::type v) -> uint32_t {    \
                auto *s = ss->get(handle);                                             \
                if (s == nullptr) return 0;                                            \
                auto result = *s op static_cast<cpptype>(v);                           \
                return ss->store(std::move(result));                                   \
            }                                                                          \
        )                                                                              \
        .unwrap();

#define BIND_BITWISE_REVERSE(suffix, cpptype, name, op)                                \
    linker                                                                             \
        .func_wrap(                                                                    \
            MODULE_NAME,                                                               \
            "element_" #name "_" #suffix,                                              \
            [ss](typename WasmType<cpptype>::type v, uint32_t handle) -> uint32_t {    \
                auto *s = ss->get(handle);                                             \
                if (s == nullptr) return 0;                                            \
                auto result = static_cast<cpptype>(v) op * s;                          \
                return ss->store(std::move(result));                                   \
            }                                                                          \
        )                                                                              \
        .unwrap();

#define BIND_BITWISE_SERIES(suffix, name, op)                                          \
    linker                                                                             \
        .func_wrap(                                                                    \
            MODULE_NAME,                                                               \
            "series_" #name "_" #suffix,                                               \
            [ss](uint32_t a, uint32_t b) -> uint32_t {                                 \
                auto *sa = ss->get(a);                                                 \
                auto *sb = ss->get(b);                                                 \
                if (sa == nullptr || sb == nullptr) return 0;                          \
                if (sa->size() != sb->size())                                          \
                    throw std::runtime_error(                                          \
                        "arc panic: series length mismatch in series_" #name           \
                        "_" #suffix                                                    \
                    );                                                                 \
                auto result = *sa op * sb;                                             \
                return ss->store(std::move(result));                                   \
            }                                                                          \
        )                                                                              \
        .unwrap();

#define BIND_BITWISE_OPS(suffix, cpptype)                                              \
    BIND_BITWISE_ELEMENT(suffix, cpptype, band, &)                                     \
    BIND_BITWISE_ELEMENT(suffix, cpptype, bor, |)                                      \
    BIND_BITWISE_ELEMENT(suffix, cpptype, bxor, ^)                                     \
    BIND_BITWISE_ELEMENT(suffix, cpptype, shl, <<)                                     \
    BIND_BITWISE_ELEMENT(suffix, cpptype, shr, >>)                                     \
    BIND_BITWISE_REVERSE(suffix, cpptype, rband, &)                                    \
    BIND_BITWISE_REVERSE(suffix, cpptype, rbor, |)                                     \
    BIND_BITWISE_REVERSE(suffix, cpptype, rbxor, ^)                                    \
    BIND_BITWISE_REVERSE(suffix, cpptype, rshl, <<)                                    \
    BIND_BITWISE_REVERSE(suffix, cpptype, rshr, >>)                                    \
    BIND_BITWISE_SERIES(suffix, band, &)                                               \
    BIND_BITWISE_SERIES(suffix, bor, |)                                                \
    BIND_BITWISE_SERIES(suffix, bxor, ^)                                               \
    BIND_BITWISE_SERIES(suffix, shl, <<)                                               \
    BIND_BITWISE_SERIES(suffix, shr, >>)                                               \
    linker                                                                             \
        .func_wrap(                                                                    \
            MODULE_NAME,                                                               \
            "bnot_" #suffix,                                                           \
            [ss](uint32_t handle) -> uint32_t {                                        \
                auto *s = ss->get(handle);                                             \
                if (s == nullptr) return 0;                                            \
                auto result = ~*s;                                                     \
                return ss->store(std::move(result));                                   \
            }                                                                          \
        )                                                                              \
        .unwrap();

    BIND_BITWISE_OPS(u8, uint8_t)
    BIND_BITWISE_OPS(u16, uint16_t)
    BIND_BITWISE_OPS(u32, uint32_t)
    BIND_BITWISE_OPS(u64, uint64_t)
    BIND_BITWISE_OPS(i8, int8_t)
    BIND_BITWISE_OPS(i16, int16_t)
    BIND_BITWISE_OPS(i32, int32_t)
    BIND_BITWISE_OPS(i64, int64_t)

#undef BIND_BITWISE_OPS
#undef BIND_BITWISE_SERIES
#undef BIND_BITWISE_REVERSE
#undef BIND_BITWISE_ELEMENT

    // Registers the series operations valid on a bool series: allocation,
    // element access, and indexing. Bool series arise from element-wise
    // comparisons and support no arithmetic.
    BIND_CREATE_EMPTY(bool, x::telem::BOOLEAN_T)
    linker
        .func_wrap(
            MODULE_NAME,
            "set_element_bool",
            [ss](uint32_t handle, uint32_t index, uint32_t value) -> uint32_t {
                auto *s = ss->get(handle);
                if (s != nullptr && index < s->size())
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
            [ss](uint32_t handle, uint32_t index) -> uint32_t {
                auto *s = ss->get(handle);
                if (s != nullptr && index < s->size() &&
                    s->at<uint8_t>(static_cast<int>(index)) != 0)
                    return 1;
                return 0;
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
            "slice",
            [ss](uint32_t handle, uint32_t start, uint32_t end) -> uint32_t {
                const auto *src = ss->get(handle);
                if (src == nullptr) return 0;
                const auto src_size = src->size();
                if (start >= src_size || end > src_size || start >= end) return 0;
                const auto slice_len = end - start;
                auto sliced = x::telem::Series(src->data_type(), slice_len);
                const auto density = src->data_type().density();
                std::memcpy(
                    sliced.data(),
                    src->data() + start * density,
                    slice_len * density
                );
                sliced.resize(slice_len);
                return ss->store(std::move(sliced));
            }
        )
        .unwrap();
}

}
