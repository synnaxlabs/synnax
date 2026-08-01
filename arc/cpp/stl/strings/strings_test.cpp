// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cmath>
#include <cstring>
#include <limits>
#include <memory>
#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/stl/strings/strings.h"

namespace arc::stl::strings {

/// Offset in WASM linear memory where tests stage format-spec bytes before
/// invoking a format_* host function. Chosen to avoid the literal data
/// placed at offsets 0 and 5.
constexpr uint32_t SPEC_OFFSET = 100;

const std::string_view STR_WAT = R"wat(
(module
  (import "strings" "from_literal" (func $from_lit (param i32 i32) (result i32)))
  (import "strings" "concat" (func $concat (param i32 i32) (result i32)))
  (import "strings" "equal" (func $equal (param i32 i32) (result i32)))
  (import "strings" "len" (func $len (param i32) (result i64)))
  (import "strings" "from_i32" (func $from_i32 (param i32) (result i32)))
  (import "strings" "from_u32" (func $from_u32 (param i32) (result i32)))
  (import "strings" "from_i64" (func $from_i64 (param i64) (result i32)))
  (import "strings" "from_u64" (func $from_u64 (param i64) (result i32)))
  (import "strings" "from_f32" (func $from_f32 (param f32) (result i32)))
  (import "strings" "from_f64" (func $from_f64 (param f64) (result i32)))
  (import "strings" "format_i32" (func $format_i32 (param i32 i32 i32) (result i32)))
  (import "strings" "format_u32" (func $format_u32 (param i32 i32 i32) (result i32)))
  (import "strings" "format_i64" (func $format_i64 (param i64 i32 i32) (result i32)))
  (import "strings" "format_u64" (func $format_u64 (param i64 i32 i32) (result i32)))
  (import "strings" "format_f32" (func $format_f32 (param f32 i32 i32) (result i32)))
  (import "strings" "format_f64" (func $format_f64 (param f64 i32 i32) (result i32)))
  (import "strings" "format_str" (func $format_str (param i32 i32 i32) (result i32)))
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
  (func (export "call_format_i32") (param i32 i32 i32) (result i32)
    (call $format_i32 (local.get 0) (local.get 1) (local.get 2)))
  (func (export "call_format_u32") (param i32 i32 i32) (result i32)
    (call $format_u32 (local.get 0) (local.get 1) (local.get 2)))
  (func (export "call_format_i64") (param i64 i32 i32) (result i32)
    (call $format_i64 (local.get 0) (local.get 1) (local.get 2)))
  (func (export "call_format_u64") (param i64 i32 i32) (result i32)
    (call $format_u64 (local.get 0) (local.get 1) (local.get 2)))
  (func (export "call_format_f32") (param f32 i32 i32) (result i32)
    (call $format_f32 (local.get 0) (local.get 1) (local.get 2)))
  (func (export "call_format_f64") (param f64 i32 i32) (result i32)
    (call $format_f64 (local.get 0) (local.get 1) (local.get 2)))
  (func (export "call_format_str") (param i32 i32 i32) (result i32)
    (call $format_str (local.get 0) (local.get 1) (local.get 2)))
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

    void write_spec(const std::string &spec) {
        const auto mem_span = memory.data(store);
        auto *mem = const_cast<uint8_t *>(mem_span.data());
        std::memcpy(mem + SPEC_OFFSET, spec.data(), spec.size());
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
    EXPECT_EQ(call_from<float>(f, "call_from_f32", std::copysign(0.0f, -1.0f)), "-0");
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
    EXPECT_EQ(call_from<double>(f, "call_from_f64", std::copysign(0.0, -1.0)), "-0");
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

template<typename ValT>
static std::string call_format(
    StrModuleFixture &f,
    const std::string &fn_name,
    ValT v,
    const std::string &spec
) {
    f.write_spec(spec);
    auto fn = f.get_func(fn_name);
    auto result = fn.call(
                        f.store,
                        {wasmtime::Val(v),
                         wasmtime::Val(static_cast<int32_t>(SPEC_OFFSET)),
                         wasmtime::Val(static_cast<int32_t>(spec.size()))}
    )
                      .unwrap();
    return f.state->get(result[0].i32());
}

TEST(StrModule, FormatI32MatchesGoFmtSprintf) {
    StrModuleFixture f;
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 42, "d"), "42");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", -42, "d"), "-42");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 42, "5d"), "   42");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 42, "-5d"), "42   ");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 42, "05d"), "00042");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", -42, "05d"), "-0042");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 42, "+d"), "+42");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 42, " d"), " 42");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 255, "x"), "ff");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 255, "X"), "FF");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 255, "#x"), "0xff");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", -255, "x"), "-ff");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", -255, "#x"), "-0xff");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 5, "b"), "101");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", -5, "b"), "-101");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 5, "#b"), "0b101");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 8, "o"), "10");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 8, "#o"), "010");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 8, "O"), "0o10");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_i32", 65, "c"), "A");
}

