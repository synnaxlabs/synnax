
// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <optional>
#include <string>
#include <vector>

#include "x/cpp/telem/series.h"
#include "x/cpp/telem/telem.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/state/state.h"
#include "arc/cpp/types/types.h"

namespace arc::runtime::node {

/// @brief ResolvedInputs reads a node's inputs, handling both static brace values
/// and edge-fed expression values. Mirrors the Go runtime/node.ResolvedInputs.
class ResolvedInputs {
    /// @brief node is the IR node whose inputs are resolved.
    ir::Node node;

    struct EdgeInput {
        /// @brief name is the edge-fed input's parameter name.
        std::string name;
        /// @brief idx is the input's position in node.inputs, which matches its
        /// index in the runtime state's aligned input buffers.
        size_t idx;
    };

    /// @brief edges lists inputs supplied by an upstream edge rather than a literal.
    std::vector<EdgeInput> edges;

public:
    ResolvedInputs() = default;

    /// @brief resolve marks an input edge-fed when its value is null and it is not
    /// a channel-typed param, leaving the rest static.
    static ResolvedInputs resolve(const ir::Node &node) {
        ResolvedInputs ri;
        ri.node = node;
        for (size_t i = 0; i < node.inputs.size(); i++) {
            const auto &p = node.inputs[i];
            if (!p.value.is_null() || p.type.kind == ::arc::types::Kind::Chan) continue;
            ri.edges.push_back(EdgeInput{p.name, i});
        }
        return ri;
    }

    /// @brief has_edges reports whether any input is edge-fed.
    [[nodiscard]] bool has_edges() const { return !this->edges.empty(); }

    /// @brief value_of returns the latest edge value if edge-fed (call after
    /// refresh_inputs), else the static value, or nullopt if absent/unfed.
    [[nodiscard]] std::optional<x::telem::SampleValue>
    value_of(const state::Node &state, const std::string &name) const {
        for (const auto &e: this->edges) {
            if (e.name != name) continue;
            const auto &s = state.input(e.idx);
            if (s->size() == 0) return std::nullopt;
            return s->at(static_cast<int>(s->size() - 1));
        }
        for (const auto &p: this->node.inputs)
            if (p.name == name) return types::to_sample_value(p.value, p.type);
        return std::nullopt;
    }

    /// @brief string_of resolves the named input and extracts it as a string,
    /// returning "" when the input is absent, not yet fed, or not a string.
    [[nodiscard]] std::string
    string_of(const state::Node &state, const std::string &name) const {
        const auto sv = this->value_of(state, name);
        if (!sv.has_value()) return "";
        const auto *s = std::get_if<std::string>(&*sv);
        return s != nullptr ? *s : "";
    }
};
}
