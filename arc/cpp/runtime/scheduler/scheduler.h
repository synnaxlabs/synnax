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
#include <functional>
#include <memory>
#include <ranges>
#include <string>
#include <unordered_map>
#include <vector>

#include "glog/logging.h"

#include "x/cpp/errors/errors.h"

#include "arc/cpp/ir/ir.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/node/node.h"

namespace arc::runtime::scheduler {

static constexpr size_t NO_INDEX = ~size_t{0};

/// @brief executes an Arc program by walking the Layer-2 Scope tree attached to
/// the IR's Root. Each cycle the scheduler walks the tree, executes the active
/// members of each reachable scope, evaluates transitions on sequential scopes,
/// and activates gated scopes whose activation handle has fired via
/// MarkChanged.
class Scheduler {
    struct Node {
        std::unordered_map<std::string, std::vector<ir::Edge>> output_edges;
        std::unique_ptr<node::Node> node;
    };

    struct ScopeState;

    /// @brief runtime state for one member of a scope. Exactly one of node_key
    /// or scope is populated, mirroring the IR's Member tagged union.
    struct MemberState {
        std::string key;
        std::string node_key;
        std::unique_ptr<ScopeState> scope;

        [[nodiscard]] bool is_node_ref() const { return scope == nullptr; }
    };

    /// @brief runtime state for one scope in the program's Scope tree.
    /// Mirrors the static ir::Scope with mutable activation bookkeeping.
    struct ScopeState {
        const ir::Scope *ir = nullptr;
        ir::ScopeMode mode = ir::ScopeMode::Unspecified;
        ir::Liveness liveness = ir::Liveness::Unspecified;
        bool active = false;
        size_t active_member = NO_INDEX;
        std::vector<MemberState> members;
        std::unordered_map<std::string, size_t> member_by_key;
        /// @brief transition_owner[i] is the index of the member whose node
        /// ownership sources transition i's on-handle, or NO_INDEX when the
        /// handle is sourced from outside the scope (e.g., a module-scope
        /// channel read that drives cross-scope activation). Only
        /// transitions rooted in the active member — or entirely outside
        /// the scope — are evaluated each cycle.
        std::vector<size_t> transition_owner;
    };

    std::unordered_map<std::string, Node> nodes;
    std::unordered_map<std::string, size_t> node_index;
    std::vector<uint8_t> changed_flags;
    std::vector<uint8_t> self_changed_flags;

    /// @brief root mirrors prog.root; always a parallel, always-live scope.
    std::unique_ptr<ScopeState> root;

    /// @brief activations_by_source indexes scopes whose activation handle is
    /// sourced by the keyed node. When a node calls MarkChanged on an output
    /// that matches a scope's activation handle, the scope activates.
    std::unordered_map<std::string, std::vector<ScopeState *>> activations_by_source;

    /// @brief max_convergence_iter bounds the per-sequential-scope transition
    /// loop. A sequential scope may cascade through its own transitions within
    /// a single scheduler cycle up to this many times before the walk returns.
    size_t max_convergence_iter = 0;

    x::telem::TimeSpan tolerance_;
    errors::Handler error_handler;

    node::Context ctx;
    x::telem::TimeSpan next_deadline_ = x::telem::TimeSpan::max();
    const std::string *curr_node_ptr = nullptr;

    ir::IR prog_;

    static std::unique_ptr<ScopeState> build_scope_state(const ir::Scope *scope) {
        auto state = std::make_unique<ScopeState>();
        state->ir = scope;
        state->mode = scope->mode;
        state->liveness = scope->liveness;

        const auto append_member = [&](const ir::Member &m) {
            MemberState ms;
            ms.key = m.key;
            if (m.node_ref.has_value())
                ms.node_key = m.node_ref->key;
            else if (m.scope)
                ms.scope = build_scope_state(&*m.scope);
            state->members.push_back(std::move(ms));
        };

        if (scope->mode == ir::ScopeMode::Parallel) {
            for (const auto &phase: scope->phases)
                for (const auto &m: phase.members)
                    append_member(m);
        } else if (scope->mode == ir::ScopeMode::Sequential) {
            for (const auto &m: scope->members)
                append_member(m);
        }

        for (size_t i = 0; i < state->members.size(); ++i)
            state->member_by_key[state->members[i].key] = i;
        return state;
    }

