// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <memory>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "glog/logging.h"

#include "x/cpp/errors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::scheduler {
/// @brief Sentinel value indicating no valid index.
static constexpr size_t NO_INDEX = ~size_t{0};

/// @brief Reactive scheduler that executes nodes based on stratified dependencies.
class Scheduler {
    /// @brief State for a single node including its implementation and edges.
    struct Node {
        /// @brief Outgoing edges keyed by output parameter name.
        std::unordered_map<std::string, std::vector<ir::Edge>> output_edges;
        /// @brief The node implementation.
        std::unique_ptr<node::Node> node;
    };

    /// @brief State for a single stage within a sequence.
    struct Stage {
        /// @brief Stratified node keys defining execution order.
        ir::Strata strata;
        /// @brief One-shot edges that have already fired in this stage activation.
        std::unordered_set<ir::Edge> fired_one_shots;
    };

    /// @brief State for a sequence of stages.
    struct Sequence {
        /// @brief Ordered list of stages in this sequence.
        std::vector<Stage> stages;
        /// @brief Index of the currently active stage, or npos if none.
        size_t active_stage_idx = NO_INDEX;
    };

    // Graph structure (immutable after construction)

    /// @brief All nodes keyed by their unique identifier.
    std::unordered_map<std::string, Node> nodes;
    /// @brief Stratified node keys for global (non-sequence) execution.
    ir::Strata global_strata;
    /// @brief All sequences in the program.
    std::vector<Sequence> sequences;
    /// @brief Maps entry node keys to their target (sequence_idx, stage_idx).
    std::unordered_map<std::string, std::pair<std::size_t, std::size_t>> transitions;
    /// @brief Maximum iterations for stage convergence loop.
    size_t max_convergence_iterations = 0;
    /// @brief Tolerance for timing comparisons to handle OS scheduling jitter.
    x::telem::TimeSpan tolerance_;
    /// @brief Error handler for reporting node execution errors.
    errors::Handler error_handler;

    // Execution state (changes during next()) ─────────────────

    /// @brief Context passed to nodes during execution.
    node::Context ctx;
    /// @brief Maps node keys to dense indices for flag arrays.
    std::unordered_map<std::string, size_t> node_index;
    /// @brief Flag array indicating which nodes need execution in the current
    /// stratum pass.
    std::vector<uint8_t> changed_flags;
    /// @brief Flag array for nodes that requested re-execution on the next cycle.
    std::vector<uint8_t> self_changed_flags;
    /// @brief Minimum deadline (absolute elapsed time) across all nodes in the
    /// current next() call. Reset to max at the start of each next() call.
    x::telem::TimeSpan next_deadline_ = x::telem::TimeSpan::max();
    /// @brief One-shot edges that have fired in global strata (never reset).
    std::unordered_set<ir::Edge> global_fired_one_shots;
    /// @brief Pointer to the key of the currently executing node (points into
    /// strata string vectors which are immutable after construction).
    const std::string *curr_node_ptr = nullptr;
    /// @brief Index of the currently executing sequence, or npos if global.
    size_t curr_seq_idx = NO_INDEX;
    /// @brief Index of the currently executing stage, or npos if none.
    size_t curr_stage_idx = NO_INDEX;
    /// @brief Set to true when transition_stage fires during strata execution.
    /// Used to stop evaluating remaining statements after the first transition.
    bool transitioned = false;

public:
    /// @brief Constructs a scheduler from an IR program and node implementations.
    Scheduler(
        const ir::IR &prog,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls,
        const x::telem::TimeSpan tolerance,
        errors::Handler error_handler = errors::noop_handler
    ):
        tolerance_(tolerance), error_handler(std::move(error_handler)) {
        this->ctx.mark_changed = std::bind_front(&Scheduler::mark_changed, this);
        this->ctx.mark_self_changed = std::bind_front(
            &Scheduler::mark_self_changed,
            this
        );
        this->ctx.set_deadline = [this](const x::telem::TimeSpan d) {
            if (d < this->next_deadline_) this->next_deadline_ = d;
        };
        this->ctx.report_error = std::bind_front(&Scheduler::report_error, this);
        this->ctx.activate_stage = std::bind_front(&Scheduler::transition_stage, this);
        size_t idx = 0;
        for (auto &[key, node]: node_impls) {
            this->node_index[key] = idx++;
            this->nodes[key] = Node{
                .output_edges = prog.edges_from(key),
                .node = std::move(node),
            };
        }
        this->changed_flags.resize(idx, 0);
        this->self_changed_flags.resize(idx, 0);
        this->global_strata = prog.strata;
        this->sequences.resize(prog.sequences.size());
        for (size_t i = 0; i < prog.sequences.size(); i++) {
            const auto &seq_ir = prog.sequences[i];
            auto &seq = this->sequences[i];
            seq.stages.resize(seq_ir.stages.size());
            this->max_convergence_iterations += seq_ir.stages.size();
            for (size_t j = 0; j < seq_ir.stages.size(); j++) {
                const auto &stage_ir = seq_ir.stages[j];
                auto &stage = seq.stages[j];
                stage.strata = stage_ir.strata;
                const auto entry_key = "entry_" + seq_ir.key + "_" + stage_ir.key;
                this->transitions[entry_key] = {i, j};
            }
        }
    }

    // Make Scheduler non-movable to prevent dangling 'this' in callbacks
    Scheduler(Scheduler &&) = delete;
    Scheduler &operator=(Scheduler &&) = delete;
    Scheduler(const Scheduler &) = delete;
    Scheduler &operator=(const Scheduler &) = delete;

