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

namespace arc::runtime::scheduler {

/// Helper to create a combined key for sequence+stage lookup.
inline std::string stage_key(const std::string &seq, const std::string &stage) {
    return seq + "_" + stage;
}

/// Identifies a stage within a sequence.
struct StageRef {
    std::string sequence;
    std::string stage;
};

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
    /// The IR sequences for terminal stage detection.
    std::vector<ir::Sequence> sequences;
    /// Maps sequence name -> currently active stage name.
    /// Multiple sequences can be active concurrently.
    std::unordered_map<std::string, std::string> active_stages;
    /// Maps "sequence_stage" -> list of node keys in that stage
    std::unordered_map<std::string, std::vector<std::string>> stage_to_nodes;
    /// Set of all nodes that belong to any stage (for filtering)
    std::unordered_set<std::string> staged_nodes;
    /// Maps node keys to their (sequence, stage) pair for reverse lookup.
    std::unordered_map<std::string, StageRef> node_to_stage;
    /// Tracks which one-shot edges have fired, keyed by sequence.
    /// Each sequence has its own set of fired edges, cleared when that sequence's stage
    /// changes.
    std::unordered_map<std::string, std::unordered_set<std::string>> fired_one_shots;

    void mark_changed(const std::string &param) {
        for (const auto &edge: current_state->output_edges) {
            if (edge.source.param != param) continue;

            // For one-shot edges, only propagate if not already fired
            if (edge.kind == ir::EdgeKind::OneShot) {
                std::string edge_key = edge.source.to_string() + "=>" +
                                       edge.target.to_string();
                // Determine which sequence this edge belongs to
                std::string seq_name;
                auto it = node_to_stage.find(current_state->key);
                if (it != node_to_stage.end()) { seq_name = it->second.sequence; }
                // Check if already fired
                auto &fired_set = fired_one_shots[seq_name];
                if (fired_set.contains(edge_key)) continue;
                fired_set.insert(edge_key);
            }
            this->changed.insert(edge.target.node);
        }
    }

    /// Looks up the stage that a node belongs to and activates it.
    void activate_stage_by_node(const std::string &node_key) {
        auto it = node_to_stage.find(node_key);
        if (it == node_to_stage.end()) return;
        activate_stage(it->second.sequence, it->second.stage);
    }

    /// Check if a node should be executed based on stage filtering.
    [[nodiscard]] bool should_execute_node(const std::string &node_key) const {
        // If no stage filtering is active, run all nodes
        if (stage_to_nodes.empty()) return true;

        // If the node is not part of any stage, always run it
        if (!staged_nodes.contains(node_key)) return true;

        // Check if node's sequence is active and in the correct stage
        auto it = node_to_stage.find(node_key);
        if (it == node_to_stage.end()) return false;

        auto stage_it = active_stages.find(it->second.sequence);
        if (stage_it == active_stages.end()) return false;

        return stage_it->second == it->second.stage;
    }

    /// Check terminal stages and deactivate sequences that have completed.
    void check_terminal_stages() {
        // Collect sequences to deactivate (can't modify while iterating)
        std::vector<std::string> to_deactivate;

        for (const auto &[seq_name, stage_name]: active_stages) {
            // Find the sequence
            const ir::Sequence *seq = nullptr;
            for (const auto &s: sequences) {
                if (s.key == seq_name) {
                    seq = &s;
                    break;
                }
            }
            if (seq == nullptr) continue;

            // Check if terminal (no next stage)
            if (seq->next_stage(stage_name) != nullptr) continue;

            // Check if all one-shot edges have fired
            if (stage_has_unfired_one_shots(seq_name, stage_name)) continue;

            to_deactivate.push_back(seq_name);
        }

        for (const auto &seq_name: to_deactivate) {
            deactivate_sequence(seq_name);
        }
    }