    /// @brief recursively walks the ScopeState tree building the activation
    /// lookup (so runtime activation firing is O(1) per node execution) and
    /// the per-sequential-scope transition ownership table (so transitions
    /// only fire for the currently-active member).
    void register_scope(ScopeState *state) {
        if (state->ir->liveness == ir::Liveness::Gated &&
            state->ir->activation.has_value()) {
            this->activations_by_source[state->ir->activation->node].push_back(state);
        }
        if (state->mode == ir::ScopeMode::Sequential) {
            this->max_convergence_iter += state->ir->members.size() + 1;
            state->transition_owner = compute_transition_owners(
                *state,
                state->ir->transitions
            );
        }
        for (auto &m: state->members)
            if (m.scope) this->register_scope(m.scope.get());
    }

    /// @brief computes which member of state owns the source node of each
    /// transition's on-handle. Returns NO_INDEX for transitions whose
    /// on-node sits outside the scope entirely.
    static std::vector<size_t> compute_transition_owners(
        const ScopeState &state,
        const std::vector<ir::Transition> &transitions
    ) {
        std::unordered_map<std::string, size_t> node_to_member;
        for (size_t i = 0; i < state.members.size(); ++i)
            collect_member_nodes(state.members[i], i, node_to_member);

        std::vector<size_t> owners(transitions.size(), NO_INDEX);
        for (size_t i = 0; i < transitions.size(); ++i) {
            const auto it = node_to_member.find(transitions[i].on.node);
            if (it != node_to_member.end()) owners[i] = it->second;
        }
        return owners;
    }

    static void collect_member_nodes(
        const MemberState &m,
        const size_t idx,
        std::unordered_map<std::string, size_t> &out
    ) {
        if (m.is_node_ref()) {
            out[m.node_key] = idx;
            return;
        }
        if (m.scope) collect_scope_nodes(*m.scope, idx, out);
    }

    static void collect_scope_nodes(
        const ScopeState &s,
        const size_t idx,
        std::unordered_map<std::string, size_t> &out
    ) {
        for (const auto &inner: s.members) {
            if (inner.is_node_ref())
                out[inner.node_key] = idx;
            else if (inner.scope)
                collect_scope_nodes(*inner.scope, idx, out);
        }
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

        this->root = build_scope_state(&this->prog_.root);
        this->register_scope(this->root.get());
        this->activate_scope(*this->root);
    }

    Scheduler(Scheduler &&) = delete;
    Scheduler &operator=(Scheduler &&) = delete;
    Scheduler(const Scheduler &) = delete;
    Scheduler &operator=(const Scheduler &) = delete;

    void reset() {
        std::ranges::fill(this->changed_flags, 0);
        std::ranges::fill(this->self_changed_flags, 0);
        this->curr_node_ptr = nullptr;
        this->reset_scope_state(*this->root);
        for (const auto &node_state: this->nodes | std::views::values)
            node_state.node->reset();
        this->activate_scope(*this->root);
    }

    void next(const x::telem::TimeSpan elapsed, const node::RunReason reason) {
        this->next_deadline_ = x::telem::TimeSpan::max();
        this->ctx.elapsed = elapsed;
        this->ctx.tolerance = this->tolerance_;
        this->ctx.reason = reason;

        this->walk(*this->root);

        std::ranges::fill(this->changed_flags, 0);
    }

    [[nodiscard]] x::telem::TimeSpan next_deadline() const {
        return this->next_deadline_;
    }

    void mark_node_changed(const std::string &node_key) {
        const auto it = this->node_index.find(node_key);
        if (it != this->node_index.end()) this->changed_flags[it->second] = 1;
    }

private:
    /// @brief executes one pass over a scope. Parallel scopes iterate all
    /// phases; sequential scopes internally loop to converge cascading
    /// transitions within a single cycle.
    void walk(ScopeState &state) {
        if (!state.active) return;
        if (state.mode == ir::ScopeMode::Parallel)
            this->walk_parallel(state);
        else if (state.mode == ir::ScopeMode::Sequential)
            this->walk_sequential(state);
    }

    void walk_parallel(ScopeState &state) {
        const auto &phases = state.ir->phases;
        for (size_t phase_idx = 0; phase_idx < phases.size(); ++phase_idx)
            for (const auto &m: phases[phase_idx].members)
                this->execute_parallel_member(phase_idx, state, m);
    }

