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
#include <optional>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "glog/logging.h"

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::scheduler {
/// @brief identifies a stage within a sequence.
struct StageRef {
    std::string sequence;
    std::string stage;

    bool operator==(const StageRef &other) const {
        return sequence == other.sequence && stage == other.stage;
    }

    bool operator<(const StageRef &other) const {
        if (sequence != other.sequence) return sequence < other.sequence;
        return stage < other.stage;
    }
};
}

template <>
struct std::hash<arc::runtime::scheduler::StageRef> {
    size_t operator()(const arc::runtime::scheduler::StageRef &ref) const noexcept {
        return std::hash<std::string>{}(ref.sequence) ^
               std::hash<std::string>{}(ref.stage) << 1;
    }
};

namespace arc::runtime::scheduler {
/// @brief reactive scheduler that executes nodes based on strata and stages.
class Scheduler {
    /// @brief internal state for a node including its outgoing edges.
    struct NodeState {
        /// @brief the node key
        std::string key;
        /// @brief the runtime node implementation.
        std::unique_ptr<node::Node> node;
        /// @brief cached output edges for the node.
        std::vector<ir::Edge> output_edges;
        /// @brief ref for the stage that the node belongs to. null for global nodes.
        std::optional<StageRef> stage;
    };

    ///////////////////// Constant (set only during construction) /////////////////////
    /// @brief global strata for nodes not in any sequence stage.
    ir::Strata strata;
    /// @brief per-stage strata for independent execution within each stage.
    std::unordered_map<StageRef, ir::Strata> stage_strata;
    /// @brief all nodes in the program, keyed by node key.
    std::unordered_map<std::string, NodeState> nodes;
    /// @brief maximum iterations for convergence loop (total stages + 1).
    size_t max_convergence_iterations = 1;
    /// @brief nodes belonging to each stage.
    std::unordered_map<StageRef, std::vector<std::string>> stage_nodes;
    /// @brief maps entry node keys to their target stages.
    std::unordered_map<std::string, StageRef> entry_node_targets;

    ///////////////////// Variable (modified during execution) /////////////////////
    /// @brief context passed to nodes during execution.
    node::Context ctx = node::Context{
        .mark_changed = std::bind_front(&Scheduler::mark_changed, this),
        .report_error = std::bind_front(&Scheduler::report_error),
        .activate_stage = std::bind_front(&Scheduler::transition_stage, this)
    };
    /// @brief nodes marked as changed during this execution cycle.
    std::unordered_set<std::string> changed;
    /// @brief the node currently being executed.
    NodeState *curr_node = nullptr;
    /// @brief currently active stages.
    std::unordered_set<StageRef> active_stages;
    /// @brief snapshot of active stages before executing them.
    std::unordered_set<StageRef> prev_active_stages;
    /// @brief scratch space for computing stage diffs during convergence.
    std::unordered_set<StageRef> stage_diff;
    /// @brief one-shot edges that have fired, keyed by stage.
    std::unordered_map<StageRef, std::unordered_set<ir::Edge>> fired_one_shots;
    /// @brief tracks the currently active stage for each sequence.
    std::unordered_map<std::string, StageRef> active_stage_per_sequence;

public:
    /// @brief constructs a scheduler from IR and node implementations.
    Scheduler(
        const ir::IR &prog,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &nodes
    ):
        strata(prog.strata) {
        for (auto &[key, node]: nodes)
            this->nodes[key] = NodeState{
                .key = key,
                .node = std::move(node),
                .output_edges = prog.outgoing_edges(key)
            };

        size_t total_stages = 0;
        for (const auto &seq: prog.sequences) {
            total_stages += seq.stages.size();
            for (const auto &stage: seq.stages) {
                const StageRef ref{seq.key, stage.key};
                this->stage_nodes[ref] = stage.nodes;
                this->stage_strata[ref] = stage.strata;
                for (const auto &node_key: stage.nodes)
                    this->nodes[node_key].stage = ref;
                const std::string entry_key = "entry_" + seq.key + "_" + stage.key;
                this->entry_node_targets[entry_key] = ref;
            }
        }
        this->max_convergence_iterations = total_stages + 1;
    }

    /// @brief executes one cycle of the reactive scheduler.
    void next(const telem::TimeSpan elapsed) {
        this->ctx.elapsed = elapsed;
        this->exec_globals();
        this->exec_stages();
    }

private:
    /// @brief executes nodes in the given strata.
    void execute_strata(const ir::Strata &s) {
        this->changed.clear();
        bool first_stratum = true;
        for (const auto &stratum: s.strata) {
            for (const auto &node_key: stratum)
                if (first_stratum || this->changed.contains(node_key))
                    this->exec_node(node_key);
            first_stratum = false;
        }
    }

    static void report_error(const xerrors::Error &e) {
        LOG(ERROR) << "node encountered error" << e;
    }

    /// @brief executes nodes in global strata.
    void exec_globals() {
        this->execute_strata(this->strata);
    }

    /// @brief executes nodes within a specific stage's strata.
    void exec_stage(const StageRef &ref) {
        this->execute_strata(this->stage_strata[ref]);
    }

    /// @brief executes a single node by key.
    void exec_node(const std::string &node_key) {
        this->curr_node = &this->nodes[node_key];
        this->curr_node->node->next(this->ctx);
    }

    /// @brief executes active stages and processes transitions until stable.
    void exec_stages() {
        for (size_t i = 0; i < this->max_convergence_iterations; ++i) {
            this->prev_active_stages = this->active_stages;
            for (const auto &ref: this->prev_active_stages)
                this->exec_stage(ref);
            this->compute_stage_diff();
            if (this->stage_diff.empty()) break;
            for (const auto &ref: this->stage_diff)
                this->reset_stage(ref);
        }
        if (!this->stage_diff.empty())
            LOG(ERROR) << "[arc] convergence loop exceeded max iterations";
    }

    /// @brief computes newly activated stages (in active_stages but not in prev).
    void compute_stage_diff() {
        this->stage_diff.clear();
        for (const auto &ref: this->active_stages)
            if (!this->prev_active_stages.contains(ref))
                this->stage_diff.insert(ref);
    }

    /// @brief marks downstream nodes as changed when a parameter is updated.
    void mark_changed(const std::string &param) {
        for (const auto &edge: this->curr_node->output_edges) {
            if (edge.source.param != param) continue;
            if (edge.kind == ir::EdgeKind::OneShot) {
                if (!this->curr_node->node->is_output_truthy(edge.source.param))
                    continue;
                if (this->curr_node->stage.has_value()) {
                    auto &fired = this->fired_one_shots[*this->curr_node->stage];
                    if (fired.contains(edge)) continue;
                    fired.insert(edge);
                }
            }
            this->changed.insert(edge.target.node);
        }
    }

    /// @brief queues a stage transition when a node triggers activation.
    void transition_stage(const std::string &node_key) {
        const auto next = this->entry_node_targets[node_key];
        if (const auto it = this->active_stage_per_sequence.find(next.sequence);
            it != this->active_stage_per_sequence.end())
            this->active_stages.erase(it->second);
        this->active_stages.insert(next);
        this->active_stage_per_sequence[next.sequence] = next;
    }

    /// @brief resets all nodes in a stage and clears fired one-shots.
    void reset_stage(const StageRef &ref) {
        this->fired_one_shots.erase(ref);
        for (const auto &node_key: this->stage_nodes[ref])
            this->nodes[node_key].node->reset();
    }
};
}
