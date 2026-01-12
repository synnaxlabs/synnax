// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <random>
#include <string>

#include "gtest/gtest.h"

#include "client/cpp/synnax.h"
#include "client/cpp/testutil/testutil.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/test/test.h"

#include "arc/cpp/runtime/wasm/bindings.h"
#include "arc/cpp/runtime/wasm/module.h"

using namespace arc::runtime::wasm;

namespace {
std::mt19937 gen_rand = random_generator("Module Tests");

std::string random_name(const std::string &prefix) {
    std::uniform_int_distribution<> dis(10000, 99999);
    return prefix + "_" + std::to_string(dis(gen_rand));
}

/// @brief Compiles an Arc program via the Synnax client.
arc::module::Module
compile_arc(const synnax::Synnax &client, const std::string &source) {
    synnax::arc::Arc arc;
    arc.name = random_name("test_arc");
    arc.text.raw = source;
    if (const auto create_err = client.arcs.create(arc))
        throw std::runtime_error("Failed to create arc: " + create_err.message());

    synnax::arc::RetrieveOptions opts;
    opts.compile = true;
    auto [compiled, err] = client.arcs.retrieve_by_key(arc.key, opts);
    if (err) throw std::runtime_error("Failed to compile arc: " + err.message());
    if (!compiled.module.has_value())
        throw std::runtime_error("Compiled arc has no module");
    return *compiled.module;
}
}

/// @brief Module::open returns error for empty WASM bytes.
TEST(ModuleOpenTest, ReturnsErrorForEmptyWasmBytes) {
    arc::module::Module mod;
    mod.wasm = {};
    const ModuleConfig cfg{.module = mod};
    const auto [module, err] = Module::open(cfg);
    ASSERT_TRUE(err.matches(x::errors::VALIDATION));
    ASSERT_NE(err.message().find("empty"), std::string::npos);
}

/// @brief Module::open returns error for invalid WASM bytes.
TEST(ModuleOpenTest, ReturnsErrorForInvalidWasmBytes) {
    arc::module::Module mod;
    mod.wasm = {0x00, 0x01, 0x02, 0x03};
    const ModuleConfig cfg{.module = mod};
    const auto [module, err] = Module::open(cfg);
    ASSERT_TRUE(err.matches(x::errors::VALIDATION));
    ASSERT_NE(err.message().find("compile"), std::string::npos);
}

/// @brief Module::open succeeds with valid compiled module.
TEST(ModuleOpenTest, SucceedsWithValidModule) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2
}
)" + ch.name + " -> double{}";

    const auto mod = compile_arc(client, source);
    ASSERT_FALSE(mod.wasm.empty());

    const ModuleConfig cfg{.module = mod};
    const auto module = ASSERT_NIL_P(Module::open(cfg));
    ASSERT_NE(module, nullptr);
}

/// @brief Module::func returns NOT_FOUND for non-existent export.
TEST(ModuleFuncTest, ReturnsNotFoundForMissingExport) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2
}
)" + ch.name + " -> double{}";

    const auto mod = compile_arc(client, source);
    const ModuleConfig cfg{.module = mod};
    auto module = ASSERT_NIL_P(Module::open(cfg));

    auto [func, func_err] = module->func("nonexistent");
    ASSERT_TRUE(func_err.matches(x::errors::NOT_FOUND));
}

/// @brief Module::func returns valid function for existing export.
TEST(ModuleFuncTest, ReturnsValidFunctionForExistingExport) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2
}
)" + ch.name + " -> double{}";

    const auto mod = compile_arc(client, source);
    const ModuleConfig cfg{.module = mod};
    auto module = ASSERT_NIL_P(Module::open(cfg));
    ASSERT_NIL_P(module->func("double"));
}

/// @brief Function::call executes and returns results.
TEST(FunctionCallTest, ExecutesFunctionAndReturnsResults) {
    const auto client = new_test_client();
    const auto ch = ASSERT_NIL_P(
        client.channels.create(random_name("input"), x::telem::FLOAT32_T, true)
    );

    const std::string source = R"(
func double(val f32) f32 {
    return val * 2
}
)" + ch.name + " -> double{}";

    const auto mod = compile_arc(client, source);
    const ModuleConfig cfg{.module = mod};
    auto module = ASSERT_NIL_P(Module::open(cfg));
    auto func = ASSERT_NIL_P(module->func("double"));

    std::vector<x::telem::SampleValue> params = {5.0f};
    auto results = ASSERT_NIL_P(func.call(params));
    ASSERT_EQ(results.size(), 1);
    ASSERT_TRUE(results[0].changed);
    EXPECT_FLOAT_EQ(std::get<float>(results[0].value), 10.0f);
}

/// @brief sample_to_wasm converts integer types correctly.
TEST(SampleToWasmTest, ConvertsIntegerTypes) {
    const auto i32_val = sample_to_wasm(
        x::telem::SampleValue(static_cast<int32_t>(42))
    );
    EXPECT_EQ(i32_val.i32(), 42);

    const auto i64_val = sample_to_wasm(
        x::telem::SampleValue(static_cast<int64_t>(123456789))
    );
    EXPECT_EQ(i64_val.i64(), 123456789);

    const auto u8_val = sample_to_wasm(
        x::telem::SampleValue(static_cast<uint8_t>(255))
    );
    EXPECT_EQ(i32_val.i32(), 42);

    const auto u64_val = sample_to_wasm(
        x::telem::SampleValue(static_cast<uint64_t>(999))
    );
    EXPECT_EQ(u64_val.i64(), 999);
}

