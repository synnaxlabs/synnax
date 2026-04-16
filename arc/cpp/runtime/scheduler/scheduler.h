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

namespace detail {
class Builder;
}

/// @brief executes an Arc program by walking the Layer-2 Scope tree attached to
/// the IR's Root. Each cycle the scheduler walks the tree, executes the active
/// members of each reachable scope, evaluates transitions on sequential scopes,
/// and activates gated scopes whose activation handle has fired via
/// mark_changed.
class Scheduler {
    friend class detail::Builder;

    /// @brief pre-resolved form of an ir::Edge cached on the source node.
    struct OutEdge {
        /// @brief destination node's flag-slice index.
        size_t target_idx;
        /// @brief edge only fires when the source output is truthy.
        bool conditional;
    };

    struct ScopeState;

    /// @brief per-output propagation table built at construction and indexed
    /// by the node-local output ordinal. Hot-path access is pure array
    /// indexing.
    struct OutputResolved {
        /// @brief outgoing dataflow edges from this output.
        std::vector<OutEdge> edges;
        /// @brief marked_flags index for sequential-scope transitions
        /// sourced from this output, or NO_INDEX if none.
        size_t mark_handle_idx = NO_INDEX;
        /// @brief gated scopes whose activation handle is this output;
        /// activated by mark_changed when the source fires.
        std::vector<ScopeState *> activates;
    };

    /// @brief pairs a runtime node with its pre-resolved per-output
    /// propagation table.
    struct Node {
        /// @brief owned runtime node implementation.
        std::unique_ptr<node::Node> node;
        /// @brief IR node key, retained for error reporting.
        std::string key;
        /// @brief position in changed_flags / self_changed_flags.
        size_t idx = 0;
        /// @brief propagation table indexed by output ordinal.
        std::vector<OutputResolved> outputs;
    };

    /// @brief mirrors an ir::Member: exactly one of node or scope is set.
    /// An unresolved node key leaves both nullptr and is skipped at execution.
    struct MemberState {
        /// @brief IR member key, used for `=> name` transition lookups.
        std::string key;
        /// @brief resolved leaf-node target, nullptr for scope members and
        /// unresolved node keys. Points into Scheduler::nodes; remains valid
        /// for the Scheduler's lifetime because nodes is reserved exactly
        /// at construction and never grows.
        Node *node = nullptr;
        /// @brief nested scope state, nullptr for leaf-node members.
        std::unique_ptr<ScopeState> scope;

        [[nodiscard]] bool is_node() const { return scope == nullptr; }
    };

    /// @brief runtime mirror of an ir::Scope, holding activation bookkeeping
    /// for one scope.
    struct ScopeState {
        /// @brief static IR scope this state mirrors.
        const ir::Scope *ir = nullptr;
        /// @brief gates whether walk descends into this scope.
        bool active = false;
        /// @brief running sequential step's index, or NO_INDEX when
        /// inactive or in a parallel scope.
        size_t active_step = NO_INDEX;
        /// @brief members flattened in execution order: stratum-major for
        /// parallel scopes, sequence order for sequential.
        std::vector<MemberState> members;
        /// @brief resolves `=> name` transition targets to step indices.
        std::unordered_map<std::string, size_t> member_by_key;
        /// @brief transition_owner[i] is the step index that owns
        /// transition i's `on`-handle source, or NO_INDEX if the source is
        /// outside this scope. Transitions owned by inactive steps are
        /// skipped each cycle.
        std::vector<size_t> transition_owner;
        /// @brief transitions_for_step[s] holds transition indices to
        /// evaluate when step s is active, in source order. Each entry
        /// is the union of:
        ///   - transitions whose owner is s (sourced inside that step)
        ///   - transitions whose owner is NO_INDEX (sourced outside the
        ///     scope, always evaluated)
        /// Pre-computed once at construction so evaluate_transitions
        /// iterates a short per-step list instead of the full
        /// transitions slice. Reduces sequential-cascade work from O(N²)
        /// to O(N·K) where K is the typical per-step transition count
        /// (almost always 1).
        std::vector<std::vector<size_t>> transitions_for_step;
        /// @brief transition_on_idx[i] is the marked_flags index for
        /// transition i's `on`-handle, or NO_INDEX if unresolved.
        std::vector<size_t> transition_on_idx;
        /// @brief transition_on_node[i] is the source node of transition
        /// i's `on`-handle, or nullptr if unresolved.
        std::vector<Node *> transition_on_node;
        /// @brief transition_on_output_idx[i] is the output ordinal on
        /// transition_on_node[i] for transition i's `on`-handle, or
        /// NO_INDEX if unresolved. Paired with transition_on_node[i] so
        /// the truthy check is a direct virtual call.
        std::vector<size_t> transition_on_output_idx;
    };

