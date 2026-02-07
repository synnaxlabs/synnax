// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdio>
#include <string>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/stl/stl.h"

namespace arc::runtime::stl::error {

class Module : public stl::Module {
    errors::Handler handler;
    wasmtime::Store *store = nullptr;
    wasmtime::Memory *memory = nullptr;

public:
    explicit Module(errors::Handler handler): handler(std::move(handler)) {}

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
        auto self = this;
        linker
            .func_wrap(
                "error",
                "panic",
                [self](uint32_t ptr, uint32_t len) {
                    std::string message;
                    if (!self->memory || !self->store) {
                        message = "no memory available";
                    } else {
                        const auto mem_span = self->memory->data(*self->store);
                        const uint8_t *mem_data = mem_span.data();
                        const size_t mem_size = mem_span.size();
                        if (ptr + len > mem_size)
                            message = "out of bounds";
                        else
                            message = std::string(
                                reinterpret_cast<const char *>(mem_data + ptr),
                                len
                            );
                    }
                    std::fprintf(stderr, "WASM panic: %s\n", message.c_str());
                    self->handler(xerrors::Error(errors::WASM_PANIC, message));
                }
            )
            .unwrap();
    }

    void set_wasm_context(wasmtime::Store *store, wasmtime::Memory *memory) override {
        this->store = store;
        this->memory = memory;
    }
};

}