TEST(StrModule, FormatU32MatchesGoFmtSprintf) {
    StrModuleFixture f;
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 255, "x"), "ff");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 255, "X"), "FF");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 255, "#x"), "0xff");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 255, "08x"), "000000ff");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 255, ".4x"), "00ff");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 0, "x"), "0");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 0, "#x"), "0x0");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 5, "b"), "101");
    EXPECT_EQ(call_format<int32_t>(f, "call_format_u32", 0, ".0d"), "");
    EXPECT_EQ(
        call_format<int32_t>(
            f,
            "call_format_u32",
            static_cast<int32_t>(0xFFFFFFFFU),
            "x"
        ),
        "ffffffff"
    );
}

TEST(StrModule, FormatI64MatchesGoFmtSprintf) {
    StrModuleFixture f;
    EXPECT_EQ(
        call_format<int64_t>(f, "call_format_i64", 9223372036854775807LL, "d"),
        "9223372036854775807"
    );
    EXPECT_EQ(
        call_format<int64_t>(
            f,
            "call_format_i64",
            std::numeric_limits<int64_t>::min(),
            "d"
        ),
        "-9223372036854775808"
    );
    EXPECT_EQ(call_format<int64_t>(f, "call_format_i64", -1LL, "x"), "-1");
    EXPECT_EQ(
        call_format<int64_t>(f, "call_format_i64", 0xDEADBEEFLL, "x"),
        "deadbeef"
    );
    EXPECT_EQ(
        call_format<int64_t>(f, "call_format_i64", 0xDEADBEEFLL, "X"),
        "DEADBEEF"
    );
}

TEST(StrModule, FormatU64MatchesGoFmtSprintf) {
    StrModuleFixture f;
    EXPECT_EQ(
        call_format<int64_t>(
            f,
            "call_format_u64",
            static_cast<int64_t>(0xFFFFFFFFFFFFFFFFULL),
            "x"
        ),
        "ffffffffffffffff"
    );
    EXPECT_EQ(
        call_format<int64_t>(
            f,
            "call_format_u64",
            static_cast<int64_t>(0xFFFFFFFFFFFFFFFFULL),
            "d"
        ),
        "18446744073709551615"
    );
}

TEST(StrModule, FormatF32MatchesGoFmtSprintf) {
    StrModuleFixture f;
    EXPECT_EQ(call_format<float>(f, "call_format_f32", 3.14f, ".2f"), "3.14");
    EXPECT_EQ(call_format<float>(f, "call_format_f32", 1.5f, ".1f"), "1.5");
    EXPECT_EQ(call_format<float>(f, "call_format_f32", 1.5f, "10.1f"), "       1.5");
    EXPECT_EQ(call_format<float>(f, "call_format_f32", 1.5f, "-10.1f"), "1.5       ");
    EXPECT_EQ(call_format<float>(f, "call_format_f32", 1.5f, "010.1f"), "00000001.5");
    EXPECT_EQ(call_format<float>(f, "call_format_f32", 1.5f, "+.1f"), "+1.5");
    EXPECT_EQ(call_format<float>(f, "call_format_f32", -1.5f, ".1f"), "-1.5");
}

TEST(StrModule, FormatF64MatchesGoFmtSprintf) {
    StrModuleFixture f;
    EXPECT_EQ(call_format<double>(f, "call_format_f64", 3.14159, ".2f"), "3.14");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", 1.5, ".1f"), "1.5");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", 2.71828, ".3f"), "2.718");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", 1.0e6, ".2e"), "1.00e+06");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", 1.0e6, ".2E"), "1.00E+06");
    EXPECT_EQ(
        call_format<double>(f, "call_format_f64", 1234567.89, "g"),
        "1.23456789e+06"
    );
}

TEST(StrModule, FormatF64HandlesNaNAndInfinityWithGoCapitalization) {
    StrModuleFixture f;
    const double nan = std::numeric_limits<double>::quiet_NaN();
    const double pos_inf = std::numeric_limits<double>::infinity();
    const double neg_inf = -pos_inf;
    EXPECT_EQ(call_format<double>(f, "call_format_f64", nan, "f"), "NaN");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", pos_inf, "f"), "+Inf");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", neg_inf, "f"), "-Inf");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", nan, "5f"), "  NaN");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", pos_inf, "6f"), "  +Inf");
    EXPECT_EQ(call_format<double>(f, "call_format_f64", pos_inf, "05f"), " +Inf");
}