    /// @brief owns every runtime Node wrapper. Sized exactly at construction
    /// via reserve + emplace_back so raw Node* pointers held by MemberState
    /// and ScopeState::transition_on_node remain valid for the Scheduler's
    /// lifetime. Never grown after construction.
    std::vector<Node> nodes;
    /// @brief changed_flags[i] is set when node i has a pending upstream
    /// change for the current cycle. Cleared at end of cycle.
    std::vector<uint8_t> changed_flags;
    /// @brief self_changed_flags[i] is set by node i via mark_self_changed
    /// to request replay on the next cycle. Cleared when the replay runs
    /// or when the owning member is deactivated.
    std::vector<uint8_t> self_changed_flags;
    /// @brief marked_flags[i] is set when the (node, output) pair behind
    /// transition handle i fired truthy this cycle. Cleared at end of
    /// cycle so transitions fire on fresh marks, not stale truthiness.
    std::vector<uint8_t> marked_flags;
    /// @brief program's parallel + always-live root scope.
    std::unique_ptr<ScopeState> root;
    /// @brief how early a timer-based node may fire relative to its deadline.
    x::telem::TimeSpan tolerance;
    /// @brief receives errors raised by nodes via ctx.report_error.
    errors::Handler error_handler;
    /// @brief node::Context passed to every node's next; rebound per cycle
    /// with the latest elapsed time and run reason.
    node::Context ctx;
    /// @brief earliest deadline reported by any node during the current
    /// cycle. Reset to TimeSpan::max at the start of every next();
    /// exposed via next_deadline().
    x::telem::TimeSpan min_deadline = x::telem::TimeSpan::max();
    /// @brief node whose next is currently executing, cached so
    /// mark_changed / mark_self_changed callbacks know whom they came
    /// from. nullptr between cycles.
    Node *curr_node = nullptr;
    /// @brief owns the IR by value so the ir::Scope * pointers held by
    /// every ScopeState remain valid for the Scheduler's lifetime.
    ir::IR prog;

public:
    Scheduler(
        ir::IR prog,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls,
        x::telem::TimeSpan tolerance,
        errors::Handler error_handler = errors::noop_handler
    );

    Scheduler(Scheduler &&) = delete;
    Scheduler &operator=(Scheduler &&) = delete;
    Scheduler(const Scheduler &) = delete;
    Scheduler &operator=(const Scheduler &) = delete;

    /// @brief clears all per-cycle state, calls reset on every node, and
    /// re-activates the root scope. Used by the runtime when stopping a
    /// program so subsequent runs start from a clean state.
    void reset() {
        std::ranges::fill(this->changed_flags, 0);
        std::ranges::fill(this->self_changed_flags, 0);
        std::ranges::fill(this->marked_flags, 0);
        this->curr_node = nullptr;
        this->reset_scope_state(*this->root);
        for (auto &n: this->nodes)
            n.node->reset();
        this->activate_scope(*this->root);
    }

