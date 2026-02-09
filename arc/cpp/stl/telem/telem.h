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
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/stl/stl.h"

namespace arc::stl::telem {

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
        auto [data, index_data, ok] = this->state.read_chan(this->channel_key);
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
        // TODO: Fix this hacky code
        const auto start = ::x::telem::TimeStamp::now();
        const auto time = x::mem::local_shared(
            ::x::telem::Series::linspace(
                start,
                start + 100 * ::x::telem::MICROSECOND,
                data->size()
            )
        );
        this->state.write_chan(this->channel_key, data, time);
        return x::errors::NIL;
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
    };
};

}
