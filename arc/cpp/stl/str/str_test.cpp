// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <limits>
#include <memory>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "arc/cpp/stl/str/state.h"
#include "arc/cpp/stl/str/str.h"
#include "wasmtime.hh"

namespace arc::stl::str {
const std::string_view STR_WAT = R"wat(
(module
  (import "string" "from_literal" (func $from_lit (param i32 i32) (result i32)))
  (import "string" "concat" (func $concat (param i32 i32) (result i32)))
  (import "string" "equal" (func $equal (param i32 i32) (result i32)))
  (import "string" "len" (func $len (param i32) (result i64)))
  (import "string" "from_i32" (func $from_i32 (param i32) (result i32)))
  (import "string" "from_u32" (func $from_u32 (param i32) (result i32)))
  (import "string" "from_i64" (func $from_i64 (param i64) (result i32)))
  (import "string" "from_u64" (func $from_u64 (param i64) (result i32)))
  (import "string" "from_f32" (func $from_f32 (param f32) (result i32)))
  (import "string" "from_f64" (func $from_f64 (param f64) (result i32)))
  (memory (export "memory") 1)
  (data (i32.const 0) "hello")
  (data (i32.const 5) " world")
  (func (export "make_hello") (result i32)
    (call $from_lit (i32.const 0) (i32.const 5)))
  (func (export "make_world") (result i32)
    (call $from_lit (i32.const 5) (i32.const 6)))
  (func (export "concat_handles") (param i32 i32) (result i32)
    (call $concat (local.get 0) (local.get 1)))
  (func (export "equal_handles") (param i32 i32) (result i32)
    (call $equal (local.get 0) (local.get 1)))
  (func (export "len_handle") (param i32) (result i64)
    (call $len (local.get 0)))
  (func (export "from_oob") (result i32)
    (call $from_lit (i32.const 65530) (i32.const 100)))
  (func (export "call_from_i32") (param i32) (result i32)
    (call $from_i32 (local.get 0)))
  (func (export "call_from_u32") (param i32) (result i32)
    (call $from_u32 (local.get 0)))
  (func (export "call_from_i64") (param i64) (result i32)
    (call $from_i64 (local.get 0)))
  (func (export "call_from_u64") (param i64) (result i32)
    (call $from_u64 (local.get 0)))
  (func (export "call_from_f32") (param f32) (result i32)
    (call $from_f32 (local.get 0)))
  (func (export "call_from_f64") (param f64) (result i32)
    (call $from_f64 (local.get 0)))
)
)wat";

struct StrModuleFixture {
    std::shared_ptr<State> state;
    Module mod;
    wasmtime::Engine engine;
    wasmtime::Store store;
    wasmtime::Linker linker;
    wasmtime::Instance instance;
    wasmtime::Memory memory;

    StrModuleFixture():
        state(std::make_shared<State>()),
        mod(state),
        store(engine),
        linker(engine),
        instance(setup_instance()),
        memory(std::get<wasmtime::Memory>(*instance.get(store, "memory"))) {
        mod.set_wasm_context(&store, &memory);
    }

    wasmtime::Func get_func(const std::string &name) {
        return std::get<wasmtime::Func>(*instance.get(store, name));
    }

private:
    wasmtime::Instance setup_instance() {
        mod.bind_to(linker, store);
        auto wasm_mod = wasmtime::Module::compile(engine, STR_WAT).unwrap();
        return linker.instantiate(store, wasm_mod).unwrap();
    }
};

TEST(StrModule, FromLiteralCreatesStringFromWasmMemory) {
    StrModuleFixture f;
    auto make_hello = f.get_func("make_hello");
    auto result = make_hello.call(f.store, {}).unwrap();
    const auto handle = result[0].i32();
    EXPECT_GT(handle, 0);
    EXPECT_EQ(f.state->get(handle), "hello");
}

TEST(StrModule, FromLiteralReturnsZeroForOutOfBounds) {
    StrModuleFixture f;
    auto from_oob = f.get_func("from_oob");
    auto result = from_oob.call(f.store, {}).unwrap();
    EXPECT_EQ(result[0].i32(), 0);
}

