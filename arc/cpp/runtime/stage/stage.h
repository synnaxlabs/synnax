// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <string>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::stage {
/// StageEntry is a special node type that represents a stage entry point.
/// When it receives an activation signal (input value of 1), it calls the
/// activate_stage callback to transition the runtime to this stage.
class StageEntry : public node::Node {
    state::Node state;
public:
    explicit StageEntry(state::Node state):
        state(std::move(state)) {}

    xerrors::Error next(node::Context &ctx) override {
        // Check if we have an activation signal
        if (!state.refresh_inputs()) return xerrors::NIL;

        auto &input = state.input(0);
        if (input->size() == 0) return xerrors::NIL;

        const auto signal = input->at<std::uint8_t>(0);
        if (signal == 1) ctx.activate(this->state)
        return xerrors::NIL;
    }
};

/// Factory creates StageEntry nodes for "stage_entry" type nodes in the IR.
class Factory : public node::Factory {
public:
    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "stage_entry") return {nullptr, xerrors::NOT_FOUND};
        auto sequence_param = cfg.node.config.get("sequence");
        auto stage_param = cfg.node.config.get("stage");
        if (sequence_param == nullptr || stage_param == nullptr) {
            return {
                nullptr,
                xerrors::Error("stage_entry node missing sequence or stage config")
            };
        }

        auto sequence_name = sequence_param->value.get<std::string>();
        auto stage_name = stage_param->value.get<std::string>();

        auto node = std::make_unique<StageEntry>(
            sequence_name,
            stage_name,
            shared_callback,
            cfg.state
        );

        return {std::move(node), xerrors::NIL};
    }

    /// Set the callback after scheduler is created.
    void set_activate_callback(ActivateCallback callback) {
        shared_callback->callback = std::move(callback);
    }
};

}
