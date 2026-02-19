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

#include "x/cpp/errors/errors.h"
#include "x/cpp/mem/local_shared.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/stl/channel/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/str/state.h"

namespace arc::stl::channel {

/// @brief Source node that reads from a channel and outputs the data.
/// Tracks a high water mark to avoid duplicate processing of the same data.
class On : public runtime::node::Node {
    runtime::state::Node state;
    types::ChannelKey channel_key;
    ::x::telem::Alignment high_water_mark{0};

public:
    On(runtime::state::Node &&state, const types::ChannelKey channel_key):
        state(std::move(state)), channel_key(channel_key) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        auto [data, index_data, ok] = this->state.read_series(this->channel_key);
        if (!ok) return x::errors::NIL;

        for (size_t i = 0; i < data.series.size(); i++) {
            auto &ser = data.series[i];
            auto lower = ser.alignment;
            const auto upper_val = lower.uint64() +
                                   (ser.size() > 0 ? ser.size() - 1 : 0);

            if (lower.uint64() < this->high_water_mark.uint64()) continue;

            const bool generate_synthetic = index_data.empty();
            if (!generate_synthetic && i >= index_data.series.size())
                return x::errors::NIL;

            ::x::telem::Series time_series = generate_synthetic
                                               ? ::x::telem::Series(
                                                     ::x::telem::TIMESTAMP_T,
                                                     ser.size()
                                                 )
                                               : std::move(index_data.series[i]);

            if (generate_synthetic) {
                const auto now = ::x::telem::TimeStamp::now();
                for (size_t j = 0; j < ser.size(); j++)
                    time_series.write(
                        ::x::telem::TimeStamp(
                            now.nanoseconds() + static_cast<int64_t>(j)
                        )
                    );
                time_series.alignment = ser.alignment;
            } else if (time_series.alignment != ser.alignment)
                return x::errors::NIL;

            this->state.output(0) = x::mem::make_local_shared<::x::telem::Series>(
                ser.deep_copy()
            );
            this->state.output_time(0) = x::mem::make_local_shared<::x::telem::Series>(
                std::move(time_series)
            );
            this->high_water_mark = ::x::telem::Alignment(upper_val + 1);
            ctx.mark_changed(ir::default_output_param);
            return x::errors::NIL;
        }
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return this->state.is_output_truthy(param_name);
    }
};

/// @brief Sink node that writes input data to a channel.
class Write : public runtime::node::Node {
    runtime::state::Node state;
    types::ChannelKey channel_key;

public:
    Write(runtime::state::Node &&state, const types::ChannelKey channel_key):
        state(std::move(state)), channel_key(channel_key) {}

    x::errors::Error next(runtime::node::Context & /*ctx*/) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &data = this->state.input(0);
        if (data->empty()) return x::errors::NIL;
        const auto start = ::x::telem::TimeStamp::now();
        const auto time = x::mem::local_shared(
            ::x::telem::Series::linspace(
                start,
                start + 100 * ::x::telem::MICROSECOND,
                data->size()
            )
        );
        this->state.write_series(this->channel_key, data, time);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return this->state.is_output_truthy(param_name);
    }
};

class Module : public stl::Module {
    std::shared_ptr<State> channel;
    std::shared_ptr<str::State> str_state;

public:
    Module(std::shared_ptr<State> channel, std::shared_ptr<str::State> str_state):
        channel(std::move(channel)), str_state(std::move(str_state)) {}

    bool handles(const std::string &node_type) const override {
        return node_type == "on" || node_type == "write";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        auto channel_key = cfg.node.config["channel"].get<types::ChannelKey>();
        if (cfg.node.type == "on")
            return {
                std::make_unique<On>(std::move(cfg.state), channel_key),
                x::errors::NIL
            };
        return {
            std::make_unique<Write>(std::move(cfg.state), channel_key),
            x::errors::NIL
        };
    }

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
        auto ch = this->channel;
        linker
            .func_wrap(
                "channel",
                "read_" + suffix,
                [ch](uint32_t channel_id) -> W {
                    auto [multi_series, ok] = ch->read_value(
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
                [ch, dt](uint32_t channel_id, W value) {
                    auto data = x::mem::make_local_shared<x::telem::Series>(
                        static_cast<T>(value),
                        dt
                    );
                    auto time = x::mem::make_local_shared<x::telem::Series>(
                        x::telem::TimeStamp::now()
                    );
                    ch->write_value(
                        static_cast<types::ChannelKey>(channel_id),
                        data,
                        time
                    );
                }
            )
            .unwrap();
    }

    void bind_str_ops(wasmtime::Linker &linker) {
        auto ch = this->channel;
        auto ss = this->str_state;
        linker
            .func_wrap(
                "channel",
                "read_str",
                [ch, ss](uint32_t channel_id) -> uint32_t {
                    auto [multi_series, ok] = ch->read_value(channel_id);
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
                [ch, ss](uint32_t channel_id, uint32_t str_handle) {
                    std::string str_value = ss->get(str_handle);
                    if (str_value.empty()) return;
                    const auto data = x::mem::make_local_shared<x::telem::Series>(
                        str_value
                    );
                    const auto time = x::mem::make_local_shared<x::telem::Series>(
                        x::telem::TimeStamp::now()
                    );
                    ch->write_value(
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