    /// @brief executes one cycle of the reactive computation. Nodes with
    /// pending changes execute in phase order; sequential scopes advance
    /// via their transitions; gated scopes activate when their activation
    /// handle fires.
    void next(const x::telem::TimeSpan elapsed, const node::RunReason reason) {
        this->min_deadline = x::telem::TimeSpan::max();
        this->ctx.elapsed = elapsed;
        this->ctx.tolerance = this->tolerance;
        this->ctx.reason = reason;

        this->walk(*this->root);

        std::ranges::fill(this->changed_flags, 0);
        std::ranges::fill(this->marked_flags, 0);
    }

    /// @brief earliest deadline reported by any node during the previous
    /// next call.
    [[nodiscard]] x::telem::TimeSpan next_deadline() const {
        return this->min_deadline;
    }

private:
    /// @brief executes one pass over a scope; no-op if inactive.
    void walk(ScopeState &state) {
        if (!state.active) return;
        if (state.ir->mode == ir::ScopeMode::Sequential) {
            this->walk_sequential(state);
            return;
        }
        this->walk_parallel(state);
    }

    /// @brief runs every member of a parallel scope in stratum order.
    /// Members are stored stratum-flattened in state.members; state.ir->strata
    /// is read only for the stratum-index argument passed to execute_member.
    void walk_parallel(ScopeState &state) {
        const auto &strata = state.ir->strata;
        size_t flat = 0;
        for (size_t stratum_idx = 0; stratum_idx < strata.size(); ++stratum_idx)
            for (size_t i = 0; i < strata[stratum_idx].size(); ++i)
                this->execute_member(stratum_idx, state.members[flat++]);
    }

    /// @brief executes the active step and loops on its transitions for
    /// same-cycle cascading. Bounded by members+1 so a chain of N steps
    /// can fire N transitions. stratum_idx=0 forces the active step to run
    /// unconditionally, matching stratum-0 parallel semantics.
    void walk_sequential(ScopeState &state) {
        const size_t budget = state.members.size() + 1;
        for (size_t i = 0; i < budget; ++i) {
            if (state.active_step == NO_INDEX) return;
            this->execute_member(0, state.members[state.active_step]);
            if (!this->evaluate_transitions(state)) return;
        }
    }

    /// @brief walks a nested-scope member or runs a leaf-node member.
    /// A leaf runs when stratum_idx==0, when changed_flags is set, or when
    /// the node was self-changed on a prior cycle.
    void execute_member(const size_t stratum_idx, MemberState &m) {
        if (m.scope) {
            this->walk(*m.scope);
            return;
        }
        if (m.node == nullptr) return;
        const size_t idx = m.node->idx;
        const bool was_self_changed = this->self_changed_flags[idx] != 0;
        if (was_self_changed) this->self_changed_flags[idx] = 0;
        if (stratum_idx == 0 || this->changed_flags[idx] || was_self_changed) {
            this->curr_node = m.node;
            this->curr_node->node->next(this->ctx);
        }
    }

    /// @brief clears self-changed and calls reset on m's node. No-op if m is
    /// a scope member or an unresolved node-key.
    void reset_leaf_node(MemberState &m) {
        if (m.is_node() && m.node) {
            this->self_changed_flags[m.node->idx] = 0;
            m.node->node->reset();
        }
    }

    /// @brief clears self-changed on m's node. No-op if m is a scope member
    /// or an unresolved node-key.
    void clear_leaf_node_self_changed(MemberState &m) {
        if (m.is_node() && m.node) this->self_changed_flags[m.node->idx] = 0;
    }

