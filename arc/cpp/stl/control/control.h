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

namespace arc::stl::control {

/// @brief Node that buffers an authority change request in the runtime state.
/// When executed, it calls set_authority() to enqueue the change. The channel
/// and value are read at fire time so var-bound params track live values.
class SetAuthority : public runtime::node::Node {
    runtime::state::Node state;
    runtime::state::State &auth;
    bool initialized = false;

public:
    SetAuthority(runtime::state::Node &&state, runtime::state::State &auth):
        state(std::move(state)), auth(auth) {}

    x::errors::Error next(runtime::node::Context & /*ctx*/) override {
        if (this->initialized) return x::errors::NIL;
        this->initialized = true;
        std::optional<types::ChannelKey> channel_key;
        if (const auto key = this->state.numeric_input<types::ChannelKey>("channel");
            key != 0)
            channel_key = key;
        this->auth.set_authority(
            channel_key,
            this->state.numeric_input<uint8_t>("value")
        );
        return x::errors::NIL;
    }

    void reset() override {
        this->state.reset();
        this->initialized = false;
    }

    [[nodiscard]] bool is_output_truthy(size_t) const override { return false; }
};

class Module : public stl::Module {
    std::shared_ptr<runtime::state::State> state;

public:
    explicit Module(std::shared_ptr<runtime::state::State> state):
        state(std::move(state)) {}

    [[nodiscard]] std::string module_name() const override { return "control"; }

    bool handles(const std::string &node_type) const override {
        return node_type == "set_authority";
    }

    std::pair<std::unique_ptr<runtime::node::Node>, x::errors::Error>
    create(runtime::node::Config &&cfg) override {
        if (!this->handles(cfg.node.type)) return {nullptr, x::errors::NOT_FOUND};
        const auto &auth_param = cfg.node.inputs["value"];
        if (!types::to_sample_value(auth_param.value, auth_param.type).has_value())
            return {
                nullptr,
                x::errors::Error(
                    x::errors::VALIDATION,
                    "set_authority node missing required value parameter"
                )
            };
        return {
            std::make_unique<SetAuthority>(std::move(cfg.state), *this->state),
            x::errors::NIL
        };
    }
};

}
