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

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::io {

/// On is a source node that reads from a channel and outputs the data.
/// When channel data arrives, it outputs the latest value.
class On : public node::Node {
    state::Node state;
    types::ChannelKey channel_key;

public:
    On(state::Node state, types::ChannelKey channel_key):
        state(std::move(state)),
        channel_key(channel_key) {}

    xerrors::Error next(node::Context &ctx) override {
        // Source nodes read from channel state via channel_key.
        // The data is made available via the scheduler when channel data arrives.
        (void)channel_key; // TODO: use channel_key to read from state
        auto &output = state.output(0);
        if (output->size() == 0) {
            output->resize(1);
        }
        ctx.mark_changed(ir::default_output_param);
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
        // Input data is written to channel via channel_key
        (void)channel_key; // TODO: use channel_key to write to state
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
