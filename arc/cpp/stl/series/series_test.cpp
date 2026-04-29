// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cstdint>
#include <memory>
#include <sstream>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "arc/cpp/stl/series/series.h"
#include "arc/cpp/stl/series/state.h"
#include "wasmtime.hh"

namespace arc::stl::series {

const char *wasm_type_str(const std::string &suffix) {
    if (suffix == "f32") return "f32";
    if (suffix == "f64") return "f64";
    if (suffix == "u64" || suffix == "i64") return "i64";
    return "i32";
}

bool is_float(const std::string &suffix) {
    return suffix == "f32" || suffix == "f64";
}

std::string build_core_wat(const std::string &suffix) {
    std::string wt = wasm_type_str(suffix);
    std::ostringstream w;
    w << "(module\n";
    w << "  (import \"series\" \"create_empty_" << suffix
      << "\" (func $create_empty (param i32) (result i32)))\n";
    w << "  (import \"series\" \"set_element_" << suffix
      << "\" (func $set_element (param i32 i32 " << wt << ") (result i32)))\n";
    w << "  (import \"series\" \"index_" << suffix
      << "\" (func $index (param i32 i32) (result " << wt << ")))\n";
    w << "  (func (export \"create_empty\") (param i32) (result i32)\n";
    w << "    (call $create_empty (local.get 0)))\n";
    w << "  (func (export \"set_element\") (param i32 i32 " << wt << ") (result i32)\n";
    w << "    (call $set_element (local.get 0) (local.get 1) (local.get 2)))\n";
    w << "  (func (export \"index\") (param i32 i32) (result " << wt << ")\n";
    w << "    (call $index (local.get 0) (local.get 1)))\n";
    w << ")";
    return w.str();
}

std::string build_element_ops_wat(const std::string &suffix) {
    std::string wt = wasm_type_str(suffix);
    std::ostringstream w;
    w << "(module\n";
    w << "  (import \"series\" \"create_empty_" << suffix
      << "\" (func $create_empty (param i32) (result i32)))\n";
    w << "  (import \"series\" \"set_element_" << suffix
      << "\" (func $set_element (param i32 i32 " << wt << ") (result i32)))\n";
    w << "  (import \"series\" \"index_" << suffix
      << "\" (func $index (param i32 i32) (result " << wt << ")))\n";

    const std::vector<std::string> ops =
        {"element_add", "element_mul", "element_sub", "element_div", "element_mod"};
    for (const auto &op: ops) {
        if (is_float(suffix) && op == "element_mod") continue;
        w << "  (import \"series\" \"" << op << "_" << suffix << "\" (func $" << op
          << " (param i32 " << wt << ") (result i32)))\n";
    }

    const std::vector<std::string> rops = {
        "element_radd",
        "element_rmul",
        "element_rsub",
        "element_rdiv",
        "element_rmod"
    };
    for (const auto &op: rops) {
        if (is_float(suffix) && op == "element_rmod") continue;
        w << "  (import \"series\" \"" << op << "_" << suffix << "\" (func $" << op
          << " (param " << wt << " i32) (result i32)))\n";
    }

    w << "  (func (export \"create_empty\") (param i32) (result i32)\n";
    w << "    (call $create_empty (local.get 0)))\n";
    w << "  (func (export \"set_element\") (param i32 i32 " << wt << ") (result i32)\n";
    w << "    (call $set_element (local.get 0) (local.get 1) (local.get 2)))\n";
    w << "  (func (export \"index\") (param i32 i32) (result " << wt << ")\n";
    w << "    (call $index (local.get 0) (local.get 1)))\n";

    for (const auto &op: ops) {
        if (is_float(suffix) && op == "element_mod") continue;
        w << "  (func (export \"" << op << "\") (param i32 " << wt
          << ") (result i32)\n";
        w << "    (call $" << op << " (local.get 0) (local.get 1)))\n";
    }
    for (const auto &op: rops) {
        if (is_float(suffix) && op == "element_rmod") continue;
        w << "  (func (export \"" << op << "\") (param " << wt
          << " i32) (result i32)\n";
        w << "    (call $" << op << " (local.get 0) (local.get 1)))\n";
    }

    w << ")";
    return w.str();
}

std::string build_series_ops_wat(const std::string &suffix) {
    std::string wt = wasm_type_str(suffix);
    std::ostringstream w;
    w << "(module\n";
    w << "  (import \"series\" \"create_empty_" << suffix
      << "\" (func $create_empty (param i32) (result i32)))\n";
    w << "  (import \"series\" \"set_element_" << suffix
      << "\" (func $set_element (param i32 i32 " << wt << ") (result i32)))\n";
    w << "  (import \"series\" \"index_" << suffix
      << "\" (func $index (param i32 i32) (result " << wt << ")))\n";

    const std::vector<std::string> ops =
        {"series_add", "series_mul", "series_sub", "series_div", "series_mod"};
    for (const auto &op: ops) {
        if (is_float(suffix) && op == "series_mod") continue;
        w << "  (import \"series\" \"" << op << "_" << suffix << "\" (func $" << op
          << " (param i32 i32) (result i32)))\n";
    }

    w << "  (func (export \"create_empty\") (param i32) (result i32)\n";
    w << "    (call $create_empty (local.get 0)))\n";
    w << "  (func (export \"set_element\") (param i32 i32 " << wt << ") (result i32)\n";
    w << "    (call $set_element (local.get 0) (local.get 1) (local.get 2)))\n";
    w << "  (func (export \"index\") (param i32 i32) (result " << wt << ")\n";
    w << "    (call $index (local.get 0) (local.get 1)))\n";

    for (const auto &op: ops) {
        if (is_float(suffix) && op == "series_mod") continue;
        w << "  (func (export \"" << op << "\") (param i32 i32) (result i32)\n";
        w << "    (call $" << op << " (local.get 0) (local.get 1)))\n";
    }

    w << ")";
    return w.str();
}

std::string build_compare_wat(const std::string &suffix) {
    std::string wt = wasm_type_str(suffix);
    std::ostringstream w;
    w << "(module\n";
    w << "  (import \"series\" \"create_empty_" << suffix
      << "\" (func $create_empty (param i32) (result i32)))\n";
    w << "  (import \"series\" \"set_element_" << suffix
      << "\" (func $set_element (param i32 i32 " << wt << ") (result i32)))\n";

    const std::vector<std::string> cmp = {
        "compare_gt",
        "compare_lt",
        "compare_ge",
        "compare_le",
        "compare_eq",
        "compare_ne"
    };
    for (const auto &op: cmp) {
        w << "  (import \"series\" \"" << op << "_" << suffix << "\" (func $" << op
          << " (param i32 i32) (result i32)))\n";
    }
    const std::vector<std::string> scmp = {
        "compare_gt_scalar",
        "compare_lt_scalar",
        "compare_ge_scalar",
        "compare_le_scalar",
        "compare_eq_scalar",
        "compare_ne_scalar"
    };
    for (const auto &op: scmp) {
        w << "  (import \"series\" \"" << op << "_" << suffix << "\" (func $" << op
          << " (param i32 " << wt << ") (result i32)))\n";
    }

    w << "  (import \"series\" \"index_u8"
      << "\" (func $index_u8 (param i32 i32) (result i32)))\n";

    w << "  (func (export \"create_empty\") (param i32) (result i32)\n";
    w << "    (call $create_empty (local.get 0)))\n";
    w << "  (func (export \"set_element\") (param i32 i32 " << wt << ") (result i32)\n";
    w << "    (call $set_element (local.get 0) (local.get 1) (local.get 2)))\n";
    w << "  (func (export \"index_u8\") (param i32 i32) (result i32)\n";
    w << "    (call $index_u8 (local.get 0) (local.get 1)))\n";

    for (const auto &op: cmp) {
        w << "  (func (export \"" << op << "\") (param i32 i32) (result i32)\n";
        w << "    (call $" << op << " (local.get 0) (local.get 1)))\n";
    }
    for (const auto &op: scmp) {
        w << "  (func (export \"" << op << "\") (param i32 " << wt
          << ") (result i32)\n";
        w << "    (call $" << op << " (local.get 0) (local.get 1)))\n";
    }

    w << ")";
    return w.str();
}

struct Fixture {
    std::shared_ptr<State> state;
    Module mod;
    wasmtime::Engine engine;
    wasmtime::Store store;
    wasmtime::Linker linker;
    wasmtime::Instance instance;

