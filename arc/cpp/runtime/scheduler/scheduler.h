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
    /// @brief pre-resolved form of an ir::Edge cached on the source node.
    /// target_idx is the index into the dense node table; conditional
    /// records whether the edge only fires when the source output is
    /// truthy.
    struct OutEdge {
        size_t target_idx;
        bool conditional;
    };

    struct ScopeState;

    /// @brief pre-resolved per-output propagation table. Indexed by a
    /// node-local output ordinal, avoids all string hashing in the hot
    /// path when mark_changed is called with the integer overload.
    struct OutputResolved {
        /// @brief the output param name, retained so the scheduler can
        /// call node->is_output_truthy(param) and so the string overload
        /// of mark_changed can resolve back to this entry.
        std::string param;
        /// @brief outgoing dataflow edges from this output.
        std::vector<OutEdge> edges;
        /// @brief index into marked_flags, or NO_INDEX when no sequential
        /// transition consumes this output.
        size_t mark_handle_idx = NO_INDEX;
        /// @brief scopes whose activation handle is this output; polled
        /// after every truthy mark_changed.
        std::vector<ScopeState *> activates;
    };

    struct Node {
        std::unique_ptr<node::Node> node;
        std::string key;
        size_t idx = 0;
        /// @brief per-output resolved propagation data, dense-indexed by
        /// the node's local output ordinal.
        std::vector<OutputResolved> outputs;
        /// @brief maps an output param name to its index in outputs. Used
        /// by the string overload of mark_changed to resolve to an
        /// ordinal; unused in the integer fast path.
        std::unordered_map<std::string, size_t> output_by_param;
    };

    /// @brief runtime state for one member of a scope. Exactly one of node_key
    /// or scope is populated, mirroring the IR's Member tagged union.
    /// node_idx caches the dense flag-slice index for a NodeRef member; it
    /// is NO_INDEX for scope members.
    struct MemberState {
        std::string key;
        std::string node_key;
        size_t node_idx = NO_INDEX;
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
        /// @brief transition_on_idx[i] is the scheduler-wide marked_flags
        /// index for transition i's on-handle, pre-resolved at construction
        /// so the hot-path evaluation is a single array load. NO_INDEX when
        /// the on-node is unknown, which the hot path treats as never
        /// fires.
        std::vector<size_t> transition_on_idx;
    };

    std::unordered_map<std::string, Node> nodes;
    std::unordered_map<std::string, size_t> node_index;
    std::vector<uint8_t> changed_flags;
    std::vector<uint8_t> self_changed_flags;
    /// @brief holds one flag per (node, param) pair that sources a
    /// sequential-scope transition, indexed by the dense handle IDs
    /// assigned at construction. A non-zero entry means mark_changed was
    /// called with a truthy output during the current cycle. Sequential
    /// transitions fire on a fresh mark rather than on stale cached
    /// truthiness, mirroring the conditional-edge firing semantic of the
    /// pre-Scope scheduler. Marks are consumed when a transition fires
    /// and cleared at end of cycle.
    std::vector<uint8_t> marked_flags;
    /// @brief next index to assign to a freshly-discovered
    /// transition-source handle during construction. After construction
    /// this is the length of marked_flags.
    size_t next_handle_idx = 0;

    /// @brief root mirrors prog.root; always a parallel, always-live scope.
    std::unique_ptr<ScopeState> root;

    x::telem::TimeSpan tolerance;
    errors::Handler error_handler;

    node::Context ctx;
    /// @brief tracks the earliest deadline reported by any node during the
    /// current cycle. Seeded to TimeSpan::max at the start of every next();
    /// exposed to callers via next_deadline().
    x::telem::TimeSpan min_deadline = x::telem::TimeSpan::max();
    /// @brief pointer to the node currently executing, cached so
    /// mark_changed / mark_self_changed callbacks skip the per-call map
    /// lookup. nullptr between cycles.
    Node *curr_node = nullptr;

    ir::IR prog;

    std::unique_ptr<ScopeState> build_scope_state(const ir::Scope *scope) {
        auto state = std::make_unique<ScopeState>();
        state->ir = scope;
        state->mode = scope->mode;
        state->liveness = scope->liveness;

        const auto append_member = [&](const ir::Member &m) {
            MemberState ms;
            ms.key = m.key;
            if (m.node_ref.has_value()) {
                ms.node_key = m.node_ref->key;
                const auto it = this->node_index.find(m.node_ref->key);
                if (it != this->node_index.end()) ms.node_idx = it->second;
            } else if (m.scope)
                ms.scope = this->build_scope_state(&*m.scope);
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

    /// @brief recursively walks the ScopeState tree wiring per-output
    /// propagation tables on each source Node (so mark_changed needs zero
    /// scheduler-wide hash lookups) and the per-sequential-scope
    /// transition ownership / handle-index tables.
    void register_scope(ScopeState *state) {
        if (state->ir->liveness == ir::Liveness::Gated &&
            state->ir->activation.has_value()) {
            const auto node_it = this->nodes.find(state->ir->activation->node);
            if (node_it != this->nodes.end()) {
                const size_t out_idx = this->get_or_create_output(
                    node_it->second,
                    state->ir->activation->param
                );
                node_it->second.outputs[out_idx].activates.push_back(state);
            }
        }
        if (state->mode == ir::ScopeMode::Sequential) {
            state->transition_owner = compute_transition_owners(
                *state,
                state->ir->transitions
            );
            state->transition_on_idx.resize(state->ir->transitions.size(), NO_INDEX);
            for (size_t i = 0; i < state->ir->transitions.size(); ++i) {
                const auto &t = state->ir->transitions[i];
                state->transition_on_idx[i] = this->register_transition_handle(
                    t.on.node,
                    t.on.param
                );
            }
        }
        for (auto &m: state->members)
            if (m.scope) this->register_scope(m.scope.get());
    }

    /// @brief returns the node-local output index for the given param,
    /// allocating a fresh entry in Node::outputs if the param is not yet
    /// known. Used by construction paths that wire edges, transition
    /// sources, and activation sources onto the owning Node.
    size_t get_or_create_output(Node &n, const std::string &param) {
        const auto it = n.output_by_param.find(param);
        if (it != n.output_by_param.end()) return it->second;
        const size_t idx = n.outputs.size();
        n.outputs.push_back(OutputResolved{.param = param});
        n.output_by_param[param] = idx;
        return idx;
    }

    /// @brief assigns a dense marked_flags index to the node-local
    /// output entry and returns it. Idempotent — reused by multiple
    /// transitions returns the same index. Returns NO_INDEX if the node
    /// is unknown.
    size_t
    register_transition_handle(const std::string &node_key, const std::string &param) {
        const auto node_it = this->nodes.find(node_key);
        if (node_it == this->nodes.end()) return NO_INDEX;
        const size_t out_idx = this->get_or_create_output(node_it->second, param);
        auto &out = node_it->second.outputs[out_idx];
        if (out.mark_handle_idx == NO_INDEX)
            out.mark_handle_idx = this->next_handle_idx++;
        return out.mark_handle_idx;
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
        tolerance(tolerance),
        error_handler(std::move(error_handler)),
        prog(std::move(prog)) {
        this->ctx.mark_changed = std::bind_front(&Scheduler::mark_changed, this);
        this->ctx.mark_self_changed = std::bind_front(
            &Scheduler::mark_self_changed,
            this
        );
        this->ctx.set_deadline = [this](const x::telem::TimeSpan d) {
            if (d < this->min_deadline) this->min_deadline = d;
        };
        this->ctx.report_error = std::bind_front(&Scheduler::report_error, this);

        size_t idx = 0;
        for (auto &[key, node]: node_impls) {
            this->node_index[key] = idx;
            Node n{.node = std::move(node), .key = key, .idx = idx};
            // Pre-seed outputs in the node's declared order. Node impls
            // hardcode integer ordinals that match positions in this
            // list; the scheduler's edge/transition/activation wiring
            // below uses get_or_create_output, which finds these
            // pre-seeded entries by name.
            const auto declared = n.node->outputs();
            n.outputs.reserve(declared.size());
            n.output_by_param.reserve(declared.size());
            for (const auto &name: declared) {
                n.output_by_param[name] = n.outputs.size();
                n.outputs.push_back(OutputResolved{.param = name});
            }
            this->nodes[key] = std::move(n);
            ++idx;
        }
        this->changed_flags.resize(idx, 0);
        this->self_changed_flags.resize(idx, 0);

        // Single pass over edges (O(N+E)) replaces per-node edges_from
        // scans (O(N*E)). Resolves each edge's source param to its local
        // output ordinal and the target node key to its dense index so
        // mark_changed needs zero hash lookups per edge.
        for (const auto &edge: this->prog.edges) {
            const auto src_it = this->nodes.find(edge.source.node);
            if (src_it == this->nodes.end()) continue;
            const auto tgt_it = this->node_index.find(edge.target.node);
            if (tgt_it == this->node_index.end()) continue;
            const size_t out_idx = this->get_or_create_output(
                src_it->second,
                edge.source.param
            );
            src_it->second.outputs[out_idx].edges.push_back(
                OutEdge{tgt_it->second, edge.kind == ir::EdgeKind::Conditional}
            );
        }

        this->root = build_scope_state(&this->prog.root);
        this->register_scope(this->root.get());
        this->marked_flags.resize(this->next_handle_idx, 0);
        this->activate_scope(*this->root);
    }

    Scheduler(Scheduler &&) = delete;
    Scheduler &operator=(Scheduler &&) = delete;
    Scheduler(const Scheduler &) = delete;
    Scheduler &operator=(const Scheduler &) = delete;

    void reset() {
        std::ranges::fill(this->changed_flags, 0);
        std::ranges::fill(this->self_changed_flags, 0);
        std::ranges::fill(this->marked_flags, 0);
        this->curr_node = nullptr;
        this->reset_scope_state(*this->root);
        for (const auto &node_state: this->nodes | std::views::values)
            node_state.node->reset();
        this->activate_scope(*this->root);
    }

    void next(const x::telem::TimeSpan elapsed, const node::RunReason reason) {
        this->min_deadline = x::telem::TimeSpan::max();
        this->ctx.elapsed = elapsed;
        this->ctx.tolerance = this->tolerance;
        this->ctx.reason = reason;

        this->walk(*this->root);

        std::ranges::fill(this->changed_flags, 0);
        std::ranges::fill(this->marked_flags, 0);
    }

    [[nodiscard]] x::telem::TimeSpan next_deadline() const {
        return this->min_deadline;
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

    /// @brief iterates the flat state.members slice (which is stored in
    /// phase-flattened order by build_scope_state) in lockstep with the IR
    /// phases so each member access uses its pre-resolved node_idx without
    /// a map lookup.
    void walk_parallel(ScopeState &state) {
        const auto &phases = state.ir->phases;
        size_t flat = 0;
        for (size_t phase_idx = 0; phase_idx < phases.size(); ++phase_idx)
            for (size_t i = 0; i < phases[phase_idx].members.size(); ++i)
                this->execute_member(phase_idx, state.members[flat++]);
    }

    /// @brief executes the active member of a sequential scope, then
    /// loops checking transitions for same-cycle cascading. Bounded by
    /// members+1 so a chain of N members can fire N consecutive
    /// transitions. Sequential members are "always on" while active —
    /// passing phase_idx=0 to execute_member makes the node run
    /// unconditionally, matching how phase-0 parallel members execute
    /// every cycle.
    void walk_sequential(ScopeState &state) {
        const size_t budget = state.members.size() + 1;
        for (size_t i = 0; i < budget; ++i) {
            if (state.active_member == NO_INDEX) return;
            this->execute_member(0, state.members[state.active_member]);
            if (!this->evaluate_transitions(state)) return;
        }
    }

    /// @brief executes a single member of a scope. If the member is a
    /// nested scope, the scope is walked (its activation happens
    /// reactively via mark_changed when the activation source's handle
    /// fires). If the member is a NodeRef, the node runs when
    /// phase_idx==0, when it has a pending upstream change, or when it
    /// was marked self-changed by a previous cycle. Sequential scopes
    /// pass phase_idx=0 to force unconditional execution.
    void execute_member(const size_t phase_idx, MemberState &m) {
        if (m.scope) {
            this->walk(*m.scope);
            return;
        }
        if (m.node_idx == NO_INDEX) return;
        const bool was_self_changed = this->self_changed_flags[m.node_idx] != 0;
        if (was_self_changed) this->self_changed_flags[m.node_idx] = 0;
        if (phase_idx == 0 || this->changed_flags[m.node_idx] || was_self_changed)
            this->run_node(m.node_key);
    }

    /// @brief dispatches a node's next method. Activation polling happens
    /// inside mark_changed, which is called by the node during execution
    /// whenever it announces an output update.
    void run_node(const std::string &key) {
        const auto it = this->nodes.find(key);
        if (it == this->nodes.end()) return;
        this->curr_node = &it->second;
        it->second.node->next(this->ctx);
    }

    /// @brief walks the sequential scope's transitions in source order and
    /// applies the first one whose on-handle was freshly marked changed
    /// with a truthy value during the current cycle and whose ownership
    /// matches the currently-active member (or lies outside the scope). A
    /// stale-truthy source that did not re-mark this cycle does not fire
    /// the transition; this mirrors the conditional-edge firing semantic
    /// of the pre-Scope scheduler and prevents wait/interval/latched
    /// comparisons (which mark once per event) from driving spurious
    /// repeated transitions on later cycles. Firing consumes the mark, so
    /// a single MarkChanged call produces at most one transition firing
    /// per cycle. Reports whether a transition fired so the caller can
    /// drive the convergence loop.
    bool evaluate_transitions(ScopeState &state) {
        const auto &transitions = state.ir->transitions;
        for (size_t i = 0; i < transitions.size(); ++i) {
            const size_t owner = state.transition_owner[i];
            if (owner != NO_INDEX && owner != state.active_member) continue;
            const size_t handle_idx = state.transition_on_idx[i];
            if (handle_idx == NO_INDEX || !this->marked_flags[handle_idx]) continue;
            if (!this->is_handle_truthy(transitions[i].on)) continue;
            this->marked_flags[handle_idx] = 0;
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
    /// scopes become active at member 0. Parallel scopes reset every
    /// direct NodeRef member and cascade-activate every nested gated
    /// scope member whose Activation handle is unset. Gated children with
    /// an Activation handle wait for that handle to fire via
    /// mark_changed; they are not cascade-activated by their parent.
    /// This rule applies uniformly at the root and at every nested
    /// parallel scope.
    void activate_scope(ScopeState &state) {
        state.active = true;
        if (state.mode == ir::ScopeMode::Sequential) {
            if (!state.members.empty()) this->activate_sequential_member(state, 0);
            return;
        }
        for (auto &m: state.members) {
            if (m.is_node_ref()) {
                if (m.node_idx != NO_INDEX) this->self_changed_flags[m.node_idx] = 0;
                if (const auto n = this->nodes.find(m.node_key); n != this->nodes.end())
                    n->second.node->reset();
            } else if (m.scope && m.scope->ir->liveness == ir::Liveness::Gated &&
                       !m.scope->ir->activation.has_value()) {
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
            if (m.node_idx != NO_INDEX) this->self_changed_flags[m.node_idx] = 0;
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
            if (m.node_idx != NO_INDEX) this->self_changed_flags[m.node_idx] = 0;
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
            if (m.is_node_ref() && m.node_idx != NO_INDEX)
                this->self_changed_flags[m.node_idx] = 0;
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
    /// downstream nodes and fires any gated scope activations sourced
    /// from this output. Pure array access — zero hash lookups.
    /// Continuous edges always propagate; conditional edges only
    /// propagate when the source output is truthy.
    void mark_changed(const size_t output_idx) {
        auto &n = *this->curr_node;
        if (output_idx >= n.outputs.size()) return;
        auto &out = n.outputs[output_idx];
        const bool truthy = n.node->is_output_truthy(out.param);
        if (truthy && out.mark_handle_idx != NO_INDEX)
            this->marked_flags[out.mark_handle_idx] = 1;
        for (const auto &edge: out.edges)
            if (!edge.conditional || truthy) this->changed_flags[edge.target_idx] = 1;
        for (auto *scope: out.activates)
            if (!scope->active) this->activate_scope(*scope);
    }

    void mark_self_changed() { this->self_changed_flags[this->curr_node->idx] = 1; }
};

}
