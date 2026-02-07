// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cmath>

#include "arc/cpp/stl/stl.h"

namespace arc::runtime::stl::math {

template<typename T>
T int_pow(T base, T exp) {
    if (exp == 0) return 1;
    T result = 1;
    for (T i = 0; i < exp; ++i)
        result *= base;
    return result;
}

class Module : public stl::Module {
public:
    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
        bind_float<float>(linker, "f32");
        bind_float<double>(linker, "f64");
        bind_int<uint8_t>(linker, "u8");
        bind_int<uint16_t>(linker, "u16");
        bind_int<uint32_t>(linker, "u32");
        bind_int<uint64_t>(linker, "u64");
        bind_int<int8_t>(linker, "i8");
        bind_int<int16_t>(linker, "i16");
        bind_int<int32_t>(linker, "i32");
        bind_int<int64_t>(linker, "i64");
    }

private:
    template<typename T>
    static void bind_float(wasmtime::Linker &linker, const std::string &suffix) {
        using W = typename WasmType<T>::type;
        linker
            .func_wrap(
                "math",
                "pow_" + suffix,
                [](W base, W exp) -> W {
                    return static_cast<W>(
                        std::pow(static_cast<T>(base), static_cast<T>(exp))
                    );
                }
            )
            .unwrap();
    }

    template<typename T>
    static void bind_int(wasmtime::Linker &linker, const std::string &suffix) {
        using W = typename WasmType<T>::type;
        linker
            .func_wrap(
                "math",
                "pow_" + suffix,
                [](W base, W exp) -> W {
                    return static_cast<W>(
                        int_pow(static_cast<T>(base), static_cast<T>(exp))
                    );
                }
            )
            .unwrap();
    }
};

}
