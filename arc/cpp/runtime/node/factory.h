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

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc::runtime::node {
struct Config {
    ir::Node node;
    state::Node state;

    Config(ir::Node node, state::Node &&state):
        node(std::move(node)), state(std::move(state)) {}

    Config(Config &&) = default;
    Config &operator=(Config &&) = delete;
    Config(const Config &) = delete;
    Config &operator=(const Config &) = delete;
};

class Factory {
public:
    virtual ~Factory() = default;

    virtual bool handles(const std::string &node_type) const { return false; }

    virtual std::pair<std::unique_ptr<Node>, xerrors::Error> create(Config &&cfg) = 0;
};

class MultiFactory : public Factory {
    std::vector<std::shared_ptr<Factory>> factories;

public:
    explicit MultiFactory(std::vector<std::shared_ptr<Factory>> factories):
        factories(std::move(factories)) {}

    std::pair<std::unique_ptr<Node>, xerrors::Error> create(Config &&cfg) override {
        const auto node_key = cfg.node.key;
        const auto node_type = cfg.node.type;
        for (const auto &factory: this->factories) {
            if (!factory->handles(node_type)) continue;
            auto [node, err] = factory->create(std::move(cfg));
            if (err) {
                return {
                    nullptr,
                    xerrors::Error(
                        err,
                        err.data + " (while creating node '" + node_key +
                            "' of type '" + node_type + "')"
                    )
                };
            }
            return {std::move(node), xerrors::NIL};
        }
        return {
            nullptr,
            xerrors::Error(
                xerrors::NOT_FOUND,
                "No factory registered for node type '" + node_type +
                    "' (node: " + node_key + ")"
            )
        };
    }
};
}
