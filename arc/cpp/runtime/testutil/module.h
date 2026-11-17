// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <stdexcept>

#include "arc/cpp/program/program.h"
#include "arc/cpp/runtime/wasm/module.h"
#include "wasmtime/wat.hh"

namespace arc::runtime::testutil {
/// @brief opens a WASM module that exports only its memory and loop budget, for a
/// runtime that runs no WASM function. Throws if the module fails to open.
inline std::shared_ptr<wasm::Module> empty_module() {
    program::Program program;
    program.wasm = wasmtime::wat2wasm(R"((module
        (memory (export "memory") 1)
        (global (export "$loop_budget") (mut i64) (i64.const 0))
    ))")
                       .unwrap();
    auto [mod, err] = wasm::Module::open({.program = std::move(program)});
    if (err) throw std::runtime_error(err.message());
    return mod;
}

/// @brief opens a WASM module whose `spin` function uses the whole loop budget. Throws
/// if the module fails to open.
inline std::shared_ptr<wasm::Module> spin_module() {
    program::Program program;
    program.wasm = wasmtime::wat2wasm(R"((module
        (memory (export "memory") 1)
        (global $budget (export "$loop_budget") (mut i64) (i64.const 0))
        (func (export "spin") (global.set $budget (i64.const 0)))
    ))")
                       .unwrap();
    program.functions.push_back(ir::Function{.key = "spin"});
    auto [mod, err] = wasm::Module::open({.program = std::move(program)});
    if (err) throw std::runtime_error(err.message());
    return mod;
}
}
