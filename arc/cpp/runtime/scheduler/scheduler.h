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
static constexpr size_t NO_INDEX = ~size_t{0};

class Scheduler {
    struct Node {
        std::unordered_map<std::string, std::vector<ir::Edge>> output_edges;
        std::unique_ptr<node::Node> node;
    };

    struct SequenceState;

    struct StepState {
        const ir::Step *ir;
        std::vector<SequenceState> sub_seqs;
        std::unique_ptr<SequenceState> sub_seq;
    };

    struct SequenceState {
        const ir::Sequence *ir;
        std::vector<StepState> steps;
        size_t active_step_idx = NO_INDEX;
        std::unordered_map<std::string, size_t> flow_node_owner;
        std::unordered_set<std::string> flow_data_nodes;
    };

    struct TransitionTarget {
        SequenceState *seq;
        size_t step_idx;
    };

    std::unordered_map<std::string, Node> nodes;
    ir::Strata global_strata;
    std::vector<SequenceState> sequences;
    std::unordered_map<std::string, TransitionTarget> transitions;
    std::unordered_map<std::string, SequenceState *> boundaries;
    size_t max_convergence_iterations = 0;
    x::telem::TimeSpan tolerance_;
    errors::Handler error_handler;

    node::Context ctx;
    std::unordered_map<std::string, size_t> node_index;
    std::vector<uint8_t> changed_flags;
    std::vector<uint8_t> self_changed_flags;
    x::telem::TimeSpan next_deadline_ = x::telem::TimeSpan::max();
    const std::string *curr_node_ptr = nullptr;
    SequenceState *curr_seq = nullptr;
    bool transitioned = false;

    ir::IR prog_;

    static SequenceState build_sequence_state(const ir::Sequence &seq) {
        SequenceState state;
        state.ir = &seq;
        state.steps.resize(seq.steps.size());
        bool has_flow_steps = false;
        for (size_t i = 0; i < seq.steps.size(); i++) {
            auto &ss = state.steps[i];
            ss.ir = &seq.steps[i];
            if (seq.steps[i].stage) {
                for (const auto &sub_seq: seq.steps[i].stage->sequences)
                    ss.sub_seqs.push_back(build_sequence_state(sub_seq));
            }
            if (seq.steps[i].sequence) {
                ss.sub_seq = std::make_unique<SequenceState>(
                    build_sequence_state(*seq.steps[i].sequence)
                );
            }
            if (seq.steps[i].flow) has_flow_steps = true;
        }
        if (has_flow_steps) {
            for (size_t i = 0; i < seq.steps.size(); i++) {
                if (!seq.steps[i].flow) continue;
                for (const auto &nk: seq.steps[i].flow->nodes) {
                    state.flow_node_owner[nk] = i;
                    state.flow_data_nodes.insert(nk);
                }
            }
        }
        return state;
    }

    void register_transitions(SequenceState &seq) {
        for (size_t i = 0; i < seq.steps.size(); i++) {
            auto &step = seq.steps[i];
            const auto ek = "entry_" + seq.ir->key + "_" + step.ir->key;
            this->transitions[ek] = TransitionTarget{&seq, i};
            if (step.ir->stage) {
                for (auto &sub: step.sub_seqs)
                    this->register_transitions(sub);
            }
            if (step.sub_seq) this->register_transitions(*step.sub_seq);
        }
    }

    void register_boundaries(SequenceState &seq) {
        for (size_t i = 0; i < seq.steps.size(); i++) {
            auto &step = seq.steps[i];
            if (step.ir->stage) {
                for (size_t j = 0; j < step.sub_seqs.size(); j++) {
                    const auto bk = "boundary_" + step.sub_seqs[j].ir->key;
                    this->boundaries[bk] = &step.sub_seqs[j];
                    this->register_boundaries(step.sub_seqs[j]);
                }
            }
            if (step.ir->sequence) {
                const auto bk = "boundary_" + step.ir->key;
                this->boundaries[bk] = step.sub_seq.get();
                this->register_boundaries(*step.sub_seq);
            }
        }
    }

