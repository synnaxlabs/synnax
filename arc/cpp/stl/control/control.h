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

#include "x/cpp/errors/errors.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/stl/stl.h"
#include "arc/cpp/types/types.h"

namespace arc::stl::authority {

/// @brief Node that buffers an authority change request in the runtime state.
/// When executed, it calls state.set_authority() to enqueue the change.
class SetAuthority : public runtime::node::Node {
    runtime::state::State &state;
    uint8_t authority_value;
    std::optional<types::ChannelKey> channel_key;
    bool initialized = false;

public:
    SetAuthority(
        runtime::state::State &state,
        const uint8_t authority,
        std::optional<types::ChannelKey> channel_key
    ):
        state(state), authority_value(authority), channel_key(std::move(channel_key)) {}

    x::errors::Error next(runtime::node::Context & /*ctx*/) override {
        if (this->initialized) return x::errors::NIL;
        this->initialized = true;
        this->state.set_authority(this->channel_key, this->authority_value);
        return x::errors::NIL;
    }

    void reset() override { this->initialized = false; }

    [[nodiscard]] bool is_output_truthy(size_t) const override { return false; }
};

class Module : public stl::Module {
    std::shared_ptr<runtime::state::State> state;

public:
    explicit Module(std::shared_ptr<runtime::state::State> state):
        state(std::move(state)) {}

    bool handles(const std::string &node_type) const override {
        return node_type == "set_authority";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        const auto &auth_param = cfg.node.config["value"];
        auto auth_sv = types::to_sample_value(auth_param.value, auth_param.type);
        if (!auth_sv.has_value())
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    "set_authority node missing required value parameter"
                )
            };
        const auto auth = x::telem::cast<uint8_t>(*auth_sv);
        const auto &ch_param = cfg.node.config["channel"];
        auto ch_sv = types::to_sample_value(ch_param.value, ch_param.type);
        const types::ChannelKey channel = ch_sv.has_value()
                                            ? x::telem::cast<types::ChannelKey>(*ch_sv)
                                            : 0;
        std::optional<types::ChannelKey> channel_key;
        if (channel != 0) channel_key = channel;
        return {
            std::make_unique<SetAuthority>(*this->state, auth, channel_key),
            x::errors::NIL
        };
    }
};

}
