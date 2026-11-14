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

namespace arc::runtime::testutil {
/// @brief opens a WASM module that exports only its memory, for a runtime that runs no
/// WASM function. Throws if the module fails to open.
inline std::shared_ptr<wasm::Module> empty_module() {
    program::Program program;
    // (module (memory (export "memory") 1))
    program.wasm = {
        0x00, 0x61, 0x73, 0x6d, 0x01, 0x00, 0x00, 0x00, 0x05, 0x03, 0x01, 0x00, 0x01,
        0x07, 0x0a, 0x01, 0x06, 'm',  'e',  'm',  'o',  'r',  'y',  0x02, 0x00,
    };
    auto [mod, err] = wasm::Module::open({.program = std::move(program)});
    if (err) throw std::runtime_error(err.message());
    return mod;
}
}
