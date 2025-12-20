// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "glog/logging.h"

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::scheduler {
/// @brief Reactive scheduler that executes nodes based on stratified dependencies.
class Scheduler {
    /// @brief State for a single node including its implementation and edges.
    struct NodeState {
        /// @brief Outgoing edges keyed by output parameter name.
        std::unordered_map<std::string, std::vector<ir::Edge>> output_edges;
        /// @brief The node implementation.
        std::unique_ptr<node::Node> node;
    };

    /// @brief State for a single stage within a sequence.
    struct StageState {
        /// @brief Stratified node keys defining execution order.
        ir::Strata strata;
        /// @brief One-shot edges that have already fired in this stage activation.
        std::unordered_set<ir::Edge> fired_one_shots;
    };

    /// @brief State for a sequence of stages.
    struct SequenceState {
        /// @brief Ordered list of stages in this sequence.
        std::vector<StageState> stages;
        /// @brief Index of the currently active stage, or npos if none.
        size_t active_stage_idx = std::string::npos;
    };

    // Graph structure (immutable after construction)

    /// @brief All nodes keyed by their unique identifier.
    std::unordered_map<std::string, NodeState> nodes;
    /// @brief Stratified node keys for global (non-sequence) execution.
    ir::Strata global_strata;
    /// @brief All sequences in the program.
    std::vector<SequenceState> sequences;
    /// @brief Maps entry node keys to their target (sequence_idx, stage_idx).
    std::unordered_map<std::string, std::pair<std::size_t, std::size_t>> transitions;
    /// @brief Maximum iterations for stage convergence loop.
    size_t max_convergence_iterations = 0;

    // Execution state (changes during next()) ─────────────────

    /// @brief Context passed to nodes during execution.
    node::Context ctx = node::Context{
        .mark_changed = std::bind_front(&Scheduler::mark_changed, this),
        .report_error = std::bind_front(&Scheduler::report_error),
        .activate_stage = std::bind_front(&Scheduler::transition_stage, this),
    };
    /// @brief Set of node keys that need execution in the current stratum pass.
    std::unordered_set<std::string> changed;
    /// @brief Key of the currently executing node.
    std::string curr_node_key;
    /// @brief Index of the currently executing sequence, or npos if global.
    size_t curr_seq_idx = std::string::npos;
    /// @brief Index of the currently executing stage, or npos if none.
    size_t curr_stage_idx = std::string::npos;

public:
    /// @brief Constructs a scheduler from an IR program and node implementations.
    Scheduler(
        const ir::IR &prog,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls
    ) {
        for (auto &[key, node]: node_impls)
            this->nodes[key] = NodeState{
                .output_edges = prog.outgoing_edges(key),
                .node = std::move(node),
            };
        this->global_strata = prog.strata;
        this->sequences.resize(prog.sequences.size());
        for (size_t i = 0; i < prog.sequences.size(); i++) {
            const auto &seq = prog.sequences[i];
            auto &seq_state = this->sequences[i];
            seq_state.stages.resize(seq.stages.size());
            this->max_convergence_iterations += seq.stages.size();
            for (size_t j = 0; j < seq.stages.size(); j++) {
                const auto &stage = seq.stages[j];
                auto &stage_state = seq_state.stages[j];
                stage_state.strata = stage.strata;
                const auto entry_key = "entry_" + seq.key + "_" + stage.key;
                this->transitions[entry_key] = {i, j};
            }
        }
    }

    /// @brief Advances the scheduler by executing global and stage strata.
    void next(const telem::TimeSpan elapsed) {
        this->ctx.elapsed = elapsed;
        this->execute_strata(this->global_strata);
        this->exec_stages();
    }

private:
    /// @brief Returns the NodeState for the currently executing node.
    NodeState &curr_node() { return this->nodes[this->curr_node_key]; }

    /// @brief Returns the StageState for the currently executing stage.
    StageState &curr_stage() {
        return this->sequences[this->curr_seq_idx].stages[this->curr_stage_idx];
    }

    /// @brief Executes all strata, propagating changes between them.
    void execute_strata(const ir::Strata &strata) {
        this->changed.clear();
        bool first_stratum = true;
        for (const auto &stratum: strata.strata) {
            for (const auto &key: stratum)
                if (first_stratum || this->changed.contains(key)) {
                    this->curr_node_key = key;
                    this->curr_node().node->next(this->ctx);
                }
            first_stratum = false;
        }
    }

    /// @brief Executes active stages across all sequences until convergence.
    void exec_stages() {
        for (size_t iter = 0; iter < this->max_convergence_iterations; iter++) {
            bool stable = true;
            for (this->curr_seq_idx = 0; this->curr_seq_idx < this->sequences.size();
                 this->curr_seq_idx++) {
                auto &seq = this->sequences[this->curr_seq_idx];
                if (seq.active_stage_idx == std::string::npos) continue;
                this->curr_stage_idx = seq.active_stage_idx;
                this->execute_strata(seq.stages[this->curr_stage_idx].strata);
                if (seq.active_stage_idx != this->curr_stage_idx) stable = false;
            }
            if (stable) break;
        }
    }

    /// @brief Logs an error reported by a node.
    static void report_error(const xerrors::Error &e) {
        LOG(ERROR) << "[arc] node encountered error: " << e;
    }

    /// @brief Marks downstream nodes as changed based on edge propagation rules.
    void mark_changed(const std::string &param) {
        for (const auto &edge: this->curr_node().output_edges[param])
            if (edge.kind == ir::EdgeKind::Continuous ||
                (this->curr_node().node->is_output_truthy(param) &&
                 (this->curr_stage_idx == std::string::npos ||
                  this->curr_stage().fired_one_shots.insert(edge).second)))
                this->changed.insert(edge.target.node);
    }

    /// @brief Resets all nodes in a strata to their initial state.
    void reset_strata(const ir::Strata &strata) {
        for (auto &stratum: strata.strata)
            for (const auto &key: stratum)
                this->nodes[key].node->reset();
    }

    /// @brief Transitions to a new stage, deactivating the current one.
    void transition_stage() {
        if (this->curr_seq_idx != std::string::npos)
            this->sequences[this->curr_seq_idx].active_stage_idx = std::string::npos;
        const auto [target_seq_idx, target_stage_idx] = this->transitions
                                                            [this->curr_node_key];
        auto &target = this->sequences[target_seq_idx].stages[target_stage_idx];
        target.fired_one_shots.clear();
        this->reset_strata(target.strata);
        this->sequences[target_seq_idx].active_stage_idx = target_stage_idx;
    }
};
}
