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

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/stl/stl.h"

namespace arc::stl::stage {

/// @brief StageEntry is a node that triggers stage transitions when it receives
/// an activation signal (input value of u8(1)).
class StageEntry : public runtime::node::Node {
public:
    xerrors::Error next(runtime::node::Context &ctx) override {
        ctx.activate_stage();
        return xerrors::NIL;
    }

    [[nodiscard]] bool is_output_truthy(const std::string &param) const override {
        return false;
    }
};

class Module : public stl::Module {
public:
    std::shared_ptr<runtime::node::Factory> factory() override {
        return std::make_shared<StageFactory>();
    }

private:
    class StageFactory : public runtime::node::Factory {
    public:
        bool handles(const std::string &node_type) const override {
            return node_type == "stage_entry";
        }

        std::pair<std::unique_ptr<runtime::node::Node>, xerrors::Error>
        create(runtime::node::Config &&cfg) override {
            return {std::make_unique<StageEntry>(), xerrors::NIL};
        }
    };
};

}
