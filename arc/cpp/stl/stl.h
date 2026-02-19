// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <memory>

#include "x/cpp/errors/errors.h"

#include "arc/cpp/runtime/node/factory.h"
#include "wasmtime.hh"

namespace arc::stl {

/// WasmType maps C++ types to their WASM-compatible equivalents.
/// WASM only has i32, i64, f32, f64 - smaller integer types must be widened.
template<typename T>
struct WasmType {
    using type = T;
};
template<>
struct WasmType<uint8_t> {
    using type = uint32_t;
};
template<>
struct WasmType<uint16_t> {
    using type = uint32_t;
};
template<>
struct WasmType<int8_t> {
    using type = int32_t;
};
template<>
struct WasmType<int16_t> {
    using type = int32_t;
};

/// Module is the unit of STL organization. Each module groups related host
/// functions and optionally provides a node factory. Modules that create nodes
/// override handles() and create() directly.
class Module : public runtime::node::Factory {
public:
    ~Module() override = default;

    /// Registers host functions with the WASM Linker under a named module.
    virtual void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) {}

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        return {nullptr, x::errors::NOT_FOUND};
    }

    /// Provides WASM memory and store access after instantiation.
    virtual void set_wasm_context(wasmtime::Store *store, wasmtime::Memory *memory) {}

    /// Clears transient state at end of execution cycle.
    virtual void clear_cycle() {}

    /// Full reset of all persistent state.
    virtual void reset() {}
};

}
