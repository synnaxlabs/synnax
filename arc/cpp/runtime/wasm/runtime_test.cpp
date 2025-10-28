// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/wasm/runtime.h"

#include "gtest/gtest.h"
#include "x/cpp/xtest/xtest.h"

class RuntimeTest : public ::testing::Test {
protected:
    void SetUp() override {
        // Initialize WAMR runtime before each test
        auto err = arc::Runtime::initialize_runtime();
        ASSERT_NIL(err);
    }

    void TearDown() override {
        // Cleanup after each test
        arc::Runtime::destroy_runtime();
    }
};

TEST_F(RuntimeTest, InitializeRuntime) {
    // Runtime should be initialized by SetUp()
    // No specific assertions needed - if we get here, initialization succeeded
}

TEST_F(RuntimeTest, ConstructAndDestroy) {
    arc::Runtime rt;
    EXPECT_FALSE(rt.is_ready());
    // Destructor will be called automatically
}

TEST_F(RuntimeTest, MoveConstructor) {
    arc::Runtime rt1;
    arc::Runtime rt2(std::move(rt1));

    // rt1 should be moved-from (not ready)
    EXPECT_FALSE(rt1.is_ready());
}

TEST_F(RuntimeTest, MoveAssignment) {
    arc::Runtime rt1;
    arc::Runtime rt2;

    rt2 = std::move(rt1);

    // rt1 should be moved-from
    EXPECT_FALSE(rt1.is_ready());
}

TEST_F(RuntimeTest, LoadModuleWithoutInitialization) {
    // Destroy runtime to test error case
    arc::Runtime::destroy_runtime();

    arc::Runtime rt;
    std::vector<uint8_t> fake_bytes = {0x00, 0x61, 0x73, 0x6d};  // WASM magic

    auto err = rt.load_aot_module(fake_bytes);
    EXPECT_TRUE(err.matches(xerrors::Error("arc.runtime.not_initialized")));

    // Re-initialize for TearDown
    arc::Runtime::initialize_runtime();
}

TEST_F(RuntimeTest, InstantiateWithoutModule) {
    arc::Runtime rt;

    auto err = rt.instantiate(64 * 1024, 0);
    EXPECT_TRUE(err.matches(xerrors::Error("arc.runtime.no_module")));
}

TEST_F(RuntimeTest, FindFunctionWithoutInstantiation) {
    arc::Runtime rt;

    auto [func, err] = rt.find_function("main");
    EXPECT_TRUE(err.matches(xerrors::Error("arc.runtime.not_instantiated")));
    EXPECT_EQ(func, nullptr);
}

TEST_F(RuntimeTest, CallFunctionWithoutReady) {
    arc::Runtime rt;

    std::array<arc::WasmValue, 1> args = {arc::WasmValue(42)};
    std::array<arc::WasmValue, 1> results;

    auto err = rt.call_function(nullptr, args, results);
    EXPECT_TRUE(err.matches(xerrors::Error("arc.runtime.not_ready")));
}

TEST_F(RuntimeTest, WasmValueTypes) {
    // Test WasmValue construction
    arc::WasmValue v_i32(static_cast<int32_t>(42));
    EXPECT_EQ(v_i32.kind, arc::WasmValue::Kind::I32);
    EXPECT_EQ(v_i32.i32, 42);

    arc::WasmValue v_i64(static_cast<int64_t>(123456789));
    EXPECT_EQ(v_i64.kind, arc::WasmValue::Kind::I64);
    EXPECT_EQ(v_i64.i64, 123456789);

    arc::WasmValue v_f32(3.14f);
    EXPECT_EQ(v_f32.kind, arc::WasmValue::Kind::F32);
    EXPECT_FLOAT_EQ(v_f32.f32, 3.14f);

    arc::WasmValue v_f64(2.71828);
    EXPECT_EQ(v_f64.kind, arc::WasmValue::Kind::F64);
    EXPECT_DOUBLE_EQ(v_f64.f64, 2.71828);
}

// TODO: Add tests with actual AOT-compiled WASM modules once we have examples:
// - LoadValidAOTModule
// - InstantiateModule
// - FindExportedFunction
// - CallWASMFunction
// - HandleWASMTrap