    /// Check if any node in the stage has unfired one-shot edges.
    [[nodiscard]] bool stage_has_unfired_one_shots(
        const std::string &seq_name,
        const std::string &stage_name
    ) const {
        std::string key = stage_key(seq_name, stage_name);
        auto it = stage_to_nodes.find(key);
        if (it == stage_to_nodes.end()) return false;

        auto fired_it = fired_one_shots.find(seq_name);
        const std::unordered_set<std::string> *fired_set = (fired_it !=
                                                            fired_one_shots.end())
                                                             ? &fired_it->second
                                                             : nullptr;

        for (const auto &node_key: it->second) {
            auto node_it = nodes.find(node_key);
            if (node_it == nodes.end()) continue;

            for (const auto &edge: node_it->second.output_edges) {
                if (edge.kind != ir::EdgeKind::OneShot) continue;
                std::string edge_key = edge.source.to_string() + "=>" +
                                       edge.target.to_string();
                if (fired_set == nullptr || !fired_set->contains(edge_key)) {
                    return true;
                }
            }
        }
        return false;
    }

public:
    Scheduler(
        const ir::IR &prog,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &nodes
    ):
        strata(prog.strata), current_state(nullptr), sequences(prog.sequences) {
        for (auto &[key, node]: nodes)
            this->nodes[key] = NodeState{
                .key = key,
                .node = std::move(node),
                .output_edges = prog.outgoing_edges(key)
            };
        this->ctx = node::Context{
            .mark_changed =
                [&](const std::string &param) { this->mark_changed(param); },
            .report_error = [&](const xerrors::Error &err) {},
            .activate_stage = [&](
                                  const std::string &node_key
                              ) { this->activate_stage_by_node(node_key); }
        };

        // Build stage_to_nodes map from sequences
        load_sequences(prog.sequences);
    }

    /// Load sequence/stage information and build lookup maps.
    void load_sequences(const std::vector<ir::Sequence> &seqs) {
        for (const auto &seq: seqs) {
            for (const auto &stage: seq.stages) {
                std::string key = stage_key(seq.key, stage.key);
                stage_to_nodes[key] = stage.nodes;

                // Track all nodes that belong to any stage and build reverse map
                for (const auto &node_key: stage.nodes) {
                    staged_nodes.insert(node_key);
                    node_to_stage[node_key] = StageRef{seq.key, stage.key};
                }
            }
        }
    }

    /// Activate a specific stage within a sequence.
    /// Multiple sequences can be active concurrently.
    void activate_stage(const std::string &seq, const std::string &stage) {
        active_stages[seq] = stage;
        reset_stage_nodes(seq, stage);
    }

    /// Deactivate a sequence, removing it from active sequences.
    void deactivate_sequence(const std::string &seq_name) {
        active_stages.erase(seq_name);
        fired_one_shots.erase(seq_name);
    }

    /// Reset all nodes in a stage and clear one-shot tracking for the sequence.
    void reset_stage_nodes(const std::string &seq_name, const std::string &stage_name) {
        // Clear one-shot tracking for this sequence
        fired_one_shots.erase(seq_name);

        std::string key = stage_key(seq_name, stage_name);
        auto it = stage_to_nodes.find(key);
        if (it == stage_to_nodes.end()) return;

        for (const auto &node_key: it->second) {
            auto node_it = nodes.find(node_key);
            if (node_it == nodes.end()) continue;
            node_it->second.node->reset();
        }
    }

    /// Get all currently active sequence names.
    [[nodiscard]] std::vector<std::string> get_active_sequences() const {
        std::vector<std::string> seqs;
        seqs.reserve(active_stages.size());
        for (const auto &[seq, _]: active_stages) {
            seqs.push_back(seq);
        }
        return seqs;
    }

    /// Get the currently active stage for a given sequence.
    /// Returns empty string if the sequence is not active.
    [[nodiscard]] std::string get_active_stage_for(const std::string &seq_name) const {
        auto it = active_stages.find(seq_name);
        if (it == active_stages.end()) return "";
        return it->second;
    }

    /// Check if a sequence is currently active.
    [[nodiscard]] bool is_sequence_active(const std::string &seq_name) const {
        return active_stages.contains(seq_name);
    }

    void next(const telem::TimeSpan elapsed) {
        this->ctx.elapsed = elapsed;
        bool first = true;
        for (const auto &stratum: this->strata.strata) {
            for (const auto &node_key: stratum) {
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

        // Auto-deactivate sequences in terminal stages
        check_terminal_stages();
    }
};
}
