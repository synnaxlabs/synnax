// Copyright 2025 Synnax Labs, Inc.
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

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::stage {

/// @brief StageEntry is a node that triggers stage transitions when it receives
/// an activation signal (input value of u8(1)).
class StageEntry : public node::Node {
public:
    xerrors::Error next(node::Context &ctx) override {
        // Entry nodes only execute when the scheduler's mark_changed() adds them
        // to the changed set. mark_changed() already validates is_output_truthy()
        // on the upstream node for one-shot edges, so no input check is needed here.
        ctx.activate_stage();
        return xerrors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        return false;
    }
};

/// @brief factory creates StageEntry nodes for "stage_entry" type nodes in the IR.
class Factory : public node::Factory {
public:
    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(const node::Config &cfg) override {
        if (cfg.node.type != "stage_entry") return {nullptr, xerrors::NOT_FOUND};
        return {std::make_unique<StageEntry>(), xerrors::NIL};
    }
};

}
