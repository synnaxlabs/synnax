// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "arc/cpp/ir/format.h"
#include "arc/cpp/ir/json.gen.h"
#include "arc/cpp/ir/proto.gen.h"
#include "arc/cpp/ir/types.gen.h"

namespace arc::ir {
inline const std::string default_output_param = "output";
inline const std::string default_input_param = "input";
inline const std::string lhs_input_param = "a";
inline const std::string rhs_input_param = "b";

/// @brief returns the position of the input param named name, or NOT_FOUND if the
/// node has no such param. Called at construction so an unknown name fails at load.
[[nodiscard]] inline std::pair<size_t, x::errors::Error>
resolve_input(const Node &node, const std::string &name) {
    for (size_t i = 0; i < node.inputs.size(); ++i)
        if (node.inputs[i].name == name) return {i, x::errors::NIL};
    return {
        0,
        x::errors::Error(x::errors::NOT_FOUND, "node has no input named " + name)
    };
}

/// @brief one flag per input of node: true when an edge feeds it (edge-fed, the edge
/// wins over any default), false otherwise (literal-fed, the default holds).
[[nodiscard]] inline std::vector<bool> edge_fed_mask(const IR &prog, const Node &node) {
    std::vector<bool> mask(node.inputs.size());
    for (size_t i = 0; i < node.inputs.size(); i++)
        mask[i] = prog.edge_to(Handle(node.key, node.inputs[i].name)).has_value();
    return mask;
}

inline bool operator==(const Handle &lhs, const Handle &rhs) {
    return lhs.node == rhs.node && lhs.param == rhs.param;
}

inline bool operator==(const Edge &lhs, const Edge &rhs) {
    return lhs.source == rhs.source && lhs.target == rhs.target && lhs.kind == rhs.kind;
}

/// @brief builds a leaf Member referencing the node with the given key.
inline Member node_member(std::string key) {
    Member m;
    m.node_key = std::move(key);
    return m;
}

/// @brief builds a Member wrapping the given nested Scope.
inline Member scope_member(Scope s) {
    Member m;
    m.scope = x::mem::indirect<Scope>(std::move(s));
    return m;
}
}

template<>
struct std::hash<arc::ir::Handle> {
    size_t operator()(const arc::ir::Handle &h) const noexcept {
        return std::hash<std::string>{}(h.node + h.param);
    }
};

template<>
struct std::hash<arc::ir::Edge> {
    size_t operator()(const arc::ir::Edge &e) const noexcept {
        return std::hash<arc::ir::Handle>{}(e.source) ^
               std::hash<arc::ir::Handle>{}(e.target) << 1 ^
               std::hash<int>{}(static_cast<int>(e.kind)) << 2;
    }
};
