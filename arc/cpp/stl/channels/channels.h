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
#include "arc/cpp/stl/channels/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/stl/strings/state.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::channels {

inline constexpr const char *MODULE_NAME = "channels";

/// @brief marks a node whose channel input is not alias-bound.
inline constexpr size_t NO_CHANNEL = ~size_t{0};

/// @brief returns the channel a node currently targets: the binding edge's
/// latest key when present, otherwise the configured key.
inline types::ChannelKey bound_key(
    const runtime::state::Node &s,
    const size_t channel_idx,
    const types::ChannelKey configured
) {
    if (const auto t = s.ref_input(channel_idx); t != nullptr && t->size() > 0)
        return t->at<uint32_t>(-1);
    return configured;
}

/// @brief Source node that reads from a channel and outputs the data.
/// Tracks a high water mark to avoid duplicate processing of the same data.
class On : public runtime::node::Node {
    runtime::state::Node state;
    types::ChannelKey key;
    types::ChannelKey curr_key;
    /// @brief channel_idx is the channel ref input's index; NO_CHANNEL when not
    /// alias-bound.
    size_t channel_idx;
    ::x::telem::Alignment high_water_mark{0};
    ::x::telem::MonoClock clock;

    /// @brief re-points the source at key. A rebind is not a value:
    /// only values arriving afterward fire
    void rebind_to(const types::ChannelKey key) {
        this->curr_key = key;
        this->high_water_mark = ::x::telem::Alignment(0);
        this->raise_water_mark();
    }

    /// @brief advances the mark past all buffered data on the bound channel.
    void raise_water_mark() {
        auto [data, index_data, ok] = this->state.read_series(this->curr_key);
        if (!ok || data.series.empty()) return;
        const auto &last = data.series.back();
        const auto lower = last.alignment;
        const auto upper_val = lower.uint64() + (last.size() > 0 ? last.size() - 1 : 0);
        const auto upper = x::telem::Alignment(upper_val + 1);
        if (upper.uint64() > this->high_water_mark.uint64())
            this->high_water_mark = upper;
    }

public:
    On(runtime::state::Node &&state,
       const types::ChannelKey key,
       const size_t channel_idx):
        state(std::move(state)), key(key), curr_key(key), channel_idx(channel_idx) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (const auto k = bound_key(this->state, this->channel_idx, this->key);
            k != this->curr_key) {
            this->rebind_to(k);
            return x::errors::NIL;
        }
        auto [data, index_data, ok] = this->state.read_series(this->curr_key);
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
                const auto now = this->clock.now();
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
            ctx.mark_changed(0);
            return x::errors::NIL;
        }
        return x::errors::NIL;
    }

    /// @brief advances the high water mark to the current channel alignment,
    /// ensuring that when a stage is (re-)activated it only responds to
    /// data that arrives after activation rather than stale pre-existing data.
    void reset() override {
        this->state.reset();
        if (const auto k = bound_key(this->state, this->channel_idx, this->key);
            k != this->curr_key) {
            this->rebind_to(k);
            return;
        }
        this->raise_water_mark();
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }
};

/// @brief Sink node that writes input data to a channel.
class Write : public runtime::node::Node {
    runtime::state::Node state;
    types::ChannelKey key;
    size_t input_idx;
    /// @brief channel_idx is the channel ref input's index; NO_CHANNEL when not
    /// alias-bound.
    size_t channel_idx;
    ::x::telem::MonoClock clock;

public:
    Write(
        runtime::state::Node &&state,
        const types::ChannelKey key,
        size_t input_idx,
        const size_t channel_idx
    ):
        state(std::move(state)),
        key(key),
        input_idx(input_idx),
        channel_idx(channel_idx) {}

    x::errors::Error next(runtime::node::Context &ctx) override {
        if (!this->state.refresh_inputs()) return x::errors::NIL;
        const auto &data = this->state.input(this->input_idx);
        if (data->empty()) return x::errors::NIL;
        const auto start = this->clock.now();
        const auto time = x::mem::local_shared(
            ::x::telem::Series::linspace(
                start,
                start + 100 * ::x::telem::MICROSECOND,
                data->size()
            )
        );
        this->state.write_series(
            bound_key(this->state, this->channel_idx, this->key),
            data,
            time
        );
        auto &out = this->state.output(0);
        out->resize(1);
        out->set(0, static_cast<uint8_t>(1));
        out->alignment = data->alignment;
        out->time_range = data->time_range;
        auto &out_time = this->state.output_time(0);
        out_time->resize(1);
        out_time->set(0, time->at<int64_t>(time->size() - 1));
        out_time->alignment = data->alignment;
        out_time->time_range = data->time_range;
        ctx.mark_changed(0);
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(size_t output_idx) const override {
        return this->state.is_output_truthy(output_idx);
    }

    void reset() override { this->state.reset(); }
};

class Module : public stl::Module {
    std::shared_ptr<State> channel;
    std::shared_ptr<strings::State> str_state;
    x::telem::MonoClock clock;

public:
    Module(std::shared_ptr<State> channel, std::shared_ptr<strings::State> str_state):
        channel(std::move(channel)), str_state(std::move(str_state)) {}

    bool handles(const std::string &node_type) const override {
        return node_type == "on" || node_type == "write";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        size_t channel_idx = NO_CHANNEL;
        if (const auto [idx, terr] = cfg.state.resolve_input("channel"); !terr)
            channel_idx = idx;
        const auto &ch_param = cfg.node.inputs["channel"];
        auto ch_sv = types::to_sample_value(ch_param.value, ch_param.type);
        // An unbound source or sink requires its channel key as an input value.
        // An alias-bound source or sink takes its key from the binding edge at
        // runtime.
        if (!ch_sv.has_value() && !cfg.state.ref_sourced(channel_idx))
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    std::string(cfg.node.type) +
                        " node missing required channel parameter"
                )
            };
        const auto channel_key = ch_sv.has_value()
                                   ? x::telem::cast<types::ChannelKey>(*ch_sv)
                                   : types::ChannelKey(0);
        if (cfg.node.type == "on")
            return {
                std::make_unique<On>(std::move(cfg.state), channel_key, channel_idx),
                x::errors::NIL
            };
        auto [input_idx, in_err] = cfg.node.resolve_input(ir::default_input_param);
        if (in_err) return {nullptr, in_err};
        return {
            std::make_unique<Write>(
                std::move(cfg.state),
                channel_key,
                input_idx,
                channel_idx
            ),
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
                MODULE_NAME,
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
                MODULE_NAME,
                "write_" + suffix,
                [ch, dt](uint32_t channel_id, W value) {
                    ch->write_channel_typed(
                        static_cast<types::ChannelKey>(channel_id),
                        dt,
                        static_cast<T>(value)
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
                MODULE_NAME,
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
                MODULE_NAME,
                "write_str",
                [this, ch, ss](uint32_t channel_id, uint32_t str_handle) {
                    std::string str_value = ss->get(str_handle);
                    if (str_value.empty()) return;
                    const auto data = x::mem::make_local_shared<x::telem::Series>(
                        str_value
                    );
                    const auto time = x::mem::make_local_shared<x::telem::Series>(
                        this->clock.now()
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
