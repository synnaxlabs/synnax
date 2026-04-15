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
    const ir::IR &prog;
    ir::Node node;
    state::Node state;

    Config(const ir::IR &prog, ir::Node node, state::Node &&state):
        prog(prog), node(std::move(node)), state(std::move(state)) {}

    Config(Config &&) = default;
    Config &operator=(Config &&) = delete;
    Config(const Config &) = delete;
    Config &operator=(const Config &) = delete;
};

class Factory {
public:
    virtual ~Factory() = default;

    /// @brief returns the module name this factory belongs to (e.g. "authority",
    /// "status"). When the MultiFactory encounters a qualified node type like
    /// "status.set", it skips factories whose module name doesn't match the
    /// prefix. Factories that return an empty string are always considered.
    [[nodiscard]] virtual std::string module_name() const { return ""; }

    virtual bool handles(const std::string &node_type) const { return false; }

    virtual std::pair<std::unique_ptr<Node>, x::errors::Error> create(Config &&cfg) = 0;
};

class MultiFactory : public Factory {
    std::vector<std::shared_ptr<Factory>> factories;

public:
    explicit MultiFactory(std::vector<std::shared_ptr<Factory>> factories):
        factories(std::move(factories)) {}

    std::pair<std::unique_ptr<Node>, x::errors::Error> create(Config &&cfg) override {
        const auto node_key = cfg.node.key;
        // Strip module prefix from the node type so factories only match bare
        // names. The compiler emits qualified names (e.g. "time.interval",
        // "authority.set") into the IR; normalizing here keeps prefix awareness
        // out of individual factories.
        std::string module_prefix;
        if (auto dot = cfg.node.type.rfind('.'); dot != std::string::npos) {
            module_prefix = cfg.node.type.substr(0, dot);
            cfg.node.type = cfg.node.type.substr(dot + 1);
        }
        const auto node_type = cfg.node.type;
        for (const auto &factory: this->factories) {
            // When the IR node has a module prefix (e.g. "status" from
            // "status.set"), skip factories belonging to a different module.
            // Factories that don't declare a module name are always considered.
            if (!module_prefix.empty()) {
                const auto mod = factory->module_name();
                if (!mod.empty() && mod != module_prefix) continue;
            }
            if (!factory->handles(node_type)) continue;
            auto [node, err] = factory->create(std::move(cfg));
            if (err) {
                return {
                    nullptr,
                    x::errors::Error(
                        err,
                        err.data + " (while creating node '" + node_key +
                            "' of type '" + node_type + "')"
                    )
                };
            }
            return {std::move(node), x::errors::NIL};
        }
        return {
            nullptr,
            x::errors::Error(
                x::errors::NOT_FOUND,
                "No factory registered for node type '" + node_type +
                    "' (node: " + node_key + ")"
            )
        };
    }
};
}
