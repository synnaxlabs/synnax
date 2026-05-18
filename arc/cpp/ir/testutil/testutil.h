// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "arc/cpp/ir/ir.h"

namespace arc::ir::testutil {

/// @brief describes a nested Scope for use with Builder::sequence. Each
/// ScopeSpec becomes one step of the sequential scope appended by
/// Builder::sequence. Exactly one of strata or steps should be non-empty:
/// strata produces a parallel+gated child scope; steps produces a
/// sequential+gated child scope.
struct ScopeSpec {
    /// @brief key is the member key and the nested scope's own key.
    std::string key;
    /// @brief strata is the stratum layering of a parallel nested scope. Each
    /// inner vector is a stratum of node keys with no dependency among them;
    /// stratum N depends only on strata 0..N-1.
    std::vector<std::vector<std::string>> strata;
    /// @brief steps is the ordered list of node keys for a sequential
    /// nested scope. Mutually exclusive with strata.
    std::vector<std::string> steps;
};

/// @brief fluent builder for constructing IR in tests. Produces the new
/// unified Scope IR: Root is a parallel+always-live scope; Strata sets its
/// stratum layering; Sequence appends a nested sequential scope member.
///
/// Example usage:
/// @code
/// auto program = Builder()
///     .node("A")
///     .node("B")
///     .edge("A", "output", "B", "input")
///     .strata({{"A"}, {"B"}})
///     .build();
/// @endcode
class Builder {
    IR prog;

public:
    Builder() {
        this->prog.root.mode = ScopeMode::Parallel;
        this->prog.root.liveness = Liveness::Always;
    }

    /// @brief adds a minimal node keyed by key. Tests requiring richer node
    /// configuration should assemble the node directly.
    Builder &node(const std::string &key) {
        Node n;
        n.key = key;
        this->prog.nodes.push_back(std::move(n));
        return *this;
    }

    /// @brief adds a continuous edge from source.param to target.param.
    /// Continuous edges propagate changes every time the source output
    /// changes.
    Builder &edge(
        const std::string &source_node,
        const std::string &source_param,
        const std::string &target_node,
        const std::string &target_param
    ) {
        this->prog.edges.emplace_back(
            Handle{source_node, source_param},
            Handle{target_node, target_param},
            EdgeKind::Continuous
        );
        return *this;
    }

    /// @brief adds a conditional edge from source.param to target.param.
    /// Conditional edges only propagate when the source output is truthy.
    Builder &conditional(
        const std::string &source_node,
        const std::string &source_param,
        const std::string &target_node,
        const std::string &target_param
    ) {
        this->prog.edges.emplace_back(
            Handle{source_node, source_param},
            Handle{target_node, target_param},
            EdgeKind::Conditional
        );
        return *this;
    }

    /// @brief sets the Root scope's stratum layering. Each inner vector is
    /// a stratum of node keys with no data dependency among them; stratum
    /// N depends only on strata 0..N-1.
    Builder &strata(std::vector<std::vector<std::string>> strata_spec) {
        this->prog.root.strata.clear();
        this->prog.root.strata.reserve(strata_spec.size());
        for (auto &stratum_keys: strata_spec) {
            Members layer;
            layer.reserve(stratum_keys.size());
            for (auto &key: stratum_keys)
                layer.push_back(node_member(key));
            this->prog.root.strata.push_back(std::move(layer));
        }
        return *this;
    }

    /// @brief appends a sequential, gated nested Scope to the Root as a
    /// member of its final stratum (creating a stratum if the Root has none).
    /// Each ScopeSpec becomes one step of the sequential scope.
    Builder &sequence(const std::string &key, std::vector<ScopeSpec> specs) {
        Scope seq;
        seq.key = key;
        seq.mode = ScopeMode::Sequential;
        seq.liveness = Liveness::Gated;
        seq.steps.reserve(specs.size());
        for (auto &spec: specs)
            seq.steps.push_back(scope_member_from(std::move(spec)));

        if (prog.root.strata.empty()) prog.root.strata.emplace_back();
        prog.root.strata.back().push_back(scope_member(std::move(seq)));
        return *this;
    }

    /// @brief builds and returns the IR by move.
    IR build() { return std::move(this->prog); }

private:
    /// @brief builds a Member wrapping a nested Scope for ScopeSpec.
    /// ScopeSpec.steps (non-empty) takes priority and produces a sequential
    /// nested scope; otherwise ScopeSpec.strata produces a parallel nested
    /// scope.
    static Member scope_member_from(ScopeSpec spec) {
        Scope nested;
        nested.key = spec.key;
        nested.liveness = Liveness::Gated;
        if (!spec.steps.empty()) {
            nested.mode = ScopeMode::Sequential;
            nested.steps.reserve(spec.steps.size());
            for (auto &k: spec.steps)
                nested.steps.push_back(node_member(k));
        } else {
            nested.mode = ScopeMode::Parallel;
            nested.strata.reserve(spec.strata.size());
            for (auto &stratum_keys: spec.strata) {
                Members layer;
                layer.reserve(stratum_keys.size());
                for (auto &k: stratum_keys)
                    layer.push_back(node_member(k));
                nested.strata.push_back(std::move(layer));
            }
        }
        return scope_member(std::move(nested));
    }
};

}
