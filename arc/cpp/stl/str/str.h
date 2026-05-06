// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <charconv>
#include <cmath>
#include <cstdio>
#include <memory>
#include <string>

#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/str/state.h"

namespace arc::stl::str {

/// Formats v as the shortest round-trippable decimal, matching Go's
/// strconv.FormatFloat(v, 'g', -1, bitSize). NaN and ±Inf are emitted as
/// "NaN", "+Inf", "-Inf" to match Go's output exactly.
template<typename T>
std::string format_float(T v) {
    if (std::isnan(v)) return "NaN";
    if (std::isinf(v)) return v > 0 ? "+Inf" : "-Inf";
    char buf[32];
    const auto [end, ec] = std::to_chars(
        buf,
        buf + sizeof(buf),
        v,
        std::chars_format::general
    );
    return {buf, end};
}

class Module : public stl::Module {
    std::shared_ptr<State> str_state;
    wasmtime::Store *store = nullptr;
    wasmtime::Memory *memory = nullptr;

public:
    explicit Module(std::shared_ptr<State> str_state):
        str_state(std::move(str_state)) {}

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
        // SAFETY: raw `this` capture is safe because wasm::Module owns this
        // stl::Module via cfg.modules (shared_ptr), and Store/Memory are
        // stable members of the heap-pinned wasm::Module.
        auto self = this;
        auto ss = this->str_state;
        linker
            .func_wrap(
                "string",
                "from_literal",
                [self, ss](uint32_t ptr, uint32_t len) -> uint32_t {
                    if (!self->memory || !self->store) {
                        std::fprintf(
                            stderr,
                            "ERROR: string_from_literal called but no memory or "
                            "store available\n"
                        );
                        return 0;
                    }
                    const auto mem_span = self->memory->data(*self->store);
                    const uint8_t *mem_data = mem_span.data();
                    if (const size_t mem_size = mem_span.size(); ptr + len > mem_size) {
                        std::fprintf(
                            stderr,
                            "ERROR: string_from_literal ptr=%u len=%u exceeds "
                            "memory size=%zu\n",
                            ptr,
                            len,
                            mem_size
                        );
                        return 0;
                    }
                    return ss->from_memory(mem_data + ptr, len);
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "concat",
                [ss](uint32_t h1, uint32_t h2) -> uint32_t {
                    const std::string s1 = ss->get(h1);
                    const std::string s2 = ss->get(h2);
                    if (s1.empty() && s2.empty()) return 0;
                    return ss->create(s1 + s2);
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "equal",
                [ss](uint32_t h1, uint32_t h2) -> uint32_t {
                    if (!ss->exists(h1) || !ss->exists(h2)) return 0;
                    return ss->get(h1) == ss->get(h2) ? 1 : 0;
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "len",
                [ss](uint32_t handle) -> uint64_t {
                    return static_cast<uint64_t>(ss->get(handle).length());
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "from_i32",
                [ss](int32_t v) -> uint32_t { return ss->create(std::to_string(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "from_u32",
                [ss](uint32_t v) -> uint32_t { return ss->create(std::to_string(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "from_i64",
                [ss](int64_t v) -> uint32_t { return ss->create(std::to_string(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "from_u64",
                [ss](uint64_t v) -> uint32_t { return ss->create(std::to_string(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "from_f32",
                [ss](float v) -> uint32_t { return ss->create(format_float(v)); }
            )
            .unwrap();
        linker
            .func_wrap(
                "string",
                "from_f64",
                [ss](double v) -> uint32_t { return ss->create(format_float(v)); }
            )
            .unwrap();
    }

    void set_wasm_context(wasmtime::Store *store, wasmtime::Memory *memory) override {
        this->store = store;
        this->memory = memory;
    }
};

}
