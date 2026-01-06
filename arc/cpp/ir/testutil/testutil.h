// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>
#include <vector>

#include "arc/cpp/ir/ir.h"

namespace arc::ir::testutil {

/// @brief Fluent builder for constructing IR in tests.
/// Avoids verbose protobuf construction for simple test graphs.
///
/// Example usage:
/// @code
/// auto ir = IRBuilder()
///     .node("A")
///     .node("B")
///     .edge("A", "output", "B", "input")
///     .strata({{"A"}, {"B"}})
///     .build();
/// @endcode
class Builder {
    IR ir_;

public:
    /// @brief Add a node with given key.
    /// Creates a minimal node with just the key set.
    Builder &node(const std::string &key) {
        Node n;
        n.key = key;
        ir_.nodes.push_back(std::move(n));
        return *this;
    }

    /// @brief Add a continuous edge: source.param -> target.param
    /// Continuous edges propagate changes every time the source output changes.
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

    /// @brief Add a one-shot edge: source.param => target.param
    /// One-shot edges only fire when the source output is truthy,
    /// and only once per stage activation.
    Builder &oneshot(
        const std::string &source_node,
        const std::string &source_param,
        const std::string &target_node,
        const std::string &target_param
    ) {
        ir_.edges.emplace_back(
            Handle{source_node, source_param},
            Handle{target_node, target_param},
            EdgeKind::OneShot
        );
        return *this;
    }

    /// @brief Set global strata (topological execution order for non-staged nodes).
    /// Each inner vector is a stratum; nodes in the same stratum are independent.
    Builder &strata(std::vector<std::vector<std::string>> s) {
        ir_.strata = Strata(std::move(s));
        return *this;
    }

    /// @brief Add a sequence with stages.
    /// @param key The sequence key.
    /// @param stages Vector of {stage_key, stage_strata} pairs.
    ///
    /// Example:
    /// @code
    /// .sequence("my_seq", {
    ///     {"stage_a", {{"A"}, {"B"}}},  // stage_a has A in stratum 0, B in stratum 1
    ///     {"stage_b", {{"C"}}}           // stage_b has just C
    /// })
    /// @endcode
    Builder &sequence(
        const std::string &key,
        std::vector<std::pair<std::string, std::vector<std::vector<std::string>>>>
            stages
    ) {
        Sequence seq;
        seq.key = key;
        for (auto &[stage_key, stage_strata]: stages) {
            Stage stage;
            stage.key = stage_key;
            stage.strata = Strata(std::move(stage_strata));
            // Collect all node keys from strata for the nodes list
            for (const auto &stratum: stage.strata)
                for (const auto &node_key: stratum)
                    stage.nodes.push_back(node_key);
            seq.stages.push_back(std::move(stage));
        }
        ir_.sequences.push_back(std::move(seq));
        return *this;
    }

    /// @brief Build and return the IR (moves ownership).
    IR build() { return std::move(ir_); }
};
}
