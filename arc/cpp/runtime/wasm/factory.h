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
    std::shared_ptr<Module> mod;

public:
    explicit Factory(std::shared_ptr<Module> &mod): mod(mod) {}

    std::pair<std::unique_ptr<node::Node>, x::errors::Error>
    create(const node::Config &cfg) override {
        auto [func, err] = mod->func(cfg.node.type);
        if (err) return {nullptr, err};
        return {std::make_unique<Node>(cfg.node, cfg.state, func), x::errors::NIL};
    }
};
}
