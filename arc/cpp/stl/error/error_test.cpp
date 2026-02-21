// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <vector>

#include "gtest/gtest.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/stl/error/error.h"
#include "wasmtime.hh"

namespace arc::stl::error {
const std::string_view PANIC_WAT = R"wat(
(module
  (import "error" "panic" (func $panic (param i32 i32)))
  (memory (export "memory") 1)
  (data (i32.const 0) "test error message")
  (func (export "trigger_panic")
    (call $panic (i32.const 0) (i32.const 18)))
  (func (export "trigger_oob_panic")
    (call $panic (i32.const 65530) (i32.const 100)))
)
)wat";

TEST(ErrorModule, PanicInvokesHandlerWithCorrectMessage) {
    x::errors::Error captured;
    Module mod([&](const x::errors::Error &err) { captured = err; });

    wasmtime::Engine engine;
    wasmtime::Store store(engine);
    wasmtime::Linker linker(engine);
    mod.bind_to(linker, store);

    auto wasm_mod = wasmtime::Module::compile(engine, PANIC_WAT).unwrap();
    auto instance = linker.instantiate(store, wasm_mod).unwrap();
    auto memory = std::get<wasmtime::Memory>(*instance.get(store, "memory"));
    mod.set_wasm_context(&store, &memory);

    auto trigger = std::get<wasmtime::Func>(*instance.get(store, "trigger_panic"));
    (void)trigger.call(store, {});
    EXPECT_TRUE(captured.matches(runtime::errors::WASM_PANIC));
    EXPECT_NE(captured.message().find("test error message"), std::string::npos);
}

TEST(ErrorModule, PanicReportsOutOfBoundsForInvalidPointer) {
    x::errors::Error captured;
    Module mod([&](const x::errors::Error &err) { captured = err; });

    wasmtime::Engine engine;
    wasmtime::Store store(engine);
    wasmtime::Linker linker(engine);
    mod.bind_to(linker, store);

    auto wasm_mod = wasmtime::Module::compile(engine, PANIC_WAT).unwrap();
    auto instance = linker.instantiate(store, wasm_mod).unwrap();
    auto memory = std::get<wasmtime::Memory>(*instance.get(store, "memory"));
    mod.set_wasm_context(&store, &memory);

    auto trigger = std::get<wasmtime::Func>(*instance.get(store, "trigger_oob_panic"));
    (void)trigger.call(store, {});
    EXPECT_TRUE(captured.matches(runtime::errors::WASM_PANIC));
    EXPECT_NE(captured.message().find("out of bounds"), std::string::npos);
}

TEST(ErrorModule, PanicBeforeSetWasmContextReportsNoMemory) {
    x::errors::Error captured;
    Module mod([&](const x::errors::Error &err) { captured = err; });

    wasmtime::Engine engine;
    wasmtime::Store store(engine);
    wasmtime::Linker linker(engine);
    mod.bind_to(linker, store);

    auto wasm_mod = wasmtime::Module::compile(engine, PANIC_WAT).unwrap();
    auto instance = linker.instantiate(store, wasm_mod).unwrap();

    auto trigger = std::get<wasmtime::Func>(*instance.get(store, "trigger_panic"));
    (void)trigger.call(store, {});
    EXPECT_TRUE(captured.matches(runtime::errors::WASM_PANIC));
    EXPECT_NE(captured.message().find("no memory available"), std::string::npos);
}
}
