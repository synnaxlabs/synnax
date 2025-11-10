// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/factory/factory.h"

namespace arc {
void MultiFactory::add(std::unique_ptr<NodeFactory> factory) {
    factories_.push_back(std::move(factory));
}

std::pair<std::unique_ptr<Node>, xerrors::Error>
MultiFactory::create(const NodeFactoryConfig &cfg) {
    for (auto &factory: factories_) {
        auto [node, err] = factory->create(cfg);
        if (!err) return {std::move(node), xerrors::NIL};
        if (xerrors::NOT_FOUND.matches(err)) continue;
        return {
            nullptr,
            xerrors::Error(
                err,
                err.data + " (while creating node '" + cfg.ir_node.key + "' of type '" +
                    cfg.ir_node.type + "')"
            )
        };
    }

    return {
        nullptr,
        xerrors::Error(
            xerrors::NOT_FOUND,
            "No factory registered for node type '" + cfg.ir_node.type +
                "' (node: " + cfg.ir_node.key + ")"
        )
    };
}
}