    /// @brief fires the first transition whose `on` handle was freshly
    /// marked truthy by the active step this cycle. Inactive-owner
    /// transitions and stale truthiness without a fresh mark are both
    /// ignored — the latter prevents latched comparisons from driving
    /// repeat transitions. Iterates the pre-filtered transitions_for_step
    /// list for the active step, which interleaves external transitions
    /// and the active step's own transitions in source order. Returns
    /// true if a transition fired.
    bool evaluate_transitions(ScopeState &state) {
        if (state.active_step == NO_INDEX ||
            state.active_step >= state.transitions_for_step.size())
            return false;
        const auto &transitions = state.ir->transitions;
        for (const size_t i: state.transitions_for_step[state.active_step]) {
            const size_t handle_idx = state.transition_on_idx[i];
            if (handle_idx == NO_INDEX || !this->marked_flags[handle_idx]) continue;
            if (!state.transition_on_node[i]->node->is_output_truthy(
                    state.transition_on_output_idx[i]
                ))
                continue;
            this->marked_flags[handle_idx] = 0;
            if (state.active_step != NO_INDEX)
                this->deactivate_step(state.members[state.active_step]);
            const auto &target_key = transitions[i].target_key;
            if (!target_key.has_value()) {
                this->deactivate_scope(state);
            } else {
                const auto mit = state.member_by_key.find(*target_key);
                if (mit == state.member_by_key.end()) return false;
                this->activate_sequential_step(state, mit->second);
            }
            return true;
        }
        return false;
    }

    /// @brief marks a scope active and primes its members. Sequential
    /// scopes activate step 0; parallel scopes reset every leaf-node
    /// member and cascade-activate nested gated scopes that have no
    /// Activation handle — gated children with a handle wait for it to
    /// fire via mark_changed.
    void activate_scope(ScopeState &state) {
        state.active = true;
        if (state.ir->mode == ir::ScopeMode::Sequential) {
            if (!state.members.empty()) this->activate_sequential_step(state, 0);
            return;
        }
        for (auto &m: state.members) {
            if (m.is_node()) {
                this->reset_leaf_node(m);
                continue;
            }
            if (m.scope && m.scope->ir->liveness == ir::Liveness::Gated &&
                !m.scope->ir->activation.has_value())
                this->activate_scope(*m.scope);
        }
    }

    /// @brief points the active pointer at idx and resets (or
    /// cascade-activates) that step.
    void activate_sequential_step(ScopeState &state, const size_t idx) {
        state.active_step = idx;
        auto &m = state.members[idx];
        if (m.is_node()) {
            this->reset_leaf_node(m);
            return;
        }
        if (m.scope) this->activate_scope(*m.scope);
    }

    /// @brief clears self-changed for the step's node, or marks a nested
    /// scope inactive. Nested-scope state freezes and is overwritten on
    /// the next parent activation.
    void deactivate_step(MemberState &m) {
        if (m.scope) {
            this->deactivate_scope(*m.scope);
            return;
        }
        this->clear_leaf_node_self_changed(m);
    }

    /// @brief marks a scope inactive and clears self-changed on its direct
    /// leaf-node members. Does not recurse — nested scope state freezes
    /// until the next activation overwrites it.
    void deactivate_scope(ScopeState &state) {
        if (state.ir->mode == ir::ScopeMode::Sequential) state.active_step = NO_INDEX;
        for (auto &m: state.members)
            this->clear_leaf_node_self_changed(m);
        state.active = false;
    }

    /// @brief recursively resets every scope state in the tree to inactive
    /// with active_step = NO_INDEX. Does not call node reset — reset()
    /// handles that for all nodes via direct iteration.
    void reset_scope_state(ScopeState &state) {
        state.active = false;
        state.active_step = NO_INDEX;
        for (auto &m: state.members)
            if (m.scope) this->reset_scope_state(*m.scope);
    }

    void report_error(const x::errors::Error &e) const {
        LOG(ERROR) << "[arc.scheduler] node encountered error: " << e;
        this->error_handler(e);
    }

