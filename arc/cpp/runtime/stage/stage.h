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
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::stage {

/// StageEntry is a node that triggers stage transitions when it receives
/// an activation signal (input value of u8(1)).
class StageEntry : public node::Node {
    std::string node_key;
    state::Node state;

public:
    StageEntry(std::string node_key, state::Node state)
        : node_key(std::move(node_key)), state(std::move(state)) {}

    xerrors::Error next(node::Context &ctx) override {
        if (!state.refresh_inputs()) return xerrors::NIL;

        auto &input = state.input(0);
        if (input->size() == 0) return xerrors::NIL;

        // Activation signal is a u8 with value 1
        if (input->at<std::uint8_t>(0) == 1) {
            ctx.activate_stage(node_key);
        }
        return xerrors::NIL;
    }
};

/// Factory creates StageEntry nodes for "stage_entry" type nodes in the IR.
class Factory : public node::Factory {
public:
    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "stage_entry") return {nullptr, xerrors::NOT_FOUND};
        return {
            std::make_unique<StageEntry>(cfg.node.key, cfg.state),
            xerrors::NIL
        };
    }
};

}
