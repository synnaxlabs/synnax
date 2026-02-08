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
/// functions and optionally provides a node factory.
class Module {
public:
    virtual ~Module() = default;

    /// Registers host functions with the WASM Linker under a named module.
    virtual void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) {}

    /// Returns a node factory, or nullptr if this module has none.
    virtual std::shared_ptr<runtime::node::Factory> factory() { return nullptr; }

    /// Provides WASM memory and store access after instantiation.
    virtual void set_wasm_context(wasmtime::Store *store, wasmtime::Memory *memory) {}

    /// Clears transient state at end of execution cycle.
    virtual void clear_cycle() {}

    /// Full reset of all persistent state.
    virtual void reset() {}
};

}
