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

#include <glog/logging.h>

#include "x/cpp/xerrors/errors.h"

#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::scheduler {

struct StageRef {
    std::string sequence;
    std::string stage;

    bool operator==(const StageRef &other) const {
        return sequence == other.sequence && stage == other.stage;
    }

    struct Hasher {
        size_t operator()(const StageRef &ref) const {
            const size_t h1 = std::hash<std::string>()(ref.sequence);
            const size_t h2 = std::hash<std::string>()(ref.stage);
            return h1 ^ (h2 << 1);
        }
    };
};

class Scheduler {
    /// @brief Global strata for nodes not in any sequence stage.
    ir::Strata strata;
    /// @brief Per-stage strata for independent execution within each stage.
    std::unordered_map<StageRef, ir::Strata, StageRef::Hasher> stage_strata;
    std::unordered_set<std::string> changed;

    struct NodeState {
        std::string key;
        std::unique_ptr<node::Node> node;
        std::vector<ir::Edge> output_edges;
    };
    std::unordered_map<std::string, NodeState> nodes;
    NodeState *current_state;
    node::Context ctx;

    std::unordered_map<std::string, ir::Sequence> sequences_by_key;
    std::unordered_map<std::string, std::string> active_stages;
    std::unordered_set<std::string> just_activated;
    /// @brief Pending stage transitions to process in convergence loop.
    std::vector<std::pair<std::string, std::string>> pending_transitions;
    std::unordered_map<StageRef, std::vector<std::string>, StageRef::Hasher>
        stage_to_nodes;
    std::unordered_map<std::string, StageRef> node_to_stage;
    std::unordered_map<std::string, StageRef> entry_node_targets;
    std::unordered_map<std::string, std::unordered_set<ir::Edge, ir::Edge::Hasher>>
        fired_one_shots;

    void mark_changed(const std::string &param) {
        for (const auto &edge: this->current_state->output_edges) {
            if (edge.source.param != param) continue;
            if (edge.kind == ir::EdgeKind::OneShot) {
                if (!this->current_state->node->is_output_truthy(edge.source.param))
                    continue;
                std::string seq_name;
                if (auto it = this->node_to_stage.find(current_state->key);
                    it != this->node_to_stage.end())
                    seq_name = it->second.sequence;
                auto &fired_set = this->fired_one_shots[seq_name];
                if (fired_set.contains(edge)) continue;
                fired_set.insert(edge);
            }
            this->changed.insert(edge.target.node);
        }
    }

    void activate_stage_by_node(const std::string &node_key) {
        if (const auto it = this->entry_node_targets.find(node_key);
            it != this->entry_node_targets.end()) {
            // Queue transition for convergence loop processing
            this->pending_transitions.emplace_back(it->second.sequence, it->second.stage);
            return;
        }
        if (const auto it = this->node_to_stage.find(node_key);
            it != this->node_to_stage.end())
            // Queue transition for convergence loop processing
            this->pending_transitions.emplace_back(
                it->second.sequence,
                it->second.stage
            );
    }

    [[nodiscard]] bool should_execute_node(const std::string &node_key) const {
        if (this->stage_to_nodes.empty()) return true;
        const auto it = this->node_to_stage.find(node_key);
        if (it == this->node_to_stage.end()) return true;
        const auto stage_it = this->active_stages.find(it->second.sequence);
        if (stage_it == this->active_stages.end()) return false;
        return stage_it->second == it->second.stage;
    }

    void check_terminal_stages() {
        std::vector<std::string> to_deactivate;
        for (const auto &[seq_name, stage_name]: active_stages) {
            if (this->just_activated.contains(seq_name)) continue;
            auto seq_it = this->sequences_by_key.find(seq_name);
            if (seq_it == this->sequences_by_key.end()) continue;
            if (seq_it->second.next_stage(stage_name) != nullptr) continue;
            if (stage_has_unfired_one_shots(seq_name, stage_name)) continue;
            to_deactivate.push_back(seq_name);
        }
        for (const auto &seq_name: to_deactivate)
            deactivate_sequence(seq_name);
    }

    [[nodiscard]] bool stage_has_unfired_one_shots(
        const std::string &seq_name,
        const std::string &stage_name
    ) const {
        const StageRef ref{seq_name, stage_name};
        const auto stage_it = this->stage_to_nodes.find(ref);
        if (stage_it == this->stage_to_nodes.end()) return false;

        const auto fired_it = this->fired_one_shots.find(seq_name);
        const std::unordered_set<ir::Edge, ir::Edge::Hasher>
            *fired_set = (fired_it != this->fired_one_shots.end()) ? &fired_it->second
                                                                   : nullptr;

        for (const auto &node_key: stage_it->second) {
            auto node_it = this->nodes.find(node_key);
            if (node_it == this->nodes.end()) continue;
            for (const auto &edge: node_it->second.output_edges) {
                if (edge.kind != ir::EdgeKind::OneShot) continue;
                if (fired_set == nullptr || !fired_set->contains(edge)) return true;
            }
        }
        return false;
    }

public:
    Scheduler(
        const ir::IR &prog,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &nodes
    ):
        strata(prog.strata), current_state(nullptr) {
        for (auto &[key, node]: nodes)
            this->nodes[key] = NodeState{
                .key = key,
                .node = std::move(node),
                .output_edges = prog.outgoing_edges(key)
            };

        this->ctx = node::Context{
            .mark_changed =
                [&](const std::string &param) { this->mark_changed(param); },
            .report_error = [&](const xerrors::Error &) {},
            .activate_stage = [&](
                                  const std::string &node_key
                              ) { this->activate_stage_by_node(node_key); }
        };

        load_sequences(prog.sequences);
    }

