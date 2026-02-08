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

#include "x/cpp/telem/telem.h"
#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xmemory/local_shared.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/stl/stl.h"

namespace arc::stl::telem {

/// @brief Source node that reads from a channel and outputs the data.
/// Tracks a high water mark to avoid duplicate processing of the same data.
class On : public runtime::node::Node {
    runtime::state::Node state;
    types::ChannelKey channel_key;
    ::telem::Alignment high_water_mark{0};

public:
    On(runtime::state::Node &&state, const types::ChannelKey channel_key):
        state(std::move(state)), channel_key(channel_key) {}

    xerrors::Error next(runtime::node::Context &ctx) override {
        auto [data, index_data, ok] = this->state.read_chan(this->channel_key);
        if (!ok) return xerrors::NIL;

        for (size_t i = 0; i < data.series.size(); i++) {
            auto &ser = data.series[i];
            auto lower = ser.alignment;
            const auto upper_val = lower.uint64() +
                                   (ser.size() > 0 ? ser.size() - 1 : 0);

            if (lower.uint64() < this->high_water_mark.uint64()) continue;

            const bool generate_synthetic = index_data.empty();
            if (!generate_synthetic && i >= index_data.series.size())
                return xerrors::NIL;

            ::telem::Series time_series = generate_synthetic
                                            ? ::telem::Series(
                                                  ::telem::TIMESTAMP_T,
                                                  ser.size()
                                              )
                                            : std::move(index_data.series[i]);

            if (generate_synthetic) {
                const auto now = ::telem::TimeStamp::now();
                for (size_t j = 0; j < ser.size(); j++)
                    time_series.write(
                        ::telem::TimeStamp(now.nanoseconds() + static_cast<int64_t>(j))
                    );
                time_series.alignment = ser.alignment;
            } else if (time_series.alignment != ser.alignment)
                return xerrors::NIL;

            this->state.output(0) = xmemory::make_local_shared<::telem::Series>(
                ser.deep_copy()
            );
            this->state.output_time(0) = xmemory::make_local_shared<::telem::Series>(
                std::move(time_series)
            );
            this->high_water_mark = ::telem::Alignment(upper_val + 1);
            ctx.mark_changed(ir::default_output_param);
            return xerrors::NIL;
        }
        return xerrors::NIL;
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

    xerrors::Error next(runtime::node::Context & /*ctx*/) override {
        if (!this->state.refresh_inputs()) return xerrors::NIL;
        const auto &data = this->state.input(0);
        if (data->empty()) return xerrors::NIL;
        // TODO: Fix this hacky code
        const auto start = ::telem::TimeStamp::now();
        const auto time = xmemory::local_shared(
            ::telem::Series::linspace(
                start,
                start + 100 * ::telem::MICROSECOND,
                data->size()
            )
        );
        this->state.write_chan(this->channel_key, data, time);
        return xerrors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param_name) const override {
        return this->state.is_output_truthy(param_name);
    }
};

class Module : public stl::Module {
public:
    std::shared_ptr<runtime::node::Factory> factory() override {
        return std::make_shared<IOFactory>();
    }

private:
    class IOFactory : public runtime::node::Factory {
    public:
        bool handles(const std::string &node_type) const override {
            return node_type == "on" || node_type == "write";
        }

        std::pair<std::unique_ptr<runtime::node::Node>, xerrors::Error>
        create(runtime::node::Config &&cfg) override {
            if (!this->handles(cfg.node.type)) return {nullptr, xerrors::NOT_FOUND};
            auto channel_key = cfg.node.config["channel"].get<types::ChannelKey>();
            if (cfg.node.type == "on")
                return {
                    std::make_unique<On>(std::move(cfg.state), channel_key),
                    xerrors::NIL
                };
            return {
                std::make_unique<Write>(std::move(cfg.state), channel_key),
                xerrors::NIL
            };
        }
    };
};

}
