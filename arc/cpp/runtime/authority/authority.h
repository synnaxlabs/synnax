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
#include <optional>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::authority {

/// @brief Node that buffers an authority change request in the runtime state.
/// When executed, it calls state.set_authority() to enqueue the change.
class SetAuthority : public node::Node {
    state::State &state;
    uint8_t authority_value;
    std::optional<types::ChannelKey> channel_key;

public:
    SetAuthority(
        state::State &state,
        const uint8_t authority,
        std::optional<types::ChannelKey> channel_key
    ):
        state(state),
        authority_value(authority),
        channel_key(std::move(channel_key)) {}

    xerrors::Error next(node::Context & /*ctx*/) override {
        state.set_authority(channel_key, authority_value);
        return xerrors::NIL;
    }

    void reset() override {}

    [[nodiscard]] bool is_output_truthy(const std::string &) const override {
        return false;
    }
};

class Factory : public node::Factory {
    std::shared_ptr<state::State> state;

public:
    explicit Factory(std::shared_ptr<state::State> state):
        state(std::move(state)) {}

    bool handles(const std::string &node_type) const override {
        return node_type == "set_authority";
    }

    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, xerrors::NOT_FOUND};
        const auto auth = cfg.node.config["value"].get<uint8_t>();
        const auto channel = cfg.node.config["channel"].get<types::ChannelKey>();
        std::optional<types::ChannelKey> channel_key;
        if (channel != 0) channel_key = channel;
        return {
            std::make_unique<SetAuthority>(*state, auth, channel_key),
            xerrors::NIL
        };
    }
};

}
