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

#include "x/cpp/mem/local_shared.h"
#include "x/cpp/telem/series.h"

#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/str/state.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::channel {

class Module : public stl::Module {
    std::shared_ptr<runtime::state::State> state;
    std::shared_ptr<str::State> str_state;

public:
    Module(
        std::shared_ptr<runtime::state::State> state,
        std::shared_ptr<str::State> str_state
    ):
        state(std::move(state)), str_state(std::move(str_state)) {}

    void bind_to(wasmtime::Linker &linker, wasmtime::Store::Context cx) override {
        bind_ops<uint8_t>(linker, "u8", x::telem::UINT8_T);
        bind_ops<uint16_t>(linker, "u16", x::telem::UINT16_T);
        bind_ops<uint32_t>(linker, "u32", x::telem::UINT32_T);
        bind_ops<uint64_t>(linker, "u64", x::telem::UINT64_T);
        bind_ops<int8_t>(linker, "i8", x::telem::INT8_T);
        bind_ops<int16_t>(linker, "i16", x::telem::INT16_T);
        bind_ops<int32_t>(linker, "i32", x::telem::INT32_T);
        bind_ops<int64_t>(linker, "i64", x::telem::INT64_T);
        bind_ops<float>(linker, "f32", x::telem::FLOAT32_T);
        bind_ops<double>(linker, "f64", x::telem::FLOAT64_T);
        bind_str_ops(linker);
    }

private:
    template<typename T>
    void bind_ops(
        wasmtime::Linker &linker,
        const std::string &suffix,
        x::telem::DataType dt
    ) {
        using W = typename WasmType<T>::type;
        auto s = this->state;
        linker
            .func_wrap(
                "channel",
                "read_" + suffix,
                [s](uint32_t channel_id) -> W {
                    auto [multi_series, ok] = s->read_channel(
                        static_cast<types::ChannelKey>(channel_id)
                    );
                    if (!ok || multi_series.series.empty()) return W{};
                    const auto &last = multi_series.series.back();
                    if (last.size() == 0) return W{};
                    return static_cast<W>(last.at<T>(-1));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "channel",
                "write_" + suffix,
                [s, dt](uint32_t channel_id, W value) {
                    auto data = x::mem::make_local_shared<x::telem::Series>(
                        static_cast<T>(value),
                        dt
                    );
                    auto time = x::mem::make_local_shared<x::telem::Series>(
                        x::telem::TimeStamp::now()
                    );
                    s->write_channel(
                        static_cast<types::ChannelKey>(channel_id),
                        data,
                        time
                    );
                }
            )
            .unwrap();
    }

    void bind_str_ops(wasmtime::Linker &linker) {
        auto s = this->state;
        auto ss = this->str_state;
        linker
            .func_wrap(
                "channel",
                "read_str",
                [s, ss](uint32_t channel_id) -> uint32_t {
                    auto [multi_series, ok] = s->read_channel(channel_id);
                    if (!ok || multi_series.series.empty()) return 0;
                    const auto &last = multi_series.series.back();
                    if (last.size() == 0) return 0;
                    return ss->create(last.at<std::string>(-1));
                }
            )
            .unwrap();
        linker
            .func_wrap(
                "channel",
                "write_str",
                [s, ss](uint32_t channel_id, uint32_t str_handle) {
                    std::string str_value = ss->get(str_handle);
                    if (str_value.empty()) return;
                    const auto data = x::mem::make_local_shared<x::telem::Series>(
                        str_value
                    );
                    const auto time = x::mem::make_local_shared<x::telem::Series>(
                        x::telem::TimeStamp::now()
                    );
                    s->write_channel(
                        static_cast<types::ChannelKey>(channel_id),
                        data,
                        time
                    );
                }
            )
            .unwrap();
    }
};

}