    /// @brief propagates the current node's output to downstream nodes,
    /// records a fresh transition mark when truthy, and fires any gated
    /// scope activations attached to this output. Conditional edges
    /// propagate only when the source is truthy.
    void mark_changed(const size_t output_idx) {
        auto &n = *this->curr_node;
        if (output_idx >= n.outputs.size()) return;
        auto &out = n.outputs[output_idx];
        const bool truthy = n.node->is_output_truthy(output_idx);
        if (truthy && out.mark_handle_idx != NO_INDEX)
            this->marked_flags[out.mark_handle_idx] = 1;
        for (const auto &edge: out.edges)
            if (!edge.conditional || truthy) this->changed_flags[edge.target_idx] = 1;
        for (auto *scope: out.activates)
            if (!scope->active) this->activate_scope(*scope);
    }

    void mark_self_changed() { this->self_changed_flags[this->curr_node->idx] = 1; }
};

namespace detail {

/// @brief assembles a Scheduler from a compiled IR. Owns every piece of
/// state needed only during wiring — the node-key map, the per-node
/// param-to-ordinal maps, and the running marked_flags handle counter — so
/// the resulting Scheduler holds only what the hot path consults. Builder
/// state is destroyed when build returns; raw Node* pointers stay valid
/// because Scheduler::nodes is reserved exactly at construction and never
/// grows.
class Builder {
public:
    void build(
        Scheduler &s,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls
    ) {
        this->s = &s;
        this->populate_nodes(node_impls);
        this->wire_edges();
        s.root = this->build_scope_state(&s.prog.root);
        this->register_scope(*s.root);
        s.marked_flags.assign(this->next_handle_idx, 0);
        s.activate_scope(*s.root);
    }

private:
    /// @brief Scheduler being assembled; held as a raw pointer for the
    /// duration of build.
    Scheduler *s = nullptr;
    /// @brief node key to its wrapper pointer (into Scheduler::nodes).
    std::unordered_map<std::string, Scheduler::Node *> nodes_by_key;
    /// @brief per-node "output name → ordinal" map used to find or
    /// allocate matching entries in Node::outputs while wiring. Kept off
    /// Scheduler::Node so the long-lived wrapper stays lean.
    std::unordered_map<Scheduler::Node *, std::unordered_map<std::string, size_t>>
        output_by_param;
    /// @brief next index to assign to a freshly-discovered transition-source
    /// handle. After build, this is the final length of marked_flags.
    size_t next_handle_idx = 0;

    /// @brief reserves nodes to exactly prog.nodes.size() then emplace-backs
    /// each Node wrapper. The reserve is load-bearing: every Node* held by
    /// MemberState and ScopeState::transition_on_node would dangle on
    /// reallocation, so the vector must never grow past its initial
    /// capacity.
    void populate_nodes(
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls
    ) {
        const size_t n = this->s->prog.nodes.size();
        this->s->nodes.reserve(n);
        this->s->changed_flags.assign(n, 0);
        this->s->self_changed_flags.assign(n, 0);
        this->nodes_by_key.reserve(n);
        this->output_by_param.reserve(n);

        for (size_t idx = 0; idx < n; ++idx) {
            const auto &ir_node = this->s->prog.nodes[idx];
            auto &wrapper = this->s->nodes.emplace_back();
            wrapper.key = ir_node.key;
            wrapper.idx = idx;
            const auto impl_it = node_impls.find(ir_node.key);
            if (impl_it != node_impls.end()) wrapper.node = std::move(impl_it->second);
            // Pre-seed outputs in the IR's declared order. Node impls
            // hardcode integer ordinals that match positions in this list;
            // wiring passes below use get_or_create_output, which finds
            // these pre-seeded entries by name via the side map.
            wrapper.outputs.reserve(ir_node.outputs.size());
            std::unordered_map<std::string, size_t> params;
            params.reserve(ir_node.outputs.size());
            for (const auto &p: ir_node.outputs) {
                params[p.name] = wrapper.outputs.size();
                wrapper.outputs.push_back(Scheduler::OutputResolved{});
            }
            this->nodes_by_key[ir_node.key] = &wrapper;
            this->output_by_param[&wrapper] = std::move(params);
        }
    }