TEST(StrModule, ConcatCombinesTwoStrings) {
    StrModuleFixture f;
    auto h1_result = f.get_func("make_hello").call(f.store, {}).unwrap();
    auto h2_result = f.get_func("make_world").call(f.store, {}).unwrap();
    const auto h1 = h1_result[0].i32();
    const auto h2 = h2_result[0].i32();

    auto concat_fn = f.get_func("concat_handles");
    auto result = concat_fn.call(f.store, {wasmtime::Val(h1), wasmtime::Val(h2)})
                      .unwrap();
    const auto concat_handle = result[0].i32();
    EXPECT_EQ(f.state->get(concat_handle), "hello world");
}

TEST(StrModule, EqualReturnsTrueForIdenticalStrings) {
    StrModuleFixture f;
    auto h1_result = f.get_func("make_hello").call(f.store, {}).unwrap();
    auto h2_result = f.get_func("make_hello").call(f.store, {}).unwrap();
    const auto h1 = h1_result[0].i32();
    const auto h2 = h2_result[0].i32();

    auto equal_fn = f.get_func("equal_handles");
    auto result = equal_fn.call(f.store, {wasmtime::Val(h1), wasmtime::Val(h2)})
                      .unwrap();
    EXPECT_EQ(result[0].i32(), 1);
}

TEST(StrModule, EqualReturnsFalseForDifferentStrings) {
    StrModuleFixture f;
    auto h1_result = f.get_func("make_hello").call(f.store, {}).unwrap();
    auto h2_result = f.get_func("make_world").call(f.store, {}).unwrap();
    const auto h1 = h1_result[0].i32();
    const auto h2 = h2_result[0].i32();

    auto equal_fn = f.get_func("equal_handles");
    auto result = equal_fn.call(f.store, {wasmtime::Val(h1), wasmtime::Val(h2)})
                      .unwrap();
    EXPECT_EQ(result[0].i32(), 0);
}

TEST(StrModule, EqualReturnsFalseForInvalidHandle) {
    StrModuleFixture f;
    auto h1_result = f.get_func("make_hello").call(f.store, {}).unwrap();
    const auto h1 = h1_result[0].i32();

    auto equal_fn = f.get_func("equal_handles");
    auto result = equal_fn
                      .call(f.store, {wasmtime::Val(h1), wasmtime::Val(int32_t{999})})
                      .unwrap();
    EXPECT_EQ(result[0].i32(), 0);
}

TEST(StrModule, LenReturnsCorrectLength) {
    StrModuleFixture f;
    auto h_result = f.get_func("make_hello").call(f.store, {}).unwrap();
    const auto h = h_result[0].i32();

    auto len_fn = f.get_func("len_handle");
    auto result = len_fn.call(f.store, {wasmtime::Val(h)}).unwrap();
    EXPECT_EQ(result[0].i64(), 5);
}

template<typename ValT>
static std::string call_from(StrModuleFixture &f, const std::string &fn_name, ValT v) {
    auto fn = f.get_func(fn_name);
    auto result = fn.call(f.store, {wasmtime::Val(v)}).unwrap();
    return f.state->get(result[0].i32());
}

TEST(StrModule, FromI32FormatsSignedIntegers) {
    StrModuleFixture f;
    EXPECT_EQ(call_from<int32_t>(f, "call_from_i32", 0), "0");
    EXPECT_EQ(call_from<int32_t>(f, "call_from_i32", 42), "42");
    EXPECT_EQ(call_from<int32_t>(f, "call_from_i32", -42), "-42");
    EXPECT_EQ(
        call_from<int32_t>(f, "call_from_i32", std::numeric_limits<int32_t>::max()),
        "2147483647"
    );
    EXPECT_EQ(
        call_from<int32_t>(f, "call_from_i32", std::numeric_limits<int32_t>::min()),
        "-2147483648"
    );
}