    void load_sequences(const std::vector<ir::Sequence> &seqs) {
        for (const auto &seq: seqs) {
            this->sequences_by_key[seq.key] = seq;
            for (const auto &stage: seq.stages) {
                const StageRef ref{seq.key, stage.key};
                this->stage_to_nodes[ref] = stage.nodes;
                // Store per-stage strata for two-tier execution
                this->stage_strata[ref] = stage.strata;
                for (const auto &node_key: stage.nodes)
                    this->node_to_stage[node_key] = ref;

                // Map entry node to target stage
                const std::string entry_key = "entry_" + seq.key + "_" + stage.key;
                this->entry_node_targets[entry_key] = ref;
            }
        }
    }

    void activate_stage(const std::string &seq, const std::string &stage) {
        LOG(INFO) << "[arc] stage " << seq << ":" << stage;
        this->active_stages[seq] = stage;
        this->just_activated.insert(seq);
        reset_stage_nodes(seq, stage);
        // No longer need to mark nodes as changed - stage-local sources are at
        // stratum 0 within their stage and execute automatically
    }

    void deactivate_sequence(const std::string &seq_name) {
        LOG(INFO) << "[arc] deactivate " << seq_name;
        this->active_stages.erase(seq_name);
        this->fired_one_shots.erase(seq_name);
    }

    void reset_stage_nodes(const std::string &seq_name, const std::string &stage_name) {
        this->fired_one_shots.erase(seq_name);
        const StageRef ref{seq_name, stage_name};
        const auto it = this->stage_to_nodes.find(ref);
        if (it == this->stage_to_nodes.end()) return;
        for (const auto &node_key: it->second) {
            if (auto node_it = nodes.find(node_key); node_it != nodes.end())
                node_it->second.node->reset();
        }
    }

    [[nodiscard]] std::vector<std::string> get_active_sequences() const {
        std::vector<std::string> seqs;
        seqs.reserve(this->active_stages.size());
        for (const auto &seq: this->active_stages | std::views::keys)
            seqs.push_back(seq);
        return seqs;
    }

    [[nodiscard]] std::string get_active_stage_for(const std::string &seq_name) const {
        const auto it = this->active_stages.find(seq_name);
        return it != this->active_stages.end() ? it->second : "";
    }

    [[nodiscard]] bool is_sequence_active(const std::string &seq_name) const {
        return this->active_stages.contains(seq_name);
    }

    void next(const telem::TimeSpan elapsed) {
        this->ctx.elapsed = elapsed;
        this->just_activated.clear();
        this->pending_transitions.clear();

        // Step 1: Execute global strata (nodes not in any stage)
        // Stratum 0 always executes, higher strata only if changed
        execute_global_strata();

        // Step 2: For each active stage, execute its per-stage strata
        // Stage-local sources (constants, channel reads) are at stratum 0 and
        // execute every cycle. Higher strata only execute if inputs changed.
        for (const auto &[seq_name, stage_name]: this->active_stages)
            execute_stage_strata(seq_name, stage_name);

        // Step 3: Convergence loop for immediate stage transitions
        // When a transition fires during execution, the new stage executes
        // immediately within the same cycle
        convergence_loop();

        this->changed.clear();
        check_terminal_stages();
    }

private:
    void execute_global_strata() {
        bool first_stratum = true;
        for (const auto &stratum: this->strata.strata) {
            for (const auto &node_key: stratum) {
                // Skip nodes that belong to a stage (they're executed per-stage)
                if (this->node_to_stage.contains(node_key)) continue;
                // Stratum 0 always executes, higher strata only if changed
                if (first_stratum || this->changed.contains(node_key))
                    execute_node(node_key);
            }
            first_stratum = false;
        }
    }

    void execute_stage_strata(
        const std::string &seq_name,
        const std::string &stage_name
    ) {
        const StageRef ref{seq_name, stage_name};
        const auto it = this->stage_strata.find(ref);
        if (it == this->stage_strata.end()) return;

        bool first_stratum = true;
        for (const auto &stratum: it->second.strata) {
            for (const auto &node_key: stratum) {
                // Stratum 0 always executes (stage-local sources),
                // higher strata only if inputs changed
                if (first_stratum || this->changed.contains(node_key))
                    execute_node(node_key);
            }
            first_stratum = false;
        }
    }

    void execute_node(const std::string &node_key) {
        auto it = this->nodes.find(node_key);
        if (it == this->nodes.end()) return;
        this->current_state = &it->second;
        LOG(INFO) << "[arc] fire " << node_key;
        this->current_state->node->next(this->ctx);
    }

    void convergence_loop() {
        // Maximum iterations = total number of stages + 1 (safety bound)
        size_t total_stages = 0;
        for (const auto &seq: this->sequences_by_key)
            total_stages += seq.second.stages.size();
        const size_t max_iterations = total_stages + 1;

        for (size_t i = 0; i < max_iterations && !this->pending_transitions.empty();
             ++i) {
            // Copy and clear pending transitions
            auto transitions = std::move(this->pending_transitions);
            this->pending_transitions.clear();

            for (const auto &[seq_name, stage_name]: transitions) {
                activate_stage(seq_name, stage_name);
                execute_stage_strata(seq_name, stage_name);
            }
        }

        if (!this->pending_transitions.empty())
            LOG(ERROR) << "[arc] convergence loop exceeded max iterations";
    }
};
}
