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
#include <string>

#include "arc/cpp/stl/series/state.h"
#include "arc/cpp/stl/stateful/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/str/state.h"

namespace arc::stl::stateful {

class Module : public stl::Module {
    std::shared_ptr<Variables> vars;
    std::shared_ptr<series::State> series_state;
    std::shared_ptr<str::State> str_state;

public:
    Module(
        std::shared_ptr<Variables> vars,
        std::shared_ptr<series::State> series_state,
        std::shared_ptr<str::State> str_state
    ):
        vars(std::move(vars)),
        series_state(std::move(series_state)),
        str_state(std::move(str_state)) {}

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
#define BIND_STATE_OPS(suffix, cpptype)                                                \
    bind_ops<cpptype>(                                                                 \
        linker,                                                                        \
        #suffix,                                                                       \
        [](Variables &v, uint32_t id, cpptype init) -> cpptype {                       \
            return v.load_##suffix(id, init);                                          \
        },                                                                             \
        [](Variables &v, uint32_t id, cpptype val) { v.store_##suffix(id, val); }      \
    );

        BIND_STATE_OPS(u8, uint8_t)
        BIND_STATE_OPS(u16, uint16_t)
        BIND_STATE_OPS(u32, uint32_t)
        BIND_STATE_OPS(u64, uint64_t)
        BIND_STATE_OPS(i8, int8_t)
        BIND_STATE_OPS(i16, int16_t)
        BIND_STATE_OPS(i32, int32_t)
        BIND_STATE_OPS(i64, int64_t)
        BIND_STATE_OPS(f32, float)
        BIND_STATE_OPS(f64, double)

#undef BIND_STATE_OPS

        bind_str_ops(linker);

#define BIND_SERIES_STATE_OPS(suffix) bind_series_ops(linker, #suffix);

        BIND_SERIES_STATE_OPS(u8)
        BIND_SERIES_STATE_OPS(u16)
        BIND_SERIES_STATE_OPS(u32)
        BIND_SERIES_STATE_OPS(u64)
        BIND_SERIES_STATE_OPS(i8)
        BIND_SERIES_STATE_OPS(i16)
        BIND_SERIES_STATE_OPS(i32)
        BIND_SERIES_STATE_OPS(i64)
        BIND_SERIES_STATE_OPS(f32)
        BIND_SERIES_STATE_OPS(f64)

#undef BIND_SERIES_STATE_OPS
    }

private:
    template<typename T>
    void bind_ops(
        wasmtime::Linker &linker,
        const std::string &suffix,
        std::function<T(Variables &, uint32_t, T)> loader,
        std::function<void(Variables &, uint32_t, T)> storer
    ) {
        using W = typename WasmType<T>::type;
        auto v = this->vars;
        linker
            .func_wrap(
                "state",
                "load_" + suffix,
                [v, loader](uint32_t var_id, W init) -> W {
                    return static_cast<W>(loader(*v, var_id, static_cast<T>(init)));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "state",
                "store_" + suffix,
                [v, storer](uint32_t var_id, W value) {
                    storer(*v, var_id, static_cast<T>(value));
                }
            )
            .unwrap();
    }

    void bind_str_ops(wasmtime::Linker &linker) {
        auto v = this->vars;
        auto ss = this->str_state;
        linker
            .func_wrap(
                "state",
                "load_str",
                [v, ss](uint32_t var_id, uint32_t init_handle) -> uint32_t {
                    return v->load_str(var_id, init_handle, *ss);
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "state",
                "store_str",
                [v, ss](uint32_t var_id, uint32_t handle) {
                    v->store_str(var_id, handle, *ss);
                }
            )
            .unwrap();
    }

    void bind_series_ops(wasmtime::Linker &linker, const std::string &suffix) {
        auto v = this->vars;
        auto ss = this->series_state;
        linker
            .func_wrap(
                "state",
                "load_series_" + suffix,
                [v, ss](uint32_t var_id, uint32_t init_handle) -> uint32_t {
                    return v->load_series(var_id, init_handle, *ss);
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "state",
                "store_series_" + suffix,
                [v, ss](uint32_t var_id, uint32_t handle) {
                    v->store_series(var_id, handle, *ss);
                }
            )
            .unwrap();
    }
};

}