    static size_t count_steps(const ir::Sequence &seq) {
        size_t count = seq.steps.size();
        for (const auto &step: seq.steps) {
            if (step.stage)
                for (const auto &sub: step.stage->sequences)
                    count += count_steps(sub);
            if (step.sequence) count += count_steps(*step.sequence);
        }
        return count;
    }

public:
    Scheduler(
        ir::IR prog,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls,
        const x::telem::TimeSpan tolerance,
        errors::Handler error_handler = errors::noop_handler
    ):
        tolerance_(tolerance),
        error_handler(std::move(error_handler)),
        prog_(std::move(prog)) {
        this->ctx.mark_changed = std::bind_front(&Scheduler::mark_changed, this);
        this->ctx.mark_self_changed = std::bind_front(
            &Scheduler::mark_self_changed,
            this
        );
        this->ctx.set_deadline = [this](const x::telem::TimeSpan d) {
            if (d < this->next_deadline_) this->next_deadline_ = d;
        };
        this->ctx.report_error = std::bind_front(&Scheduler::report_error, this);
        this->ctx.activate_stage = std::bind_front(&Scheduler::transition_step, this);

        size_t idx = 0;
        for (auto &[key, node]: node_impls) {
            this->node_index[key] = idx++;
            this->nodes[key] = Node{
                .output_edges = this->prog_.edges_from(key),
                .node = std::move(node),
            };
        }
        this->changed_flags.resize(idx, 0);
        this->self_changed_flags.resize(idx, 0);
        this->global_strata = this->prog_.root.strata;

        this->sequences.reserve(this->prog_.root.sequences.size());
        for (const auto &seq: this->prog_.root.sequences) {
            this->sequences.push_back(build_sequence_state(seq));
            this->max_convergence_iterations += count_steps(seq);
        }
        for (auto &seq: this->sequences) {
            this->register_transitions(seq);
            this->register_boundaries(seq);
        }
    }

    Scheduler(Scheduler &&) = delete;
    Scheduler &operator=(Scheduler &&) = delete;
    Scheduler(const Scheduler &) = delete;
    Scheduler &operator=(const Scheduler &) = delete;

    void reset() {
        std::fill(this->changed_flags.begin(), this->changed_flags.end(), 0);
        std::fill(this->self_changed_flags.begin(), this->self_changed_flags.end(), 0);
        this->curr_node_ptr = nullptr;
        this->curr_seq = nullptr;
        this->transitioned = false;
        for (auto &seq: this->sequences)
            this->reset_sequence_state(seq);
        for (auto &[key, node_state]: this->nodes)
            node_state.node->reset();
    }

    void next(const x::telem::TimeSpan elapsed, const node::RunReason reason) {
        this->next_deadline_ = x::telem::TimeSpan::max();
        this->ctx.elapsed = elapsed;
        this->ctx.tolerance = this->tolerance_;
        this->ctx.reason = reason;
        this->curr_seq = nullptr;
        this->execute_strata(this->global_strata, nullptr);
        this->exec_sequences();
    }

    [[nodiscard]] x::telem::TimeSpan next_deadline() const {
        return this->next_deadline_;
    }

    void mark_node_changed(const std::string &node_key) {
        this->changed_flags[this->node_index[node_key]] = 1;
    }

private:
    Node &curr_node() { return this->nodes[*this->curr_node_ptr]; }

    void execute_strata(const ir::Strata &strata, SequenceState *active_seq) {
        std::fill(this->changed_flags.begin(), this->changed_flags.end(), 0);
        this->transitioned = false;
        const bool in_context = this->curr_seq != nullptr;
        bool first_stratum = true;
        for (const auto &stratum: strata) {
            for (const auto &key: stratum) {
                auto bit = this->boundaries.find(key);
                if (bit != this->boundaries.end()) {
                    if (bit->second->active_step_idx != NO_INDEX) {
                        // Preserve the parent's changed/transitioned state
                        // across the recursive sub-context execution.
                        // execute_strata clears changed_flags on entry;
                        // without saving, propagations from earlier nodes in
                        // this stratum would be lost.
                        auto saved_changed = this->changed_flags;
                        const bool saved_transitioned = this->transitioned;
                        this->exec_sequence_step(bit->second);
                        this->changed_flags = std::move(saved_changed);
                        this->transitioned = saved_transitioned;
                    }
                    continue;
                }

                // Stage step boundaries in a sequence's strata are markers
                // used for ordering and dependency tracking only. Their step
                // is activated and executed via exec_sequence_step, not
                // through the parent's strata walk. Skip them here.
                if (this->node_index.find(key) == this->node_index.end()) continue;

                if (active_seq != nullptr && active_seq->flow_data_nodes.count(key)) {
                    auto it = active_seq->flow_node_owner.find(key);
                    if (it != active_seq->flow_node_owner.end() &&
                        it->second != active_seq->active_step_idx)
                        continue;
                }

                bool is_active_flow_node = false;
                if (active_seq != nullptr && active_seq->flow_data_nodes.count(key)) {
                    auto it = active_seq->flow_node_owner.find(key);
                    if (it != active_seq->flow_node_owner.end() &&
                        it->second == active_seq->active_step_idx)
                        is_active_flow_node = true;
                }

                const auto idx = this->node_index[key];
                const bool was_self_changed = this->self_changed_flags[idx] != 0;
                if (was_self_changed) this->self_changed_flags[idx] = 0;
                if (first_stratum || this->changed_flags[idx] || was_self_changed ||
                    is_active_flow_node) {
                    this->curr_node_ptr = &key;
                    this->curr_node().node->next(this->ctx);
                }
                if (in_context && this->transitioned) return;
            }
            first_stratum = false;
        }
    }

