// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "arc/cpp/stl/series/series.h"

namespace arc::stl::series {

void Module::bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) {
    auto ss = this->series_state;

#define BIND_SERIES_OPS(suffix, cpptype, data_type_const)                              \
    {                                                                                  \
        using W = typename WasmType<cpptype>::type;                                    \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "create_empty_" #suffix,                                               \
                [ss](uint32_t length) -> uint32_t {                                    \
                    auto s = telem::Series(                                            \
                        data_type_const,                                               \
                        static_cast<size_t>(length)                                    \
                    );                                                                 \
                    s.resize(length);                                                  \
                    return ss->store(std::move(s));                                    \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
                "series_add_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa + *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "series_mul_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa * *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "series_sub_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa - *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "series_div_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa / *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "series_mod_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa % *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "compare_gt_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa > *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "compare_lt_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa < *sb;                                           \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "compare_ge_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa >= *sb;                                          \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "compare_le_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa <= *sb;                                          \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "compare_eq_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa == *sb;                                          \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
                "compare_ne_" #suffix,                                                 \
                [ss](uint32_t a, uint32_t b) -> uint32_t {                             \
                    auto *sa = ss->get(a);                                             \
                    auto *sb = ss->get(b);                                             \
                    if (sa == nullptr || sb == nullptr) return 0;                      \
                    auto result = *sa != *sb;                                          \
                    return ss->store(std::move(result));                               \
                }                                                                      \
            )                                                                          \
            .unwrap();                                                                 \
        linker                                                                         \
            .func_wrap(                                                                \
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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
                "series",                                                              \
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

    BIND_SERIES_OPS(u8, uint8_t, telem::UINT8_T)
    BIND_SERIES_OPS(u16, uint16_t, telem::UINT16_T)
    BIND_SERIES_OPS(u32, uint32_t, telem::UINT32_T)
    BIND_SERIES_OPS(u64, uint64_t, telem::UINT64_T)
    BIND_SERIES_OPS(i8, int8_t, telem::INT8_T)
    BIND_SERIES_OPS(i16, int16_t, telem::INT16_T)
    BIND_SERIES_OPS(i32, int32_t, telem::INT32_T)
    BIND_SERIES_OPS(i64, int64_t, telem::INT64_T)
    BIND_SERIES_OPS(f32, float, telem::FLOAT32_T)
    BIND_SERIES_OPS(f64, double, telem::FLOAT64_T)

#undef BIND_SERIES_OPS

#define BIND_NEGATE(suffix)                                                            \
    linker                                                                             \
        .func_wrap(                                                                    \
            "series",                                                                  \
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
            "series",
            "not_u8",
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
            "series",
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
            "series",
            "slice",
            [ss](uint32_t handle, uint32_t start, uint32_t end) -> uint32_t {
                const auto *src = ss->get(handle);
                if (src == nullptr) return 0;
                const auto src_size = src->size();
                if (start >= src_size || end > src_size || start >= end) return 0;
                const auto slice_len = end - start;
                auto sliced = telem::Series(src->data_type(), slice_len);
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
