// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include "arc/cpp/runtime/factory/factory.h"
#include "arc/cpp/runtime/wasm/node.h"

namespace arc::wasm {
class Factory : public NodeFactory {
    std::shared_ptr<Module> mod;

public:
    explicit Factory(std::shared_ptr<Module> &mod): mod(mod) {}

    std::pair<std::unique_ptr<arc::Node>, xerrors::Error>
    create(const NodeConfig &cfg) override {
        auto [func, err] = mod->func(cfg.node.type);
        if (err) return {nullptr, err};
        return {std::make_unique<Node>(cfg.node, cfg.state, func), xerrors::NIL};
    }
};
}