    /// @brief Resets all execution state for runtime restart.
    void reset() {
        std::fill(this->changed_flags.begin(), this->changed_flags.end(), 0);
        std::fill(this->self_changed_flags.begin(), this->self_changed_flags.end(), 0);
        this->global_fired_one_shots.clear();
        this->curr_node_ptr = nullptr;
        this->curr_seq_idx = NO_INDEX;
        this->curr_stage_idx = NO_INDEX;
        this->transitioned = false;
        for (auto &seq: this->sequences) {
            seq.active_stage_idx = NO_INDEX;
            for (auto &stage: seq.stages)
                stage.fired_one_shots.clear();
        }
        for (auto &[key, node_state]: this->nodes)
            node_state.node->reset();
    }

    /// @brief Advances the scheduler by executing global and stage strata.
    /// @param elapsed Time elapsed since runtime start.
    /// @param reason Why this scheduler run was triggered (timer tick or channel
    /// input).
    void next(const x::telem::TimeSpan elapsed, const node::RunReason reason) {
        this->next_deadline_ = x::telem::TimeSpan::max();
        this->ctx.elapsed = elapsed;
        this->ctx.tolerance = this->tolerance_;
        this->ctx.reason = reason;
        this->curr_seq_idx = NO_INDEX;
        this->curr_stage_idx = NO_INDEX;
        this->execute_strata(this->global_strata);
        this->exec_stages();
    }

    /// @brief Returns the minimum deadline reported by nodes during the last next()
    /// call. The deadline is an absolute elapsed time. Returns TimeSpan::max() if no
    /// node reported a deadline.
    [[nodiscard]] x::telem::TimeSpan next_deadline() const {
        return this->next_deadline_;
    }

private:
    /// @brief Returns the NodeState for the currently executing node.
    Node &curr_node() { return this->nodes[*this->curr_node_ptr]; }

    /// @brief Returns the StageState for the currently executing stage.
    Stage &curr_stage() {
        return this->sequences[this->curr_seq_idx].stages[this->curr_stage_idx];
    }

    /// @brief Executes all strata, propagating changes between them.
    void execute_strata(const ir::Strata &strata) {
        std::fill(this->changed_flags.begin(), this->changed_flags.end(), 0);
        this->transitioned = false;
        const bool in_stage = this->curr_stage_idx != NO_INDEX;
        bool first_stratum = true;
        for (const auto &stratum: strata) {
            for (const auto &key: stratum) {
                const auto idx = this->node_index[key];
                const bool was_self_changed = this->self_changed_flags[idx] != 0;
                if (was_self_changed) this->self_changed_flags[idx] = 0;
                if (first_stratum || this->changed_flags[idx] || was_self_changed) {
                    this->curr_node_ptr = &key;
                    this->curr_node().node->next(this->ctx);
                }
                if (in_stage && this->transitioned) return;
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
                if (seq.active_stage_idx == NO_INDEX) continue;
                this->curr_stage_idx = seq.active_stage_idx;
                this->execute_strata(seq.stages[this->curr_stage_idx].strata);
                if (seq.active_stage_idx != this->curr_stage_idx) stable = false;
            }
            if (stable) break;
        }
    }

    /// @brief Reports an error from a node to the error handler.
    void report_error(const x::errors::Error &e) {
        LOG(ERROR) << "[arc.scheduler] node encountered error: " << e;
        this->error_handler(e);
    }

    /// @brief Marks downstream nodes as changed based on edge propagation rules.
    void mark_changed(const std::string &param) {
        for (const auto &edge: this->curr_node().output_edges[param])
            if (edge.kind == ir::EdgeKind::Continuous)
                this->changed_flags[this->node_index[edge.target.node]] = 1;
            else if (this->curr_node().node->is_output_truthy(param)) {
                auto &fired_set = this->curr_stage_idx == NO_INDEX
                                    ? this->global_fired_one_shots
                                    : this->curr_stage().fired_one_shots;
                if (fired_set.insert(edge).second)
                    this->changed_flags[this->node_index[edge.target.node]] = 1;
            }
    }

    void mark_self_changed() {
        this->self_changed_flags[this->node_index[*this->curr_node_ptr]] = 1;
    }

    /// @brief Resets all nodes in a strata to their initial state.
    void reset_strata(const ir::Strata &strata) {
        for (const auto &stratum: strata)
            for (const auto &key: stratum) {
                this->self_changed_flags[this->node_index[key]] = 0;
                this->nodes[key].node->reset();
            }
    }

    /// @brief Clears self_changed flags for all nodes in a strata.
    void clear_self_changed(const ir::Strata &strata) {
        for (const auto &stratum: strata)
            for (const auto &key: stratum)
                this->self_changed_flags[this->node_index[key]] = 0;
    }

    void transition_stage() {
        if (this->curr_seq_idx != NO_INDEX) {
            auto &source = this->sequences[this->curr_seq_idx]
                               .stages[this->curr_stage_idx];
            this->clear_self_changed(source.strata);
            this->sequences[this->curr_seq_idx].active_stage_idx = NO_INDEX;
        }
        const auto [target_seq_idx, target_stage_idx] = this->transitions
                                                            [*this->curr_node_ptr];
        auto &target = this->sequences[target_seq_idx].stages[target_stage_idx];
        target.fired_one_shots.clear();
        this->reset_strata(target.strata);
        this->sequences[target_seq_idx].active_stage_idx = target_stage_idx;
        this->transitioned = true;
    }
};
}
