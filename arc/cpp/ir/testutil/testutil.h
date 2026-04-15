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
/// ScopeSpec becomes one Member of the sequential scope appended by
/// Builder::sequence. Exactly one of phases or members should be non-empty:
/// phases produces a parallel+gated child scope; members produces a
/// sequential+gated child scope.
struct ScopeSpec {
    /// @brief key is the member key and the nested scope's own key.
    std::string key;
    /// @brief phases is the phase layering of a parallel nested scope. Each
    /// inner vector is a phase of node keys with no dependency among them;
    /// phase N depends only on phases 0..N-1.
    std::vector<std::vector<std::string>> phases;
    /// @brief members is the ordered list of node keys for a sequential
    /// nested scope. Mutually exclusive with phases.
    std::vector<std::string> members;
};

/// @brief fluent builder for constructing IR in tests. Produces the new
/// unified Scope IR: Root is a parallel+always-live scope; Phases sets its
/// phase layering; Sequence appends a nested sequential scope member.
///
/// Example usage:
/// @code
/// auto program = Builder()
///     .node("A")
///     .node("B")
///     .edge("A", "output", "B", "input")
///     .phases({{"A"}, {"B"}})
///     .build();
/// @endcode
class Builder {
    IR ir_;

public:
    Builder() {
        ir_.root.mode = ScopeMode::Parallel;
        ir_.root.liveness = Liveness::Always;
    }

    /// @brief adds a minimal node keyed by key. Tests requiring richer node
    /// configuration should assemble the node directly.
    Builder &node(const std::string &key) {
        Node n;
        n.key = key;
        ir_.nodes.push_back(std::move(n));
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
        ir_.edges.emplace_back(
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
        ir_.edges.emplace_back(
            Handle{source_node, source_param},
            Handle{target_node, target_param},
            EdgeKind::Conditional
        );
        return *this;
    }

    /// @brief sets the Root scope's phase layering. Each inner vector is a
    /// phase of node keys with no data dependency among them; phase N
    /// depends only on phases 0..N-1.
    Builder &phases(std::vector<std::vector<std::string>> phases_spec) {
        ir_.root.phases.clear();
        ir_.root.phases.reserve(phases_spec.size());
        for (auto &phase_keys: phases_spec) {
            Phase p;
            p.members.reserve(phase_keys.size());
            for (auto &key: phase_keys)
                p.members.push_back(node_member(key));
            ir_.root.phases.push_back(std::move(p));
        }
        return *this;
    }

    /// @brief appends a sequential, gated nested Scope to the Root as a
    /// member of its final phase (creating a phase if the Root has none).
    /// Each ScopeSpec becomes one Member of the sequential scope.
    Builder &sequence(const std::string &key, std::vector<ScopeSpec> specs) {
        Scope seq;
        seq.key = key;
        seq.mode = ScopeMode::Sequential;
        seq.liveness = Liveness::Gated;
        seq.members.reserve(specs.size());
        for (auto &spec: specs)
            seq.members.push_back(scope_member_from(std::move(spec)));

        if (ir_.root.phases.empty()) ir_.root.phases.emplace_back();
        auto &last_phase = ir_.root.phases.back();
        Member m;
        m.key = key;
        m.scope = x::mem::indirect<Scope>(std::move(seq));
        last_phase.members.push_back(std::move(m));
        return *this;
    }

    /// @brief builds and returns the IR by move.
    IR build() { return std::move(ir_); }

private:
    /// @brief constructs a Member wrapping a NodeRef keyed by node_key.
    static Member node_member(const std::string &node_key) {
        Member m;
        m.key = node_key;
        NodeRef ref;
        ref.key = node_key;
        m.node_ref = std::move(ref);
        return m;
    }

    /// @brief builds a Member wrapping a nested Scope for ScopeSpec.
    /// ScopeSpec.members (non-empty) takes priority and produces a
    /// sequential nested scope; otherwise ScopeSpec.phases produces a
    /// parallel nested scope.
    static Member scope_member_from(ScopeSpec spec) {
        Scope nested;
        nested.key = spec.key;
        nested.liveness = Liveness::Gated;
        if (!spec.members.empty()) {
            nested.mode = ScopeMode::Sequential;
            nested.members.reserve(spec.members.size());
            for (auto &k: spec.members)
                nested.members.push_back(node_member(k));
        } else {
            nested.mode = ScopeMode::Parallel;
            nested.phases.reserve(spec.phases.size());
            for (auto &phase_keys: spec.phases) {
                Phase p;
                p.members.reserve(phase_keys.size());
                for (auto &k: phase_keys)
                    p.members.push_back(node_member(k));
                nested.phases.push_back(std::move(p));
            }
        }
        Member m;
        m.key = spec.key;
        m.scope = x::mem::indirect<Scope>(std::move(nested));
        return m;
    }
};

}
