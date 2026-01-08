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
#include <vector>

#include "x/cpp/errors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::node {
struct Config {
    ir::Node node;
    state::Node state;
};

class Factory {
public:
    virtual ~Factory() = default;

    virtual std::pair<std::unique_ptr<Node>, x::errors::Error>
    create(const Config &cfg) = 0;
};

class MultiFactory : public Factory {
    std::vector<std::shared_ptr<Factory>> factories;

public:
    explicit MultiFactory(std::vector<std::shared_ptr<Factory>> factories):
        factories(std::move(factories)) {}

    std::pair<std::unique_ptr<Node>, x::errors::Error>
    create(const Config &cfg) override {
        for (const auto &factory: this->factories) {
            auto [node, err] = factory->create(cfg);
            if (!err) return {std::move(node), x::errors::NIL};
            if (x::errors::NOT_FOUND.matches(err)) continue;
            return {
                nullptr,
                x::errors::Error(
                    err,
                    err.data + " (while creating node '" + cfg.node.key +
                        "' of type '" + cfg.node.type + "')"
                )
            };
        }

        return {
            nullptr,
            x::errors::Error(
                x::errors::NOT_FOUND,
                "No factory registered for node type '" + cfg.node.type +
                    "' (node: " + cfg.node.key + ")"
            )
        };
    }
};
}