/// @brief sample_to_wasm converts float types correctly.
TEST(SampleToWasmTest, ConvertsFloatTypes) {
    const auto f32_val = sample_to_wasm(x::telem::SampleValue(3.14f));
    EXPECT_FLOAT_EQ(f32_val.f32(), 3.14f);

    const auto f64_val = sample_to_wasm(x::telem::SampleValue(2.71828));
    EXPECT_DOUBLE_EQ(f64_val.f64(), 2.71828);
}

/// @brief sample_to_wasm converts timestamp correctly.
TEST(SampleToWasmTest, ConvertsTimestamp) {
    const x::telem::TimeStamp ts(1000000000);
    const auto val = sample_to_wasm(x::telem::SampleValue(ts));
    EXPECT_EQ(val.i64(), 1000000000);
}

/// @brief sample_from_wasm converts integer types correctly.
TEST(SampleFromWasmTest, ConvertsIntegerTypes) {
    wasmtime::Val i32_wasm(static_cast<int32_t>(42));
    const auto i32_sample = sample_from_wasm(
        i32_wasm,
        arc::types::Type{.kind = arc::types::Kind::I32}
    );
    EXPECT_EQ(std::get<int32_t>(i32_sample), 42);

    wasmtime::Val i64_wasm(static_cast<int64_t>(123456789));
    const auto i64_sample = sample_from_wasm(
        i64_wasm,
        arc::types::Type{.kind = arc::types::Kind::I64}
    );
    EXPECT_EQ(std::get<int64_t>(i64_sample), 123456789);

    wasmtime::Val u8_wasm(static_cast<int32_t>(255));
    const auto u8_sample = sample_from_wasm(
        u8_wasm,
        arc::types::Type{.kind = arc::types::Kind::U8}
    );
    EXPECT_EQ(std::get<uint8_t>(u8_sample), 255);

    wasmtime::Val u64_wasm(static_cast<int64_t>(999));
    const auto u64_sample = sample_from_wasm(
        u64_wasm,
        arc::types::Type{.kind = arc::types::Kind::U64}
    );
    EXPECT_EQ(std::get<uint64_t>(u64_sample), 999);
}

/// @brief sample_from_wasm converts float types correctly.
TEST(SampleFromWasmTest, ConvertsFloatTypes) {
    wasmtime::Val f32_wasm(3.14f);
    const auto f32_sample = sample_from_wasm(
        f32_wasm,
        arc::types::Type{.kind = arc::types::Kind::F32}
    );
    EXPECT_FLOAT_EQ(std::get<float>(f32_sample), 3.14f);

    wasmtime::Val f64_wasm(2.71828);
    const auto f64_sample = sample_from_wasm(
        f64_wasm,
        arc::types::Type{.kind = arc::types::Kind::F64}
    );
    EXPECT_DOUBLE_EQ(std::get<double>(f64_sample), 2.71828);
}

/// @brief sample_from_wasm converts timestamp correctly.
TEST(SampleFromWasmTest, ConvertsTimestamp) {
    wasmtime::Val ts_wasm(static_cast<int64_t>(1000000000));
    arc::types::Dimensions dims{.time = 1};
    arc::types::Unit ns_unit{.dimensions = dims, .scale = 1.0, .name = "ns"};
    arc::types::Type ts_type{.kind = arc::types::Kind::I64, .unit = ns_unit};
    const auto ts_sample = sample_from_wasm(ts_wasm, ts_type);
    EXPECT_EQ(std::get<x::telem::TimeStamp>(ts_sample).nanoseconds(), 1000000000);
}

/// @brief sample_from_bits converts integer types correctly.
TEST(SampleFromBitsTest, ConvertsIntegerTypes) {
    const auto i32_sample = sample_from_bits(
        42,
        arc::types::Type{.kind = arc::types::Kind::I32}
    );
    EXPECT_EQ(std::get<int32_t>(i32_sample), 42);

    const auto i64_sample = sample_from_bits(
        123456789,
        arc::types::Type{.kind = arc::types::Kind::I64}
    );
    EXPECT_EQ(std::get<int64_t>(i64_sample), 123456789);

    const auto u8_sample = sample_from_bits(
        255,
        arc::types::Type{.kind = arc::types::Kind::U8}
    );
    EXPECT_EQ(std::get<uint8_t>(u8_sample), 255);

    const auto u64_sample = sample_from_bits(
        999,
        arc::types::Type{.kind = arc::types::Kind::U64}
    );
    EXPECT_EQ(std::get<uint64_t>(u64_sample), 999);
}

/// @brief sample_from_bits converts float types correctly via bit reinterpretation.
TEST(SampleFromBitsTest, ConvertsFloatTypes) {
    float f32_val = 3.14f;
    uint32_t f32_bits;
    memcpy(&f32_bits, &f32_val, sizeof(float));
    const auto f32_sample = sample_from_bits(
        f32_bits,
        arc::types::Type{.kind = arc::types::Kind::F32}
    );
    EXPECT_FLOAT_EQ(std::get<float>(f32_sample), 3.14f);

    double f64_val = 2.71828;
    uint64_t f64_bits;
    memcpy(&f64_bits, &f64_val, sizeof(double));
    const auto f64_sample = sample_from_bits(
        f64_bits,
        arc::types::Type{.kind = arc::types::Kind::F64}
    );
    EXPECT_DOUBLE_EQ(std::get<double>(f64_sample), 2.71828);
}

/// @brief sample_from_bits handles timestamp special case.
TEST(SampleFromBitsTest, HandlesTimestamp) {
    arc::types::Dimensions dims{.time = 1};
    arc::types::Unit ns_unit{.dimensions = dims, .scale = 1.0, .name = "ns"};
    arc::types::Type ts_type{.kind = arc::types::Kind::I64, .unit = ns_unit};
    const auto ts_sample = sample_from_bits(1000000000, ts_type);
    EXPECT_EQ(std::get<x::telem::TimeStamp>(ts_sample).nanoseconds(), 1000000000);
}
