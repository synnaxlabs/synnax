// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xmemory/local_shared.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::io {

/// On is a source node that reads from a channel and outputs the data.
/// Tracks a high water mark to avoid duplicate processing of the same data.
class On : public node::Node {
    state::Node state;
    types::ChannelKey channel_key;
    telem::Alignment high_water_mark{0};

public:
    On(state::Node state, types::ChannelKey channel_key):
        state(std::move(state)),
        channel_key(channel_key) {}

    xerrors::Error next(node::Context &ctx) override {
        auto [data, index_data, ok] = state.read_chan(channel_key);
        if (!ok) return xerrors::NIL;

        for (size_t i = 0; i < data.series.size(); i++) {
            auto &ser = data.series[i];
            auto lower = ser.alignment;
            auto upper_val = lower.uint64() + (ser.size() > 0 ? ser.size() - 1 : 0);

            if (lower.uint64() < high_water_mark.uint64()) continue;

            const bool generate_synthetic = index_data.empty();
            if (!generate_synthetic && i >= index_data.series.size())
                return xerrors::NIL;

            telem::Series time_series = generate_synthetic
                ? telem::Series(telem::TIMESTAMP_T, ser.size())
                : std::move(index_data.series[i]);

            if (generate_synthetic) {
                auto now = telem::TimeStamp::now();
                for (size_t j = 0; j < ser.size(); j++)
                    time_series.write(telem::TimeStamp(now.nanoseconds() + static_cast<int64_t>(j)));
                time_series.alignment = ser.alignment;
            } else if (time_series.alignment != ser.alignment) {
                return xerrors::NIL;
            }

            state.output(0) = xmemory::make_local_shared<telem::Series>(ser.deep_copy());
            state.output_time(0) = xmemory::make_local_shared<telem::Series>(std::move(time_series));
            high_water_mark = telem::Alignment(upper_val + 1);
            ctx.mark_changed(ir::default_output_param);
            return xerrors::NIL;
        }
        return xerrors::NIL;
    }
};

/// Write is a sink node that writes input data to a channel.
class Write : public node::Node {
    state::Node state;
    types::ChannelKey channel_key;

public:
    Write(state::Node state, types::ChannelKey channel_key):
        state(std::move(state)),
        channel_key(channel_key) {}

    xerrors::Error next(node::Context & /*ctx*/) override {
        if (!state.refresh_inputs()) return xerrors::NIL;
        const auto &data = state.input(0);
        const auto &time = state.input_time(0);
        if (data->empty()) return xerrors::NIL;
        state.write_chan(channel_key, data, time);
        return xerrors::NIL;
    }
};

/// Factory creates On and Write nodes for "on" and "write" type nodes in the IR.
class Factory : public node::Factory {
public:
    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "on" && cfg.node.type != "write")
            return {nullptr, xerrors::NOT_FOUND};

        auto channel_param = cfg.node.config.get("channel");
        if (channel_param == nullptr)
            return {nullptr, xerrors::Error("telem node missing channel config")};

        auto channel_key = channel_param->value.get<types::ChannelKey>();

        if (cfg.node.type == "on") {
            return {std::make_unique<On>(cfg.state, channel_key), xerrors::NIL};
        }
        return {std::make_unique<Write>(cfg.state, channel_key), xerrors::NIL};
    }
};

} // namespace arc::runtime::io