    /// @brief single pass over edges (O(N+E)). Resolves each edge's source
    /// param to its local output ordinal and the target node to its dense
    /// flag-slice index so mark_changed needs zero hash lookups per edge.
    void wire_edges() {
        for (const auto &edge: this->s->prog.edges) {
            Scheduler::Node *src = this->lookup_node(edge.source.node);
            if (src == nullptr) continue;
            Scheduler::Node *tgt = this->lookup_node(edge.target.node);
            if (tgt == nullptr) continue;
            const size_t out_idx = this->get_or_create_output(*src, edge.source.param);
            src->outputs[out_idx].edges.push_back(
                Scheduler::OutEdge{tgt->idx, edge.kind == ir::EdgeKind::Conditional}
            );
        }
    }

    /// @brief recursively constructs a ScopeState mirror of the IR scope
    /// tree. The returned state is inert — activation runs only after
    /// build completes, via Scheduler::activate_scope on the root.
    std::unique_ptr<Scheduler::ScopeState> build_scope_state(const ir::Scope *sc) {
        auto state = std::make_unique<Scheduler::ScopeState>();
        state->ir = sc;

        const auto append_member = [&](const ir::Member &m) {
            Scheduler::MemberState ms;
            ms.key = m.key();
            if (m.node_key.has_value()) {
                ms.node = this->lookup_node(*m.node_key);
            } else if (m.scope) {
                ms.scope = this->build_scope_state(&*m.scope);
            }
            state->members.push_back(std::move(ms));
        };

        if (sc->mode == ir::ScopeMode::Parallel) {
            for (const auto &stratum: sc->strata)
                for (const auto &m: stratum)
                    append_member(m);
        } else if (sc->mode == ir::ScopeMode::Sequential) {
            for (const auto &m: sc->steps)
                append_member(m);
        }

        state->member_by_key.reserve(state->members.size());
        for (size_t i = 0; i < state->members.size(); ++i)
            state->member_by_key[state->members[i].key] = i;
        return state;
    }

    /// @brief recursively wires per-output activation entries on source
    /// Nodes (so mark_changed needs zero scheduler-wide hash lookups) and
    /// resolves the per-sequential-scope transition tables.
    void register_scope(Scheduler::ScopeState &state) {
        if (state.ir->liveness == ir::Liveness::Gated &&
            state.ir->activation.has_value()) {
            if (Scheduler::Node *src = this->lookup_node(state.ir->activation->node);
                src != nullptr) {
                const size_t out_idx = this->get_or_create_output(
                    *src,
                    state.ir->activation->param
                );
                src->outputs[out_idx].activates.push_back(&state);
            }
        }
        if (state.ir->mode == ir::ScopeMode::Sequential)
            this->resolve_transitions(state);
        for (auto &m: state.members)
            if (m.scope) this->register_scope(*m.scope);
    }

