// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "arc/cpp/runtime/node/factory.h"
#include "arc/cpp/runtime/wasm/node.h"

namespace arc::runtime::wasm {
class Factory : public node::Factory {
    mutable std::shared_ptr<Module> mod;

public:
    explicit Factory(std::shared_ptr<Module> &mod): mod(mod) {}

    bool handles(const std::string &node_type) const override {
        return this->mod->has_func(node_type);
    }

    std::pair<std::unique_ptr<node::Node>, xerrors::Error>
    create(node::Config &&cfg) override {
        auto [func, err] = this->mod->func(cfg.node.type);
        if (err) return {nullptr, err};
        return {
            std::make_unique<Node>(cfg.node, std::move(cfg.state), func),
            xerrors::NIL
        };
    }
};
}