    void walk_sequential(ScopeState &state) {
        if (state.active_member == NO_INDEX) return;
        const size_t budget = state.members.size() + 1;
        for (size_t i = 0; i < budget; ++i) {
            if (state.active_member == NO_INDEX) return;
            this->execute_sequential_member(state.members[state.active_member]);
            if (!this->evaluate_transitions(state)) return;
        }
    }

    /// @brief executes a single member of a parallel scope. If the member is
    /// a nested scope, the scope is walked (its activation happens
    /// reactively via mark_changed when the activation source's handle
    /// fires). If the member is a NodeRef, the node runs subject to the
    /// usual phase-0 / changed / self-changed filtering.
    void execute_parallel_member(
        const size_t phase_idx,
        ScopeState &parent,
        const ir::Member &m
    ) {
        if (m.scope) {
            const auto it = parent.member_by_key.find(m.key);
            if (it == parent.member_by_key.end()) return;
            auto *child = parent.members[it->second].scope.get();
            if (child == nullptr) return;
            this->walk(*child);
            return;
        }
        if (!m.node_ref.has_value()) return;
        const auto &key = m.node_ref->key;
        const auto idx_it = this->node_index.find(key);
        if (idx_it == this->node_index.end()) return;
        const size_t idx = idx_it->second;
        const bool was_self_changed = this->self_changed_flags[idx] != 0;
        if (was_self_changed) this->self_changed_flags[idx] = 0;
        if (phase_idx == 0 || this->changed_flags[idx] || was_self_changed)
            this->run_node(key);
    }

    /// @brief unconditionally executes a sequential scope's active member.
    /// Sequential members are "always on" while active — they do not wait
    /// for an upstream change signal.
    void execute_sequential_member(const MemberState &m) {
        if (!m.is_node_ref()) {
            if (m.scope) this->walk(*m.scope);
            return;
        }
        const auto it = this->node_index.find(m.node_key);
        if (it != this->node_index.end()) this->self_changed_flags[it->second] = 0;
        this->run_node(m.node_key);
    }

    /// @brief dispatches a node's next method. Activation polling happens
    /// inside mark_changed, which is called by the node during execution
    /// whenever it announces an output update.
    void run_node(const std::string &key) {
        const auto it = this->nodes.find(key);
        if (it == this->nodes.end()) return;
        this->curr_node_ptr = &key;
        it->second.node->next(this->ctx);
    }

    /// @brief walks the sequential scope's transitions in source order and
    /// applies the first one whose on-handle is truthy and whose ownership
    /// matches the currently-active member (or lies outside the scope).
    /// Reports whether a transition fired so the caller can drive the
    /// convergence loop.
    bool evaluate_transitions(ScopeState &state) {
        const auto &transitions = state.ir->transitions;
        for (size_t i = 0; i < transitions.size(); ++i) {
            const size_t owner = state.transition_owner[i];
            if (owner != NO_INDEX && owner != state.active_member) continue;
            if (!this->is_handle_truthy(transitions[i].on)) continue;
            if (state.active_member != NO_INDEX)
                this->deactivate_member(state, state.active_member);
            const auto &target = transitions[i].target;
            if (target.exit.has_value() && *target.exit) {
                this->deactivate_scope(state);
            } else if (target.member_key.has_value()) {
                const auto mit = state.member_by_key.find(*target.member_key);
                if (mit == state.member_by_key.end()) return false;
                this->activate_sequential_member(state, mit->second);
            }
            return true;
        }
        return false;
    }

    /// @brief marks a scope active and primes its member(s). Sequential
    /// scopes become active at member 0. Parallel + gated scopes reset
    /// every direct NodeRef member and cascade-activate every nested gated
    /// scope member. Parallel + always-live scopes (the root) only reset
    /// direct NodeRef members; their gated children activate lazily when
    /// their activation handle fires.
    void activate_scope(ScopeState &state) {
        state.active = true;
        if (state.mode == ir::ScopeMode::Sequential) {
            if (!state.members.empty()) this->activate_sequential_member(state, 0);
            return;
        }
        for (auto &m: state.members) {
            if (m.is_node_ref()) {
                const auto it = this->node_index.find(m.node_key);
                if (it != this->node_index.end())
                    this->self_changed_flags[it->second] = 0;
                if (const auto n = this->nodes.find(m.node_key); n != this->nodes.end())
                    n->second.node->reset();
            } else if (m.scope && state.liveness == ir::Liveness::Gated) {
                this->activate_scope(*m.scope);
            }
        }
    }