    /// @brief populates the parallel transition slices on state from the
    /// IR's transition list. Each transition's on-handle is resolved to a
    /// (Node*, output ordinal) pair so the hot-path truthy check is a
    /// direct virtual call. The handle is also assigned a dense
    /// marked_flags index for the fresh-mark check, and the source node
    /// is mapped back to its owning member so transitions originating in
    /// inactive siblings can be skipped. Transitions whose on-node is
    /// unknown get nullptr/NO_INDEX throughout and are silently skipped
    /// at evaluation time.
    void resolve_transitions(Scheduler::ScopeState &state) {
        std::unordered_map<Scheduler::Node *, size_t> node_to_member;
        for (size_t i = 0; i < state.members.size(); ++i)
            collect_member_nodes(state.members[i], i, node_to_member);

        const auto &transitions = state.ir->transitions;
        state.transition_owner.assign(transitions.size(), NO_INDEX);
        state.transition_on_idx.assign(transitions.size(), NO_INDEX);
        state.transition_on_node.assign(transitions.size(), nullptr);
        state.transition_on_output_idx.assign(transitions.size(), NO_INDEX);
        for (size_t i = 0; i < transitions.size(); ++i) {
            Scheduler::Node *on = this->lookup_node(transitions[i].on.node);
            if (on == nullptr) continue;
            const size_t out_idx = this->get_or_create_output(
                *on,
                transitions[i].on.param
            );
            auto &out = on->outputs[out_idx];
            if (out.mark_handle_idx == NO_INDEX)
                out.mark_handle_idx = this->next_handle_idx++;
            state.transition_on_node[i] = on;
            state.transition_on_output_idx[i] = out_idx;
            state.transition_on_idx[i] = out.mark_handle_idx;
            if (const auto it = node_to_member.find(on); it != node_to_member.end())
                state.transition_owner[i] = it->second;
        }
        // Pre-filter transitions per step so evaluate_transitions can
        // iterate a short list instead of scanning all N transitions per
        // cascade step. Each step's list contains transitions owned by
        // that step plus every external transition (owner == NO_INDEX),
        // in source order.
        state.transitions_for_step.assign(state.members.size(), {});
        for (size_t m = 0; m < state.members.size(); ++m) {
            auto &list = state.transitions_for_step[m];
            for (size_t i = 0; i < transitions.size(); ++i) {
                const size_t owner = state.transition_owner[i];
                if (owner == NO_INDEX || owner == m) list.push_back(i);
            }
        }
    }

    /// @brief returns the node-local output index for the given param,
    /// allocating a fresh entry in Node::outputs if the param is not yet
    /// known. Used by every wiring pass — edges, transition sources, and
    /// activation sources — onto the owning Node.
    size_t get_or_create_output(Scheduler::Node &n, const std::string &param) {
        auto &params = this->output_by_param[&n];
        const auto it = params.find(param);
        if (it != params.end()) return it->second;
        const size_t idx = n.outputs.size();
        n.outputs.push_back(Scheduler::OutputResolved{});
        params[param] = idx;
        return idx;
    }

    /// @brief returns the wrapper for the given node key, or nullptr if the
    /// key is unknown.
    Scheduler::Node *lookup_node(const std::string &key) {
        const auto it = this->nodes_by_key.find(key);
        return it == this->nodes_by_key.end() ? nullptr : it->second;
    }

    /// @brief adds every node owned (directly or transitively) by m to out,
    /// tagged with the member index. A leaf-node member owns one node; a
    /// nested-scope member owns every node reachable through its scope
    /// tree. Unresolved node keys are skipped.
    static void collect_member_nodes(
        Scheduler::MemberState &m,
        const size_t idx,
        std::unordered_map<Scheduler::Node *, size_t> &out
    ) {
        if (m.is_node()) {
            if (m.node) out[m.node] = idx;
            return;
        }
        if (m.scope) collect_scope_nodes(*m.scope, idx, out);
    }

    static void collect_scope_nodes(
        Scheduler::ScopeState &s,
        const size_t idx,
        std::unordered_map<Scheduler::Node *, size_t> &out
    ) {
        for (auto &inner: s.members) {
            if (inner.is_node()) {
                if (inner.node) out[inner.node] = idx;
            } else if (inner.scope)
                collect_scope_nodes(*inner.scope, idx, out);
        }
    }
};

}

inline Scheduler::Scheduler(
    ir::IR prog,
    std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls,
    const x::telem::TimeSpan tolerance,
    errors::Handler error_handler
):
    tolerance(tolerance),
    error_handler(std::move(error_handler)),
    prog(std::move(prog)) {
    this->ctx.mark_changed = std::bind_front(&Scheduler::mark_changed, this);
    this->ctx.mark_self_changed = std::bind_front(&Scheduler::mark_self_changed, this);
    this->ctx.set_deadline = [this](const x::telem::TimeSpan d) {
        if (d < this->min_deadline) this->min_deadline = d;
    };
    this->ctx.report_error = std::bind_front(&Scheduler::report_error, this);
    detail::Builder().build(*this, node_impls);
}

}
