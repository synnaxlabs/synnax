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

/// ActivateCallback is called when a stage_entry node receives an activation signal.
/// Parameters are (sequence_name, stage_name).
using ActivateCallback = std::function<void(const std::string&, const std::string&)>;

/// SharedCallback allows callbacks to be set after node creation.
/// The nodes store a shared_ptr to this, and the callback can be set later.
struct SharedCallback {
    ActivateCallback callback;

    void invoke(const std::string& seq, const std::string& stage) {
        if (callback) callback(seq, stage);
    }
};

/// StageEntry is a special node type that represents a stage entry point.
/// When it receives an activation signal (input value of 1), it calls the
/// activate_stage callback to transition the runtime to this stage.
class StageEntry : public node::Node {
    state::Node state;
    std::string sequence_name;
    std::string stage_name;
    std::shared_ptr<SharedCallback> shared_callback;

public:
    StageEntry(
        std::string sequence_name,
        std::string stage_name,
        std::shared_ptr<SharedCallback> shared_callback,
        state::Node state
    ):
        state(std::move(state)),
        sequence_name(std::move(sequence_name)),
        stage_name(std::move(stage_name)),
        shared_callback(std::move(shared_callback)) {}

    xerrors::Error next(node::Context &ctx) override {
        // Check if we have an activation signal
        if (!state.refresh_inputs()) return xerrors::NIL;

        auto &input = state.input(0);
        if (input->size() == 0) return xerrors::NIL;

        // Activation signal is a u8 with value 1
        auto signal = input->at(0).get<std::uint8_t>();
        if (signal == 1 && shared_callback) {
            shared_callback->invoke(sequence_name, stage_name);
        }

        return xerrors::NIL;
    }
};

/// Factory creates StageEntry nodes for "stage_entry" type nodes in the IR.
class Factory : public node::Factory {
public:
    /// Shared callback that can be set after node creation.
    std::shared_ptr<SharedCallback> shared_callback = std::make_shared<SharedCallback>();

    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "stage_entry") return {nullptr, xerrors::NOT_FOUND};

        auto sequence_param = cfg.node.config.get("sequence");
        auto stage_param = cfg.node.config.get("stage");

        if (sequence_param == nullptr || stage_param == nullptr) {
            return {nullptr, xerrors::Error("stage_entry node missing sequence or stage config")};
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
