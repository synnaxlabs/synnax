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

    /// @brief returns the module name this factory belongs to (e.g. "control",
    /// "time"). When the MultiFactory encounters a qualified node type like
    /// "time.interval", it skips factories whose module name doesn't match the
    /// prefix. Factories that return an empty string are always considered.
    [[nodiscard]] virtual std::string module_name() const { return ""; }

    [[nodiscard]] virtual bool handles(const std::string &node_type) const {
        return false;
    }

    virtual std::pair<std::unique_ptr<Node>, x::errors::Error> create(Config &&cfg) = 0;
};

class MultiFactory : public Factory {
    std::vector<std::shared_ptr<Factory>> factories;

public:
    explicit MultiFactory(std::vector<std::shared_ptr<Factory>> factories):
        factories(std::move(factories)) {}

    std::pair<std::unique_ptr<Node>, x::errors::Error> create(Config &&cfg) override {
        // A program function is keyed by its qualified name, so the full type is
        // tried first. Only then is the module prefix split off for the host
        // factories.
        if (auto *factory = this->find(cfg.node.type, ""); factory != nullptr)
            return this->create_with(*factory, std::move(cfg));
        const auto dot = cfg.node.type.rfind('.');
        if (dot != std::string::npos) {
            const auto module_prefix = cfg.node.type.substr(0, dot);
            cfg.node.type = cfg.node.type.substr(dot + 1);
            if (auto *factory = this->find(cfg.node.type, module_prefix);
                factory != nullptr)
                return this->create_with(*factory, std::move(cfg));
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

private:
    /// @brief returns the first factory that handles node_type, skipping factories
    /// that name a module other than module_prefix when one is given.
    [[nodiscard]] Factory *
    find(const std::string &node_type, const std::string &module_prefix) const {
        for (const auto &factory: this->factories) {
            if (!module_prefix.empty()) {
                const auto mod = factory->module_name();
                if (!mod.empty() && mod != module_prefix) continue;
            }
            if (factory->handles(node_type)) return factory.get();
        }
        return nullptr;
    }

    static std::pair<std::unique_ptr<Node>, x::errors::Error>
    create_with(Factory &factory, Config &&cfg) {
        const auto node_key = cfg.node.key;
        const auto node_type = cfg.node.type;
        auto [node, err] = factory.create(std::move(cfg));
        if (err)
            return {
                nullptr,
                x::errors::Error(
                    err,
                    err.data + " (while creating node '" + node_key + "' of type '" +
                        node_type + "')"
                )
            };
        return {std::move(node), x::errors::NIL};
    }
};
}