TEST(StrModule, FromU32FormatsUnsignedIntegers) {
    StrModuleFixture f;
    EXPECT_EQ(call_from<int32_t>(f, "call_from_u32", 0), "0");
    EXPECT_EQ(call_from<int32_t>(f, "call_from_u32", 255), "255");
    EXPECT_EQ(
        call_from<int32_t>(
            f,
            "call_from_u32",
            static_cast<int32_t>(static_cast<uint32_t>(4000000000U))
        ),
        "4000000000"
    );
    EXPECT_EQ(
        call_from<int32_t>(f, "call_from_u32", static_cast<int32_t>(0xFFFFFFFFU)),
        "4294967295"
    );
}

TEST(StrModule, FromI64FormatsSignedIntegers) {
    StrModuleFixture f;
    EXPECT_EQ(call_from<int64_t>(f, "call_from_i64", 0), "0");
    EXPECT_EQ(call_from<int64_t>(f, "call_from_i64", 42), "42");
    EXPECT_EQ(call_from<int64_t>(f, "call_from_i64", -42), "-42");
    EXPECT_EQ(
        call_from<int64_t>(f, "call_from_i64", std::numeric_limits<int64_t>::max()),
        "9223372036854775807"
    );
    EXPECT_EQ(
        call_from<int64_t>(f, "call_from_i64", std::numeric_limits<int64_t>::min()),
        "-9223372036854775808"
    );
}

TEST(StrModule, FromU64FormatsUnsignedIntegers) {
    StrModuleFixture f;
    EXPECT_EQ(call_from<int64_t>(f, "call_from_u64", 0), "0");
    EXPECT_EQ(call_from<int64_t>(f, "call_from_u64", 42), "42");
    EXPECT_EQ(
        call_from<int64_t>(
            f,
            "call_from_u64",
            static_cast<int64_t>(static_cast<uint64_t>(0xFFFFFFFFFFFFFFFFULL))
        ),
        "18446744073709551615"
    );
}

TEST(StrModule, FromF32FormatsShortestRoundTrip) {
    StrModuleFixture f;
    EXPECT_EQ(call_from<float>(f, "call_from_f32", 3.1f), "3.1");
    EXPECT_EQ(call_from<float>(f, "call_from_f32", 0.1f), "0.1");
    EXPECT_EQ(call_from<float>(f, "call_from_f32", 1.0f), "1");
    EXPECT_EQ(call_from<float>(f, "call_from_f32", 100.0f), "100");
    EXPECT_EQ(call_from<float>(f, "call_from_f32", -2.5f), "-2.5");
    EXPECT_EQ(call_from<float>(f, "call_from_f32", 42.5f), "42.5");
}

TEST(StrModule, FromF64FormatsShortestRoundTrip) {
    StrModuleFixture f;
    EXPECT_EQ(call_from<double>(f, "call_from_f64", 3.1), "3.1");
    EXPECT_EQ(call_from<double>(f, "call_from_f64", 3.14), "3.14");
    EXPECT_EQ(call_from<double>(f, "call_from_f64", 1.0), "1");
    EXPECT_EQ(call_from<double>(f, "call_from_f64", -2.5), "-2.5");
    EXPECT_EQ(
        call_from<double>(f, "call_from_f64", 0.1234567890123456),
        "0.1234567890123456"
    );
}

TEST(StrModule, FromF64HandlesNaNAndInfinityWithGoCapitalization) {
    StrModuleFixture f;
    EXPECT_EQ(
        call_from<double>(f, "call_from_f64", std::numeric_limits<double>::quiet_NaN()),
        "NaN"
    );
    EXPECT_EQ(
        call_from<double>(f, "call_from_f64", std::numeric_limits<double>::infinity()),
        "+Inf"
    );
    EXPECT_EQ(
        call_from<double>(f, "call_from_f64", -std::numeric_limits<double>::infinity()),
        "-Inf"
    );
}

TEST(StrModule, FromF32HandlesNaNAndInfinityWithGoCapitalization) {
    StrModuleFixture f;
    EXPECT_EQ(
        call_from<float>(f, "call_from_f32", std::numeric_limits<float>::quiet_NaN()),
        "NaN"
    );
    EXPECT_EQ(
        call_from<float>(f, "call_from_f32", std::numeric_limits<float>::infinity()),
        "+Inf"
    );
    EXPECT_EQ(
        call_from<float>(f, "call_from_f32", -std::numeric_limits<float>::infinity()),
        "-Inf"
    );
}
}