    void exec_sequences() {
        for (size_t iter = 0; iter < this->max_convergence_iterations; iter++) {
            bool stable = true;
            for (auto &seq: this->sequences) {
                if (seq.active_step_idx == NO_INDEX) continue;
                const auto prev = seq.active_step_idx;
                this->exec_sequence_step(&seq);
                if (seq.active_step_idx != prev) stable = false;
            }
            if (stable) break;
        }
    }

    void exec_sequence_step(SequenceState *seq) {
        if (seq->active_step_idx >= seq->steps.size()) return;
        auto &step = seq->steps[seq->active_step_idx];
        auto *prev_seq = this->curr_seq;
        this->curr_seq = seq;

        if (step.ir->stage) {
            this->execute_strata(step.ir->stage->strata, nullptr);
        } else if (step.ir->flow) {
            this->execute_strata(seq->ir->strata, seq);
        } else if (step.ir->sequence && step.sub_seq) {
            if (step.sub_seq->active_step_idx != NO_INDEX)
                this->exec_sequence_step(step.sub_seq.get());
        }

        this->curr_seq = prev_seq;
    }

    void report_error(const x::errors::Error &e) {
        LOG(ERROR) << "[arc.scheduler] node encountered error: " << e;
        this->error_handler(e);
    }

    /// @brief Marks downstream nodes as changed based on edge propagation rules.
    /// For continuous edges, always propagates. For conditional edges, only
    /// propagates when the source output is truthy.
    void mark_changed(const std::string &param) {
        for (const auto &edge: this->curr_node().output_edges[param])
            if (edge.kind != ir::EdgeKind::Conditional ||
                this->curr_node().node->is_output_truthy(param))
                this->changed_flags[this->node_index[edge.target.node]] = 1;
    }

    void mark_self_changed() {
        this->self_changed_flags[this->node_index[*this->curr_node_ptr]] = 1;
    }

    /// @brief Transitions to the step associated with the currently executing
    /// entry node. If entering from global strata and the target's sequence
    /// already has an active step, this is a no-op to prevent re-entering a
    /// sequence that has already been started.
    void transition_step() {
        auto it = this->transitions.find(*this->curr_node_ptr);
        if (it == this->transitions.end()) return;
        auto &[target_seq, target_step_idx] = it->second;
        if (this->curr_seq == nullptr && target_seq->active_step_idx != NO_INDEX)
            return;

        if (target_seq->active_step_idx != NO_INDEX) this->deactivate_step(target_seq);

        this->activate_step(target_seq, target_step_idx);
        this->transitioned = true;
    }

    void activate_step(SequenceState *seq, size_t step_idx) {
        seq->active_step_idx = step_idx;
        auto &step = seq->steps[step_idx];

        if (step.ir->stage) {
            this->reset_strata(step.ir->stage->strata);
            for (auto &sub: step.sub_seqs)
                this->enter_sequence(&sub);
        } else if (step.ir->flow) {
            for (const auto &nk: step.ir->flow->nodes) {
                this->self_changed_flags[this->node_index[nk]] = 0;
                auto nit = this->nodes.find(nk);
                if (nit != this->nodes.end()) nit->second.node->reset();
            }
        } else if (step.ir->sequence && step.sub_seq) {
            this->enter_sequence(step.sub_seq.get());
        }
    }

    void deactivate_step(SequenceState *seq) {
        auto &step = seq->steps[seq->active_step_idx];
        if (step.ir->stage)
            this->clear_self_changed(step.ir->stage->strata);
        else if (step.ir->flow)
            for (const auto &nk: step.ir->flow->nodes)
                this->self_changed_flags[this->node_index[nk]] = 0;
        seq->active_step_idx = NO_INDEX;
    }

    void enter_sequence(SequenceState *seq) {
        if (seq->steps.empty()) return;
        this->activate_step(seq, 0);
    }

    void reset_strata(const ir::Strata &strata) {
        for (const auto &stratum: strata)
            for (const auto &key: stratum) {
                if (this->boundaries.count(key)) continue;
                this->self_changed_flags[this->node_index[key]] = 0;
                this->nodes[key].node->reset();
            }
    }

    void clear_self_changed(const ir::Strata &strata) {
        for (const auto &stratum: strata)
            for (const auto &key: stratum) {
                if (this->boundaries.count(key)) continue;
                this->self_changed_flags[this->node_index[key]] = 0;
            }
    }

    void reset_sequence_state(SequenceState &seq) {
        seq.active_step_idx = NO_INDEX;
        for (auto &step: seq.steps) {
            for (auto &sub: step.sub_seqs)
                this->reset_sequence_state(sub);
            if (step.sub_seq) this->reset_sequence_state(*step.sub_seq);
        }
    }
};
}