    Fixture(const std::string &wat):
        state(std::make_shared<State>()),
        mod(state),
        store(engine),
        linker(engine),
        instance(setup(wat)) {}

    wasmtime::Func get(const std::string &name) {
        return std::get<wasmtime::Func>(*instance.get(store, name));
    }

private:
    wasmtime::Instance setup(const std::string &wat) {
        mod.bind_to(linker, store);
        auto wasm_mod = wasmtime::Module::compile(engine, wat).unwrap();
        return linker.instantiate(store, wasm_mod).unwrap();
    }
};

////////////////////// Chunk 1: create_empty, set_element, index //////////////////////

#define TEST_CREATE_EMPTY(suffix, wasm_val_type)                                       \
    TEST(SeriesModule, CreateEmpty_##suffix) {                                         \
        Fixture f(build_core_wat(#suffix));                                            \
        auto result = f.get("create_empty")                                            \
                          .call(f.store, {wasmtime::Val(int32_t{3})})                  \
                          .unwrap();                                                   \
        EXPECT_GT(result[0].i32(), 0);                                                 \
        auto *s = f.state->get(result[0].i32());                                       \
        ASSERT_NE(s, nullptr);                                                         \
        EXPECT_EQ(s->size(), 3);                                                       \
    }

#define TEST_SET_AND_INDEX(suffix, wasm_val_type, make_val, get_val, cpp_val)          \
    TEST(SeriesModule, SetElementAndIndex_##suffix) {                                  \
        Fixture f(build_core_wat(#suffix));                                            \
        auto create = f.get("create_empty");                                           \
        auto handle_r = create.call(f.store, {wasmtime::Val(int32_t{2})}).unwrap();    \
        auto h = handle_r[0].i32();                                                    \
        auto set = f.get("set_element");                                               \
        (void) set                                                                     \
            .call(                                                                     \
                f.store,                                                               \
                {wasmtime::Val(h), wasmtime::Val(int32_t{0}), wasmtime::Val(make_val)} \
            )                                                                          \
            .unwrap();                                                                 \
        auto idx = f.get("index");                                                     \
        auto val_r = idx.call(f.store, {wasmtime::Val(h), wasmtime::Val(int32_t{0})})  \
                         .unwrap();                                                    \
        EXPECT_EQ(val_r[0].get_val(), cpp_val);                                        \
    }

#define TEST_INDEX_NULL_HANDLE(suffix, get_val)                                        \
    TEST(SeriesModule, IndexNullHandle_##suffix) {                                     \
        Fixture f(build_core_wat(#suffix));                                            \
        auto idx = f.get("index");                                                     \
        auto val_r = idx.call(                                                         \
                            f.store,                                                   \
                            {wasmtime::Val(int32_t{999}), wasmtime::Val(int32_t{0})}   \
        )                                                                              \
                         .unwrap();                                                    \
        EXPECT_EQ(val_r[0].get_val(), 0);                                              \
    }

TEST_CREATE_EMPTY(u8, i32)
TEST_CREATE_EMPTY(u16, i32)
TEST_CREATE_EMPTY(u32, i32)
TEST_CREATE_EMPTY(u64, i64)
TEST_CREATE_EMPTY(i8, i32)
TEST_CREATE_EMPTY(i16, i32)
TEST_CREATE_EMPTY(i32, i32)
TEST_CREATE_EMPTY(i64, i64)
TEST_CREATE_EMPTY(f32, f32)
TEST_CREATE_EMPTY(f64, f64)

TEST_SET_AND_INDEX(u8, i32, int32_t{42}, i32, 42)
TEST_SET_AND_INDEX(u16, i32, int32_t{1000}, i32, 1000)
TEST_SET_AND_INDEX(u32, i32, int32_t{70000}, i32, 70000)
TEST_SET_AND_INDEX(u64, i64, int64_t{100000}, i64, 100000)
TEST_SET_AND_INDEX(i8, i32, int32_t{-5}, i32, -5)
TEST_SET_AND_INDEX(i16, i32, int32_t{-300}, i32, -300)
TEST_SET_AND_INDEX(i32, i32, int32_t{-70000}, i32, -70000)
TEST_SET_AND_INDEX(i64, i64, int64_t{-100000}, i64, -100000)
TEST_SET_AND_INDEX(f32, f32, float{3.14f}, f32, 3.14f)
TEST_SET_AND_INDEX(f64, f64, double{2.718}, f64, 2.718)

TEST_INDEX_NULL_HANDLE(u8, i32)
TEST_INDEX_NULL_HANDLE(u16, i32)
TEST_INDEX_NULL_HANDLE(u32, i32)
TEST_INDEX_NULL_HANDLE(u64, i64)
TEST_INDEX_NULL_HANDLE(i8, i32)
TEST_INDEX_NULL_HANDLE(i16, i32)
TEST_INDEX_NULL_HANDLE(i32, i32)
TEST_INDEX_NULL_HANDLE(i64, i64)
TEST_INDEX_NULL_HANDLE(f32, f32)
TEST_INDEX_NULL_HANDLE(f64, f64)

#undef TEST_CREATE_EMPTY
#undef TEST_SET_AND_INDEX
#undef TEST_INDEX_NULL_HANDLE

////////////////////// Chunk 2: element ops (add/sub/mul/div/mod + reverse) ////////////

wasmtime::Val make_wasm_val(const std::string &suffix, double v) {
    if (suffix == "f32") return wasmtime::Val(static_cast<float>(v));
    if (suffix == "f64") return wasmtime::Val(v);
    if (suffix == "u64" || suffix == "i64")
        return wasmtime::Val(static_cast<int64_t>(v));
    return wasmtime::Val(static_cast<int32_t>(v));
}

#define SETUP_ELEMENT_FIXTURE(suffix, v1, v2)                                          \
    Fixture f(build_element_ops_wat(#suffix));                                         \
    auto h_r = f.get("create_empty")                                                   \
                   .call(f.store, {wasmtime::Val(int32_t{2})})                         \
                   .unwrap();                                                          \
    auto h = h_r[0].i32();                                                             \
    (void) f.get("set_element")                                                        \
        .call(                                                                         \
            f.store,                                                                   \
            {wasmtime::Val(h), wasmtime::Val(int32_t{0}), make_wasm_val(#suffix, v1)}  \
        )                                                                              \
        .unwrap();                                                                     \
    (void) f.get("set_element")                                                        \
        .call(                                                                         \
            f.store,                                                                   \
            {wasmtime::Val(h), wasmtime::Val(int32_t{1}), make_wasm_val(#suffix, v2)}  \
        )                                                                              \
        .unwrap();

#define TEST_ELEMENT_OP(suffix, op_name, scalar, expected, get_val)                    \
    TEST(SeriesModule, op_name##_##suffix) {                                           \
        SETUP_ELEMENT_FIXTURE(suffix, 10, 20)                                          \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(h), make_wasm_val(#suffix, scalar)}       \
                          )                                                            \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index")                                                      \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].get_val(), expected);                                         \
    }

#define TEST_ELEMENT_OP_NULL(suffix, op_name)                                          \
    TEST(SeriesModule, op_name##NullHandle_##suffix) {                                 \
        Fixture f(build_element_ops_wat(#suffix));                                     \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(int32_t{999}), make_wasm_val(#suffix, 5)} \
                          )                                                            \
                          .unwrap();                                                   \
        EXPECT_EQ(result[0].i32(), 0);                                                 \
    }

#define TEST_REVERSE_OP(suffix, op_name, scalar, expected, get_val)                    \
    TEST(SeriesModule, op_name##_##suffix) {                                           \
        SETUP_ELEMENT_FIXTURE(suffix, 10, 20)                                          \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {make_wasm_val(#suffix, scalar), wasmtime::Val(h)}       \
                          )                                                            \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index")                                                      \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].get_val(), expected);                                         \
    }

#define TEST_REVERSE_OP_NULL(suffix, op_name)                                          \
    TEST(SeriesModule, op_name##NullHandle_##suffix) {                                 \
        Fixture f(build_element_ops_wat(#suffix));                                     \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {make_wasm_val(#suffix, 5), wasmtime::Val(int32_t{999})} \
                          )                                                            \
                          .unwrap();                                                   \
        EXPECT_EQ(result[0].i32(), 0);                                                 \
    }

#define TEST_ELEMENT_DIV_ZERO(suffix)                                                  \
    TEST(SeriesModule, element_divByZero_##suffix) {                                   \
        SETUP_ELEMENT_FIXTURE(suffix, 10, 20)                                          \
        auto result = f.get("element_div")                                             \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(h), make_wasm_val(#suffix, 0)}            \
                          )                                                            \
                          .unwrap();                                                   \
        EXPECT_EQ(result[0].i32(), 0);                                                 \
    }

#define TEST_ELEMENT_MOD_ZERO(suffix)                                                  \
    TEST(SeriesModule, element_modByZero_##suffix) {                                   \
        SETUP_ELEMENT_FIXTURE(suffix, 10, 20)                                          \
        auto result = f.get("element_mod")                                             \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(h), make_wasm_val(#suffix, 0)}            \
                          )                                                            \
                          .unwrap();                                                   \
        EXPECT_EQ(result[0].i32(), 0);                                                 \
    }

#define ALL_ELEMENT_TESTS(suffix, get_val)                                             \
    TEST_ELEMENT_OP(suffix, element_add, 5, 15, get_val)                               \
    TEST_ELEMENT_OP(suffix, element_sub, 3, 7, get_val)                                \
    TEST_ELEMENT_OP(suffix, element_mul, 2, 20, get_val)                               \
    TEST_ELEMENT_OP(suffix, element_div, 2, 5, get_val)                                \
    TEST_ELEMENT_OP_NULL(suffix, element_add)                                          \
    TEST_ELEMENT_OP_NULL(suffix, element_sub)                                          \
    TEST_ELEMENT_OP_NULL(suffix, element_mul)                                          \
    TEST_ELEMENT_OP_NULL(suffix, element_div)                                          \
    TEST_ELEMENT_DIV_ZERO(suffix)                                                      \
    TEST_REVERSE_OP(suffix, element_radd, 5, 15, get_val)                              \
    TEST_REVERSE_OP(suffix, element_rsub, 30, 20, get_val)                             \
    TEST_REVERSE_OP(suffix, element_rmul, 2, 20, get_val)                              \
    TEST_REVERSE_OP(suffix, element_rdiv, 100, 10, get_val)                            \
    TEST_REVERSE_OP_NULL(suffix, element_radd)                                         \
    TEST_REVERSE_OP_NULL(suffix, element_rsub)                                         \
    TEST_REVERSE_OP_NULL(suffix, element_rmul)                                         \
    TEST_REVERSE_OP_NULL(suffix, element_rdiv)

#define ALL_ELEMENT_TESTS_INT(suffix, get_val)                                         \
    ALL_ELEMENT_TESTS(suffix, get_val)                                                 \
    TEST_ELEMENT_OP(suffix, element_mod, 3, 1, get_val)                                \
    TEST_ELEMENT_OP_NULL(suffix, element_mod)                                          \
    TEST_ELEMENT_MOD_ZERO(suffix)                                                      \
    TEST_REVERSE_OP(suffix, element_rmod, 23, 3, get_val)                              \
    TEST_REVERSE_OP_NULL(suffix, element_rmod)

ALL_ELEMENT_TESTS_INT(u8, i32)
ALL_ELEMENT_TESTS_INT(u16, i32)
ALL_ELEMENT_TESTS_INT(u32, i32)
ALL_ELEMENT_TESTS_INT(u64, i64)
ALL_ELEMENT_TESTS_INT(i8, i32)
ALL_ELEMENT_TESTS_INT(i16, i32)
ALL_ELEMENT_TESTS_INT(i32, i32)
ALL_ELEMENT_TESTS_INT(i64, i64)
ALL_ELEMENT_TESTS(f32, f32)
ALL_ELEMENT_TESTS(f64, f64)

#undef SETUP_ELEMENT_FIXTURE
#undef TEST_ELEMENT_OP
#undef TEST_ELEMENT_OP_NULL
#undef TEST_REVERSE_OP
#undef TEST_REVERSE_OP_NULL
#undef TEST_ELEMENT_DIV_ZERO
#undef TEST_ELEMENT_MOD_ZERO
#undef ALL_ELEMENT_TESTS
#undef ALL_ELEMENT_TESTS_INT

////////////////////// Chunk 3: series-vs-series ops //////////////////////////////////

#define SETUP_TWO_SERIES(suffix, a0, a1, b0, b1)                                       \
    Fixture f(build_series_ops_wat(#suffix));                                          \
    auto ha_r = f.get("create_empty")                                                  \
                    .call(f.store, {wasmtime::Val(int32_t{2})})                        \
                    .unwrap();                                                         \
    auto ha = ha_r[0].i32();                                                           \
    (void) f.get("set_element")                                                        \
        .call(                                                                         \
            f.store,                                                                   \
            {wasmtime::Val(ha), wasmtime::Val(int32_t{0}), make_wasm_val(#suffix, a0)} \
        )                                                                              \
        .unwrap();                                                                     \
    (void) f.get("set_element")                                                        \
        .call(                                                                         \
            f.store,                                                                   \
            {wasmtime::Val(ha), wasmtime::Val(int32_t{1}), make_wasm_val(#suffix, a1)} \
        )                                                                              \
        .unwrap();                                                                     \
    auto hb_r = f.get("create_empty")                                                  \
                    .call(f.store, {wasmtime::Val(int32_t{2})})                        \
                    .unwrap();                                                         \
    auto hb = hb_r[0].i32();                                                           \
    (void) f.get("set_element")                                                        \
        .call(                                                                         \
            f.store,                                                                   \
            {wasmtime::Val(hb), wasmtime::Val(int32_t{0}), make_wasm_val(#suffix, b0)} \
        )                                                                              \
        .unwrap();                                                                     \
    (void) f.get("set_element")                                                        \
        .call(                                                                         \
            f.store,                                                                   \
            {wasmtime::Val(hb), wasmtime::Val(int32_t{1}), make_wasm_val(#suffix, b1)} \
        )                                                                              \
        .unwrap();

#define TEST_SERIES_OP(suffix, op_name, a0, a1, b0, b1, exp0, get_val)                 \
    TEST(SeriesModule, op_name##_##suffix) {                                           \
        SETUP_TWO_SERIES(suffix, a0, a1, b0, b1)                                       \
        auto result = f.get(#op_name)                                                  \
                          .call(f.store, {wasmtime::Val(ha), wasmtime::Val(hb)})       \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index")                                                      \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].get_val(), exp0);                                             \
    }

#define TEST_SERIES_OP_NULL(suffix, op_name)                                           \
    TEST(SeriesModule, op_name##NullHandle_##suffix) {                                 \
        Fixture f(build_series_ops_wat(#suffix));                                      \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(int32_t{999}),                            \
                               wasmtime::Val(int32_t{998})}                            \
                          )                                                            \
                          .unwrap();                                                   \
        EXPECT_EQ(result[0].i32(), 0);                                                 \
    }

#define ALL_SERIES_OPS(suffix, get_val)                                                \
    TEST_SERIES_OP(suffix, series_add, 10, 20, 3, 4, 13, get_val)                      \
    TEST_SERIES_OP(suffix, series_sub, 10, 20, 3, 4, 7, get_val)                       \
    TEST_SERIES_OP(suffix, series_mul, 10, 20, 3, 4, 30, get_val)                      \
    TEST_SERIES_OP(suffix, series_div, 10, 20, 2, 4, 5, get_val)                       \
    TEST_SERIES_OP_NULL(suffix, series_add)                                            \
    TEST_SERIES_OP_NULL(suffix, series_sub)                                            \
    TEST_SERIES_OP_NULL(suffix, series_mul)                                            \
    TEST_SERIES_OP_NULL(suffix, series_div)

#define ALL_SERIES_OPS_INT(suffix, get_val)                                            \
    ALL_SERIES_OPS(suffix, get_val)                                                    \
    TEST_SERIES_OP(suffix, series_mod, 10, 20, 3, 7, 1, get_val)                       \
    TEST_SERIES_OP_NULL(suffix, series_mod)

ALL_SERIES_OPS_INT(u8, i32)
ALL_SERIES_OPS_INT(u16, i32)
ALL_SERIES_OPS_INT(u32, i32)
ALL_SERIES_OPS_INT(u64, i64)
ALL_SERIES_OPS_INT(i8, i32)
ALL_SERIES_OPS_INT(i16, i32)
ALL_SERIES_OPS_INT(i32, i32)
ALL_SERIES_OPS_INT(i64, i64)
ALL_SERIES_OPS(f32, f32)
ALL_SERIES_OPS(f64, f64)

#undef SETUP_TWO_SERIES
#undef TEST_SERIES_OP
#undef TEST_SERIES_OP_NULL
#undef ALL_SERIES_OPS
#undef ALL_SERIES_OPS_INT

////////////////////// Chunk 4: comparisons (series and scalar) ////////////////////////

#define SETUP_CMP_SERIES(suffix, a0, b0)                                               \
    Fixture f(build_compare_wat(#suffix));                                             \
    auto ha_r = f.get("create_empty")                                                  \
                    .call(f.store, {wasmtime::Val(int32_t{1})})                        \
                    .unwrap();                                                         \
    auto ha = ha_r[0].i32();                                                           \
    (void) f.get("set_element")                                                        \
        .call(                                                                         \
            f.store,                                                                   \
            {wasmtime::Val(ha), wasmtime::Val(int32_t{0}), make_wasm_val(#suffix, a0)} \
        )                                                                              \
        .unwrap();                                                                     \
    auto hb_r = f.get("create_empty")                                                  \
                    .call(f.store, {wasmtime::Val(int32_t{1})})                        \
                    .unwrap();                                                         \
    auto hb = hb_r[0].i32();                                                           \
    (void) f.get("set_element")                                                        \
        .call(                                                                         \
            f.store,                                                                   \
            {wasmtime::Val(hb), wasmtime::Val(int32_t{0}), make_wasm_val(#suffix, b0)} \
        )                                                                              \
        .unwrap();

#define TEST_CMP(suffix, op_name, a, b, expected)                                      \
    TEST(SeriesModule, op_name##_##suffix) {                                           \
        SETUP_CMP_SERIES(suffix, a, b)                                                 \
        auto result = f.get(#op_name)                                                  \
                          .call(f.store, {wasmtime::Val(ha), wasmtime::Val(hb)})       \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index_u8")                                                   \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].i32(), expected);                                             \
    }

#define TEST_CMP_NULL(suffix, op_name)                                                 \
    TEST(SeriesModule, op_name##NullHandle_##suffix) {                                 \
        Fixture f(build_compare_wat(#suffix));                                         \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(int32_t{999}),                            \
                               wasmtime::Val(int32_t{998})}                            \
                          )                                                            \
                          .unwrap();                                                   \
        EXPECT_EQ(result[0].i32(), 0);                                                 \
    }

#define TEST_SCMP(suffix, op_name, a, scalar, expected)                                \
    TEST(SeriesModule, op_name##_##suffix) {                                           \
        Fixture f(build_compare_wat(#suffix));                                         \
        auto ha_r = f.get("create_empty")                                              \
                        .call(f.store, {wasmtime::Val(int32_t{1})})                    \
                        .unwrap();                                                     \
        auto ha = ha_r[0].i32();                                                       \
        (void) f.get("set_element")                                                    \
            .call(                                                                     \
                f.store,                                                               \
                {wasmtime::Val(ha),                                                    \
                 wasmtime::Val(int32_t{0}),                                            \
                 make_wasm_val(#suffix, a)}                                            \
            )                                                                          \
            .unwrap();                                                                 \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(ha), make_wasm_val(#suffix, scalar)}      \
                          )                                                            \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index_u8")                                                   \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].i32(), expected);                                             \
    }

#define TEST_SCMP_NULL(suffix, op_name)                                                \
    TEST(SeriesModule, op_name##NullHandle_##suffix) {                                 \
        Fixture f(build_compare_wat(#suffix));                                         \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(int32_t{999}), make_wasm_val(#suffix, 5)} \
                          )                                                            \
                          .unwrap();                                                   \
        EXPECT_EQ(result[0].i32(), 0);                                                 \
    }

#define ALL_CMP_TESTS(suffix)                                                          \
    TEST_CMP(suffix, compare_gt, 10, 5, 1)                                             \
    TEST_CMP(suffix, compare_gt, 5, 10, 0)

// Can't reuse TEST_CMP with same (op, suffix) since test names collide.
// Use separate true/false macros:

#undef ALL_CMP_TESTS

#define TEST_CMP_TRUE(suffix, op_name, a, b)                                           \
    TEST(SeriesModule, op_name##True_##suffix) {                                       \
        SETUP_CMP_SERIES(suffix, a, b)                                                 \
        auto result = f.get(#op_name)                                                  \
                          .call(f.store, {wasmtime::Val(ha), wasmtime::Val(hb)})       \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index_u8")                                                   \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].i32(), 1);                                                    \
    }

#define TEST_CMP_FALSE(suffix, op_name, a, b)                                          \
    TEST(SeriesModule, op_name##False_##suffix) {                                      \
        SETUP_CMP_SERIES(suffix, a, b)                                                 \
        auto result = f.get(#op_name)                                                  \
                          .call(f.store, {wasmtime::Val(ha), wasmtime::Val(hb)})       \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index_u8")                                                   \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].i32(), 0);                                                    \
    }

#define TEST_SCMP_TRUE(suffix, op_name, a, scalar)                                     \
    TEST(SeriesModule, op_name##True_##suffix) {                                       \
        Fixture f(build_compare_wat(#suffix));                                         \
        auto ha_r = f.get("create_empty")                                              \
                        .call(f.store, {wasmtime::Val(int32_t{1})})                    \
                        .unwrap();                                                     \
        auto ha = ha_r[0].i32();                                                       \
        (void) f.get("set_element")                                                    \
            .call(                                                                     \
                f.store,                                                               \
                {wasmtime::Val(ha),                                                    \
                 wasmtime::Val(int32_t{0}),                                            \
                 make_wasm_val(#suffix, a)}                                            \
            )                                                                          \
            .unwrap();                                                                 \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(ha), make_wasm_val(#suffix, scalar)}      \
                          )                                                            \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index_u8")                                                   \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].i32(), 1);                                                    \
    }

#define TEST_SCMP_FALSE(suffix, op_name, a, scalar)                                    \
    TEST(SeriesModule, op_name##False_##suffix) {                                      \
        Fixture f(build_compare_wat(#suffix));                                         \
        auto ha_r = f.get("create_empty")                                              \
                        .call(f.store, {wasmtime::Val(int32_t{1})})                    \
                        .unwrap();                                                     \
        auto ha = ha_r[0].i32();                                                       \
        (void) f.get("set_element")                                                    \
            .call(                                                                     \
                f.store,                                                               \
                {wasmtime::Val(ha),                                                    \
                 wasmtime::Val(int32_t{0}),                                            \
                 make_wasm_val(#suffix, a)}                                            \
            )                                                                          \
            .unwrap();                                                                 \
        auto result = f.get(#op_name)                                                  \
                          .call(                                                       \
                              f.store,                                                 \
                              {wasmtime::Val(ha), make_wasm_val(#suffix, scalar)}      \
                          )                                                            \
                          .unwrap();                                                   \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto val = f.get("index_u8")                                                   \
                       .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})  \
                       .unwrap();                                                      \
        EXPECT_EQ(val[0].i32(), 0);                                                    \
    }

#define ALL_CMP_TESTS(suffix)                                                          \
    TEST_CMP_TRUE(suffix, compare_gt, 10, 5)                                           \
    TEST_CMP_FALSE(suffix, compare_gt, 5, 10)                                          \
    TEST_CMP_TRUE(suffix, compare_lt, 5, 10)                                           \
    TEST_CMP_FALSE(suffix, compare_lt, 10, 5)                                          \
    TEST_CMP_TRUE(suffix, compare_ge, 10, 10)                                          \
    TEST_CMP_FALSE(suffix, compare_ge, 5, 10)                                          \
    TEST_CMP_TRUE(suffix, compare_le, 10, 10)                                          \
    TEST_CMP_FALSE(suffix, compare_le, 10, 5)                                          \
    TEST_CMP_TRUE(suffix, compare_eq, 7, 7)                                            \
    TEST_CMP_FALSE(suffix, compare_eq, 7, 8)                                           \
    TEST_CMP_TRUE(suffix, compare_ne, 7, 8)                                            \
    TEST_CMP_FALSE(suffix, compare_ne, 7, 7)                                           \
    TEST_CMP_NULL(suffix, compare_gt)                                                  \
    TEST_CMP_NULL(suffix, compare_lt)                                                  \
    TEST_CMP_NULL(suffix, compare_ge)                                                  \
    TEST_CMP_NULL(suffix, compare_le)                                                  \
    TEST_CMP_NULL(suffix, compare_eq)                                                  \
    TEST_CMP_NULL(suffix, compare_ne)                                                  \
    TEST_SCMP_TRUE(suffix, compare_gt_scalar, 10, 5)                                   \
    TEST_SCMP_FALSE(suffix, compare_gt_scalar, 5, 10)                                  \
    TEST_SCMP_TRUE(suffix, compare_lt_scalar, 5, 10)                                   \
    TEST_SCMP_FALSE(suffix, compare_lt_scalar, 10, 5)                                  \
    TEST_SCMP_TRUE(suffix, compare_ge_scalar, 10, 10)                                  \
    TEST_SCMP_FALSE(suffix, compare_ge_scalar, 5, 10)                                  \
    TEST_SCMP_TRUE(suffix, compare_le_scalar, 10, 10)                                  \
    TEST_SCMP_FALSE(suffix, compare_le_scalar, 10, 5)                                  \
    TEST_SCMP_TRUE(suffix, compare_eq_scalar, 7, 7)                                    \
    TEST_SCMP_FALSE(suffix, compare_eq_scalar, 7, 8)                                   \
    TEST_SCMP_TRUE(suffix, compare_ne_scalar, 7, 8)                                    \
    TEST_SCMP_FALSE(suffix, compare_ne_scalar, 7, 7)                                   \
    TEST_SCMP_NULL(suffix, compare_gt_scalar)                                          \
    TEST_SCMP_NULL(suffix, compare_lt_scalar)                                          \
    TEST_SCMP_NULL(suffix, compare_ge_scalar)                                          \
    TEST_SCMP_NULL(suffix, compare_le_scalar)                                          \
    TEST_SCMP_NULL(suffix, compare_eq_scalar)                                          \
    TEST_SCMP_NULL(suffix, compare_ne_scalar)

ALL_CMP_TESTS(u8)
ALL_CMP_TESTS(u16)
ALL_CMP_TESTS(u32)
ALL_CMP_TESTS(u64)
ALL_CMP_TESTS(i8)
ALL_CMP_TESTS(i16)
ALL_CMP_TESTS(i32)
ALL_CMP_TESTS(i64)
ALL_CMP_TESTS(f32)
ALL_CMP_TESTS(f64)

#undef SETUP_CMP_SERIES
#undef TEST_CMP_TRUE
#undef TEST_CMP_FALSE
#undef TEST_CMP_NULL
#undef TEST_SCMP_TRUE
#undef TEST_SCMP_FALSE
#undef TEST_SCMP_NULL
#undef ALL_CMP_TESTS

////////////////////// Chunk 5: negate, not_u8, len, slice /////////////////////////////

std::string build_negate_wat(const std::string &suffix) {
    std::string wt = wasm_type_str(suffix);
    std::ostringstream w;
    w << "(module\n";
    w << "  (import \"series\" \"create_empty_" << suffix
      << "\" (func $create_empty (param i32) (result i32)))\n";
    w << "  (import \"series\" \"set_element_" << suffix
      << "\" (func $set_element (param i32 i32 " << wt << ") (result i32)))\n";
    w << "  (import \"series\" \"index_" << suffix
      << "\" (func $index (param i32 i32) (result " << wt << ")))\n";
    w << "  (import \"series\" \"negate_" << suffix
      << "\" (func $negate (param i32) (result i32)))\n";
    w << "  (func (export \"create_empty\") (param i32) (result i32)\n";
    w << "    (call $create_empty (local.get 0)))\n";
    w << "  (func (export \"set_element\") (param i32 i32 " << wt << ") (result i32)\n";
    w << "    (call $set_element (local.get 0) (local.get 1) (local.get 2)))\n";
    w << "  (func (export \"index\") (param i32 i32) (result " << wt << ")\n";
    w << "    (call $index (local.get 0) (local.get 1)))\n";
    w << "  (func (export \"negate\") (param i32) (result i32)\n";
    w << "    (call $negate (local.get 0)))\n";
    w << ")";
    return w.str();
}

#define TEST_NEGATE(suffix, val, expected, get_val)                                    \
    TEST(SeriesModule, Negate_##suffix) {                                              \
        Fixture f(build_negate_wat(#suffix));                                          \
        auto h_r = f.get("create_empty")                                               \
                       .call(f.store, {wasmtime::Val(int32_t{1})})                     \
                       .unwrap();                                                      \
        auto h = h_r[0].i32();                                                         \
        (void) f.get("set_element")                                                    \
            .call(                                                                     \
                f.store,                                                               \
                {wasmtime::Val(h),                                                     \
                 wasmtime::Val(int32_t{0}),                                            \
                 make_wasm_val(#suffix, val)}                                          \
            )                                                                          \
            .unwrap();                                                                 \
        auto result = f.get("negate").call(f.store, {wasmtime::Val(h)}).unwrap();      \
        auto rh = result[0].i32();                                                     \
        EXPECT_GT(rh, 0);                                                              \
        auto v = f.get("index")                                                        \
                     .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})    \
                     .unwrap();                                                        \
        EXPECT_EQ(v[0].get_val(), expected);                                           \
    }

#define TEST_NEGATE_NULL(suffix)                                                       \
    TEST(SeriesModule, NegateNullHandle_##suffix) {                                    \
        Fixture f(build_negate_wat(#suffix));                                          \
        auto result = f.get("negate")                                                  \
                          .call(f.store, {wasmtime::Val(int32_t{999})})                \
                          .unwrap();                                                   \
        EXPECT_EQ(result[0].i32(), 0);                                                 \
    }

TEST_NEGATE(i8, 5, -5, i32)
TEST_NEGATE(i16, 300, -300, i32)
TEST_NEGATE(i32, 70000, -70000, i32)
TEST_NEGATE(i64, 100000, -100000, i64)
TEST_NEGATE(f32, 3.14, -3.14f, f32)
TEST_NEGATE(f64, 2.718, -2.718, f64)

TEST_NEGATE_NULL(i8)
TEST_NEGATE_NULL(i16)
TEST_NEGATE_NULL(i32)
TEST_NEGATE_NULL(i64)
TEST_NEGATE_NULL(f32)
TEST_NEGATE_NULL(f64)

#undef TEST_NEGATE
#undef TEST_NEGATE_NULL

const std::string_view MISC_WAT = R"wat(
(module
  (import "series" "create_empty_u8" (func $create_empty_u8 (param i32) (result i32)))
  (import "series" "set_element_u8" (func $set_element_u8 (param i32 i32 i32) (result i32)))
  (import "series" "index_u8" (func $index_u8 (param i32 i32) (result i32)))
  (import "series" "not_u8" (func $not_u8 (param i32) (result i32)))
  (import "series" "len" (func $len (param i32) (result i64)))
  (import "series" "slice" (func $slice (param i32 i32 i32) (result i32)))
  (import "series" "create_empty_i32" (func $create_empty_i32 (param i32) (result i32)))
  (import "series" "set_element_i32" (func $set_element_i32 (param i32 i32 i32) (result i32)))
  (import "series" "index_i32" (func $index_i32 (param i32 i32) (result i32)))

  (func (export "create_empty_u8") (param i32) (result i32)
    (call $create_empty_u8 (local.get 0)))
  (func (export "set_element_u8") (param i32 i32 i32) (result i32)
    (call $set_element_u8 (local.get 0) (local.get 1) (local.get 2)))
  (func (export "index_u8") (param i32 i32) (result i32)
    (call $index_u8 (local.get 0) (local.get 1)))
  (func (export "not_u8") (param i32) (result i32)
    (call $not_u8 (local.get 0)))
  (func (export "len") (param i32) (result i64)
    (call $len (local.get 0)))
  (func (export "slice") (param i32 i32 i32) (result i32)
    (call $slice (local.get 0) (local.get 1) (local.get 2)))
  (func (export "create_empty_i32") (param i32) (result i32)
    (call $create_empty_i32 (local.get 0)))
  (func (export "set_element_i32") (param i32 i32 i32) (result i32)
    (call $set_element_i32 (local.get 0) (local.get 1) (local.get 2)))
  (func (export "index_i32") (param i32 i32) (result i32)
    (call $index_i32 (local.get 0) (local.get 1)))
)
)wat";

TEST(SeriesModule, NotU8InvertsBooleanSeries) {
    Fixture f{std::string(MISC_WAT)};
    auto h_r = f.get("create_empty_u8")
                   .call(f.store, {wasmtime::Val(int32_t{2})})
                   .unwrap();
    auto h = h_r[0].i32();
    (void) f.get("set_element_u8")
        .call(
            f.store,
            {wasmtime::Val(h), wasmtime::Val(int32_t{0}), wasmtime::Val(int32_t{1})}
        )
        .unwrap();
    (void) f.get("set_element_u8")
        .call(
            f.store,
            {wasmtime::Val(h), wasmtime::Val(int32_t{1}), wasmtime::Val(int32_t{0})}
        )
        .unwrap();
    auto result = f.get("not_u8").call(f.store, {wasmtime::Val(h)}).unwrap();
    auto rh = result[0].i32();
    EXPECT_GT(rh, 0);
    auto v0 = f.get("index_u8")
                  .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})
                  .unwrap();
    auto v1 = f.get("index_u8")
                  .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{1})})
                  .unwrap();
    EXPECT_EQ(v0[0].i32(), 0);
    EXPECT_EQ(v1[0].i32(), 1);
}

TEST(SeriesModule, NotU8NullHandleReturnsZero) {
    Fixture f{std::string(MISC_WAT)};
    auto result = f.get("not_u8").call(f.store, {wasmtime::Val(int32_t{999})}).unwrap();
    EXPECT_EQ(result[0].i32(), 0);
}

TEST(SeriesModule, LenReturnsSeriesLength) {
    Fixture f{std::string(MISC_WAT)};
    auto h_r = f.get("create_empty_u8")
                   .call(f.store, {wasmtime::Val(int32_t{5})})
                   .unwrap();
    auto h = h_r[0].i32();
    auto result = f.get("len").call(f.store, {wasmtime::Val(h)}).unwrap();
    EXPECT_EQ(result[0].i64(), 5);
}

TEST(SeriesModule, LenNullHandleReturnsZero) {
    Fixture f{std::string(MISC_WAT)};
    auto result = f.get("len").call(f.store, {wasmtime::Val(int32_t{999})}).unwrap();
    EXPECT_EQ(result[0].i64(), 0);
}

TEST(SeriesModule, SliceExtractsSubrange) {
    Fixture f{std::string(MISC_WAT)};
    auto h_r = f.get("create_empty_i32")
                   .call(f.store, {wasmtime::Val(int32_t{4})})
                   .unwrap();
    auto h = h_r[0].i32();
    for (int32_t i = 0; i < 4; i++) {
        (void) f.get("set_element_i32")
            .call(
                f.store,
                {wasmtime::Val(h),
                 wasmtime::Val(i),
                 wasmtime::Val(int32_t{(i + 1) * 10})}
            )
            .unwrap();
    }
    auto result = f.get("slice")
                      .call(
                          f.store,
                          {wasmtime::Val(h),
                           wasmtime::Val(int32_t{1}),
                           wasmtime::Val(int32_t{3})}
                      )
                      .unwrap();
    auto rh = result[0].i32();
    EXPECT_GT(rh, 0);
    auto *s = f.state->get(rh);
    ASSERT_NE(s, nullptr);
    EXPECT_EQ(s->size(), 2);
    auto v0 = f.get("index_i32")
                  .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{0})})
                  .unwrap();
    auto v1 = f.get("index_i32")
                  .call(f.store, {wasmtime::Val(rh), wasmtime::Val(int32_t{1})})
                  .unwrap();
    EXPECT_EQ(v0[0].i32(), 20);
    EXPECT_EQ(v1[0].i32(), 30);
}

TEST(SeriesModule, SliceNullHandleReturnsZero) {
    Fixture f{std::string(MISC_WAT)};
    auto result = f.get("slice")
                      .call(
                          f.store,
                          {wasmtime::Val(int32_t{999}),
                           wasmtime::Val(int32_t{0}),
                           wasmtime::Val(int32_t{1})}
                      )
                      .unwrap();
    EXPECT_EQ(result[0].i32(), 0);
}

TEST(SeriesModule, SliceOutOfBoundsReturnsZero) {
    Fixture f{std::string(MISC_WAT)};
    auto h_r = f.get("create_empty_i32")
                   .call(f.store, {wasmtime::Val(int32_t{3})})
                   .unwrap();
    auto h = h_r[0].i32();
    auto result = f.get("slice")
                      .call(
                          f.store,
                          {wasmtime::Val(h),
                           wasmtime::Val(int32_t{0}),
                           wasmtime::Val(int32_t{10})}
                      )
                      .unwrap();
    EXPECT_EQ(result[0].i32(), 0);
}

TEST(SeriesModule, SliceStartGEEndReturnsZero) {
    Fixture f{std::string(MISC_WAT)};
    auto h_r = f.get("create_empty_i32")
                   .call(f.store, {wasmtime::Val(int32_t{3})})
                   .unwrap();
    auto h = h_r[0].i32();
    auto result = f.get("slice")
                      .call(
                          f.store,
                          {wasmtime::Val(h),
                           wasmtime::Val(int32_t{2}),
                           wasmtime::Val(int32_t{1})}
                      )
                      .unwrap();
    EXPECT_EQ(result[0].i32(), 0);
}

}
