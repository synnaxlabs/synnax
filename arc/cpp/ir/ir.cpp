// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <stdexcept>

#include "arc/cpp/ir/ir.h"

namespace arc::ir {

const Node &IR::node(const std::string &key) const {
    for (const auto &n: nodes)
        if (n.key == key) return n;
    throw std::runtime_error("node not found: " + key);
}

const Function &IR::function(const std::string &key) const {
    for (const auto &f: functions)
        if (f.key == key) return f;
    throw std::runtime_error("function not found: " + key);
}

std::optional<Edge> IR::edge_to(const Handle &target) const {
    for (const auto &e: edges)
        if (e.target == target) return e;
    return std::nullopt;
}

std::unordered_map<std::string, std::vector<Edge>>
IR::edges_from(const std::string &node_key) const {
    std::unordered_map<std::string, std::vector<Edge>> result;
    for (const auto &e: edges)
        if (e.source.node == node_key) result[e.source.param].push_back(e);
    return result;
}

}