    /// @brief moves a sequential scope's active pointer to idx and resets
    /// (or cascade-activates) the newly-active member.
    void activate_sequential_member(ScopeState &state, const size_t idx) {
        state.active_member = idx;
        auto &m = state.members[idx];
        if (m.is_node_ref()) {
            const auto it = this->node_index.find(m.node_key);
            if (it != this->node_index.end()) this->self_changed_flags[it->second] = 0;
            if (const auto n = this->nodes.find(m.node_key); n != this->nodes.end())
                n->second.node->reset();
            return;
        }
        if (m.scope) this->activate_scope(*m.scope);
    }

    /// @brief clears self-changed for the member's owned node and, if the
    /// member wraps a nested scope, marks that scope inactive (without
    /// recursing into its own members per RFC 0035 §3.3).
    void deactivate_member(const ScopeState &state, const size_t idx) {
        const auto &m = state.members[idx];
        if (m.is_node_ref()) {
            const auto it = this->node_index.find(m.node_key);
            if (it != this->node_index.end()) this->self_changed_flags[it->second] = 0;
            return;
        }
        if (m.scope) this->deactivate_scope(*m.scope);
    }

    /// @brief marks a scope inactive and clears self-changed for its direct
    /// NodeRef members. Per RFC 0035 §3.3 deactivation does not cascade —
    /// nested scope members retain their frozen state until they are
    /// reactivated from the parent's activate_scope, which overwrites it.
    void deactivate_scope(ScopeState &state) {
        if (state.mode == ir::ScopeMode::Sequential) state.active_member = NO_INDEX;
        for (const auto &m: state.members)
            if (m.is_node_ref()) {
                const auto it = this->node_index.find(m.node_key);
                if (it != this->node_index.end())
                    this->self_changed_flags[it->second] = 0;
            }
        state.active = false;
    }

    /// @brief reports whether the node referenced by h has a truthy output
    /// on the handle's parameter. Missing nodes produce false.
    bool is_handle_truthy(const ir::Handle &h) const {
        const auto it = this->nodes.find(h.node);
        if (it == this->nodes.end()) return false;
        return it->second.node->is_output_truthy(h.param);
    }

    /// @brief resets every scope state in the tree rooted at state to
    /// inactive with active_member = NO_INDEX. Does not call node reset —
    /// the top-level reset path handles that for all nodes.
    void reset_scope_state(ScopeState &state) {
        state.active = false;
        state.active_member = NO_INDEX;
        for (auto &m: state.members)
            if (m.scope) this->reset_scope_state(*m.scope);
    }

    void report_error(const x::errors::Error &e) const {
        LOG(ERROR) << "[arc.scheduler] node encountered error: " << e;
        this->error_handler(e);
    }

    /// @brief propagates changes from the current node's output to
    /// downstream nodes and fires any gated scope activations sourced from
    /// this output. Continuous edges always propagate; conditional edges
    /// only propagate when the source output is truthy. Activations fire on
    /// any MarkChanged for the matching handle — equivalent to the
    /// continuous-edge semantics of the pre-Scope IR.
    void mark_changed(const std::string &param) {
        auto &n = this->nodes[*this->curr_node_ptr];
        for (const auto &edge: n.output_edges[param]) {
            if (edge.kind != ir::EdgeKind::Conditional ||
                n.node->is_output_truthy(param))
                this->changed_flags[this->node_index[edge.target.node]] = 1;
        }
        const auto it = this->activations_by_source.find(*this->curr_node_ptr);
        if (it == this->activations_by_source.end()) return;
        for (auto *scope: it->second) {
            if (scope->active) continue;
            if (!scope->ir->activation.has_value()) continue;
            if (scope->ir->activation->param != param) continue;
            this->activate_scope(*scope);
        }
    }

    void mark_self_changed() {
        this->self_changed_flags[this->node_index[*this->curr_node_ptr]] = 1;
    }
};

}
