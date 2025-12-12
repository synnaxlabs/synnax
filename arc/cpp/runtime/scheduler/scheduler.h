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

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/node/node.h"
#include "arc/cpp/runtime/time/time.h"

namespace arc::runtime::scheduler {

/// Helper to create a combined key for sequence+stage lookup.
inline std::string stage_key(const std::string &seq, const std::string &stage) {
    return seq + "_" + stage;
}

class Scheduler {
    ir::Strata strata;
    std::unordered_set<std::string> changed;
    struct NodeState {
        std::string key;
        std::unique_ptr<node::Node> node;
        std::vector<ir::Edge> output_edges;
    };
    std::unordered_map<std::string, NodeState> nodes;
    NodeState *current_state;
    node::Context ctx;

    // Stage management
    std::string active_sequence;
    std::string active_stage;
    /// Maps "sequence_stage" -> list of node keys in that stage
    std::unordered_map<std::string, std::vector<std::string>> stage_to_nodes;
    /// Set of node keys that are currently active (in the active stage)
    std::unordered_set<std::string> active_node_keys;
    /// Set of all nodes that belong to any stage (for filtering)
    std::unordered_set<std::string> staged_nodes;

    void mark_changed(const std::string &param) {
        for (const auto &edge: current_state->output_edges)
            if (edge.source.param == param) this->changed.insert(edge.target.node);
    }

    /// Check if a node should be executed based on stage filtering.
    /// A node executes if:
    /// 1. No sequences are defined (no stage filtering), OR
    /// 2. The node is NOT part of any stage (always runs), OR
    /// 3. The node is in the currently active stage
    [[nodiscard]] bool should_execute_node(const std::string &node_key) const {
        // If no stage filtering is active, run all nodes
        if (stage_to_nodes.empty()) return true;

        // If the node is not part of any stage, always run it
        if (!staged_nodes.contains(node_key)) return true;

        // Otherwise, only run if in the active stage
        return active_node_keys.contains(node_key);
    }

public:
    Scheduler(
        const ir::IR &prog,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &nodes
    ):
        strata(prog.strata), current_state() {
        for (auto &[key, node]: nodes)
            this->nodes[key] = NodeState{
                .key = key,
                .node = std::move(node),
                .output_edges = prog.outgoing_edges(key)
            };
        this->ctx = node::Context{
            .mark_changed =
                [&](const std::string &param) { this->mark_changed(param); },
            .report_error = [&](const xerrors::Error &err) {}
        };

        // Build stage_to_nodes map from sequences
        load_sequences(prog.sequences);
    }

    /// Load sequence/stage information and build lookup maps.
    void load_sequences(const std::vector<ir::Sequence> &sequences) {
        for (const auto &seq : sequences) {
            for (const auto &stage : seq.stages) {
                std::string key = stage_key(seq.key, stage.key);
                stage_to_nodes[key] = stage.nodes;

                // Track all nodes that belong to any stage
                for (const auto &node_key : stage.nodes) {
                    staged_nodes.insert(node_key);
                }
            }
        }
    }

    /// Activate a specific stage within a sequence.
    /// Called by stage_entry nodes when they receive an activation signal.
    void activate_stage(const std::string &seq, const std::string &stage) {
        active_sequence = seq;
        active_stage = stage;

        // Update active nodes set
        active_node_keys.clear();
        std::string key = stage_key(seq, stage);
        auto it = stage_to_nodes.find(key);
        if (it != stage_to_nodes.end()) {
            for (const auto &node_key : it->second) {
                active_node_keys.insert(node_key);
            }
        }

        // Reset resettable nodes in this stage (e.g., wait timers)
        reset_stage_nodes(key);
    }

    /// Reset all resettable nodes in a stage (e.g., wait timers).
    void reset_stage_nodes(const std::string &key) {
        auto it = stage_to_nodes.find(key);
        if (it == stage_to_nodes.end()) return;

        for (const auto &node_key : it->second) {
            auto node_it = nodes.find(node_key);
            if (node_it == nodes.end()) continue;

            // Try to cast to Resettable and reset
            auto *resettable = dynamic_cast<time::Resettable *>(
                node_it->second.node.get()
            );
            if (resettable != nullptr) {
                resettable->reset();
            }
        }
    }

    /// Get the currently active sequence name.
    [[nodiscard]] const std::string &get_active_sequence() const {
        return active_sequence;
    }

    /// Get the currently active stage name.
    [[nodiscard]] const std::string &get_active_stage() const {
        return active_stage;
    }

    void next(const telem::TimeSpan elapsed) {
        this->ctx.elapsed = elapsed;
        bool first = true;
        for (const auto& stratum: this->strata.strata) {
            for (const auto& node_key: stratum) {
                // Skip nodes not in active stage
                if (!should_execute_node(node_key)) continue;

                if (first || this->changed.contains(node_key)) {
                    const auto n = &this->nodes[node_key];
                    this->current_state = n;
                    this->current_state->node->next(this->ctx);
                }
            }
            first = false;
        }
        this->changed.clear();
    }
};
}