static std::string
call_format_str(StrModuleFixture &f, int32_t handle, const std::string &spec) {
    f.write_spec(spec);
    auto fn = f.get_func("call_format_str");
    auto result = fn.call(
                        f.store,
                        {wasmtime::Val(handle),
                         wasmtime::Val(static_cast<int32_t>(SPEC_OFFSET)),
                         wasmtime::Val(static_cast<int32_t>(spec.size()))}
    )
                      .unwrap();
    return f.state->get(result[0].i32());
}

TEST(StrModule, FormatStringWithSVerbPassesThrough) {
    StrModuleFixture f;
    const auto handle = f.state->create("hello");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(handle), "s"), "hello");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(handle), "10s"), "     hello");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(handle), "-10s"), "hello     ");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(handle), ".3s"), "hel");
}

TEST(StrModule, FormatStringWithQVerbQuotesGoStyle) {
    StrModuleFixture f;
    const auto handle = f.state->create("hello");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(handle), "q"), "\"hello\"");

    const auto h_special = f.state->create("a\nb\tc\"d\\e");
    EXPECT_EQ(
        call_format_str(f, static_cast<int32_t>(h_special), "q"),
        R"("a\nb\tc\"d\\e")"
    );

    const auto h_ctrl = f.state->create(std::string("\x01\x7F", 2));
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(h_ctrl), "q"), "\"\\x01\\x7f\"");

    const auto h_empty = f.state->create("");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(h_empty), "q"), "\"\"");
}

TEST(StrModule, FormatStringWithXVerbHexEncodesBytes) {
    StrModuleFixture f;
    const auto handle = f.state->create("abc");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(handle), "x"), "616263");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(handle), "X"), "616263");
    const auto h_full = f.state->create(std::string("\xDE\xAD", 2));
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(h_full), "x"), "dead");
    EXPECT_EQ(call_format_str(f, static_cast<int32_t>(h_full), "X"), "DEAD");
}

TEST(FormatSpec, ParsesAllSpecComponents) {
    EXPECT_EQ(parse_format_spec("d").verb, 'd');
    EXPECT_EQ(parse_format_spec("5d").width, 5);
    EXPECT_EQ(parse_format_spec("-5d").minus, true);
    EXPECT_EQ(parse_format_spec("05d").zero, true);
    EXPECT_EQ(parse_format_spec("+d").plus, true);
    EXPECT_EQ(parse_format_spec(" d").space, true);
    EXPECT_EQ(parse_format_spec("#x").alt, true);
    EXPECT_EQ(parse_format_spec(".2f").precision, 2);
    EXPECT_EQ(parse_format_spec("10.2f").width, 10);
    EXPECT_EQ(parse_format_spec("10.2f").precision, 2);
    const auto full = parse_format_spec("+#-010.4X");
    EXPECT_TRUE(full.plus);
    EXPECT_TRUE(full.alt);
    EXPECT_TRUE(full.minus);
    EXPECT_TRUE(full.zero);
    EXPECT_EQ(full.width, 10);
    EXPECT_EQ(full.precision, 4);
    EXPECT_EQ(full.verb, 'X');
}

TEST(Utf8Encode, EncodesAscii) {
    EXPECT_EQ(utf8_encode(0x41), "A");
    EXPECT_EQ(utf8_encode(0x7F), "\x7F");
}

TEST(Utf8Encode, EncodesMultiByteCodePoints) {
    EXPECT_EQ(utf8_encode(0xF1), "\xC3\xB1"); // ñ
    EXPECT_EQ(utf8_encode(0x2603), "\xE2\x98\x83"); // ☃
    EXPECT_EQ(utf8_encode(0x1F600), "\xF0\x9F\x98\x80"); // 😀
}

TEST(Utf8Encode, InvalidCodePointsBecomeReplacementChar) {
    EXPECT_EQ(utf8_encode(-1), "\xEF\xBF\xBD");
    EXPECT_EQ(utf8_encode(0x110000), "\xEF\xBF\xBD");
    EXPECT_EQ(utf8_encode(0xD800), "\xEF\xBF\xBD");
}

TEST(GoQuote, MatchesGoStrconvQuoteForAscii) {
    EXPECT_EQ(go_quote(""), "\"\"");
    EXPECT_EQ(go_quote("hello"), "\"hello\"");
    EXPECT_EQ(go_quote("a\"b"), "\"a\\\"b\"");
    EXPECT_EQ(go_quote("a\\b"), "\"a\\\\b\"");
    EXPECT_EQ(go_quote("a\nb"), "\"a\\nb\"");
    EXPECT_EQ(go_quote("a\tb"), "\"a\\tb\"");
    EXPECT_EQ(go_quote(std::string("\x00", 1)), "\"\\x00\"");
    EXPECT_EQ(go_quote(std::string("\x1F", 1)), "\"\\x1f\"");
    EXPECT_EQ(go_quote(std::string("\x7F", 1)), "\"\\x7f\"");
}
}
