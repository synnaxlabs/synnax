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

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::stage {

/// @brief StageEntry is a node that triggers stage transitions when it receives
/// an activation signal (input value of u8(1)).
class StageEntry : public node::Node {
public:
    x::errors::Error next(node::Context &ctx) override {
        // Entry nodes only execute when the scheduler's mark_changed() adds them
        // to the changed set. mark_changed() already validates is_output_truthy()
        // on the upstream node for one-shot edges, so no input check is needed here.
        ctx.activate_stage();
        return x::errors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        return false;
    }
};

/// @brief factory creates StageEntry nodes for "stage_entry" type nodes in the IR.
class Factory : public node::Factory {
public:
    bool handles(const std::string &node_type) const override {
        return node_type == "stage_entry";
    }

    std::pair<std::unique_ptr<node::Node>, x::errors::Error>
    create(node::Config &&cfg) override {
        return {std::make_unique<StageEntry>(), x::errors::NIL};
    }
};

}
