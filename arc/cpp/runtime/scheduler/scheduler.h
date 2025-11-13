// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "x/cpp/defer/defer.h"
#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/core/node.h"
#include "arc/cpp/runtime/state/state.h"

namespace arc {
/// @brief Stratified scheduler for reactive Arc execution.
///
/// Implements Arc's stratified execution model:
/// - Stratum 0: Always executes (source nodes, channel readers)
/// - Stratum N: Executes only if marked as "changed" by upstream nodes
///
/// The scheduler maintains a pre-computed topological ordering (stratification)
/// and tracks which nodes need re-execution via a "changed" set.
class Scheduler {
    ir::Strata strata;
    std::unordered_set<std::string> changed;
    struct NodeState {
        std::string key;
        std::unique_ptr<Node> node;
        std::vector<ir::Edge> output_edges;
    };
    std::unordered_map<std::string, NodeState> nodes;
    NodeState *current_state;
    NodeContext ctx;

    void mark_changed(const std::string &param) {
        for (const auto &edge: current_state->output_edges)
            if (edge.source.param == param) this->changed.insert(edge.target.node);
    }

public:
    Scheduler(
        const ir::IR &prog,
        std::unordered_map<std::string, std::unique_ptr<Node>> &nodes
    ):
        strata(prog.strata), current_state() {
        for (auto &[key, node]: nodes)
            this->nodes[key] = NodeState{
                .key = key,
                .node = std::move(node),
                .output_edges = prog.outgoing_edges(key)
            };
        this->ctx = NodeContext{
            .mark_changed =
                [&](const std::string &param) { this->mark_changed(param); },
            .report_error = [&](const xerrors::Error &err) {}
        };
    }

    void next() {
        for (auto it = this->strata.strata.begin(); it != this->strata.strata.end();
             ++it) {
            for (auto node_it = it->begin(); node_it != it->end(); ++node_it) {
                if (it == this->strata.strata.begin() ||
                    this->changed.contains(*node_it)) {
                    const auto n = &this->nodes[*node_it];
                    this->current_state = n;
                    this->current_state->node->next(this->ctx);
                }
            }
        }
        this->changed.clear();
    }
};
}
