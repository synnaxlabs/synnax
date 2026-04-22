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

/// @brief sentinel value indicating an absent or unresolved index. Used
/// across the scheduler where a node or scope reference may be missing
/// (unresolved IR keys, inactive sequential steps, uninitialized
/// curr_node). Long-lived node and scope references are dense size_t
/// indices into Scheduler::nodes / Scheduler::scopes; the type system
/// does not distinguish them — tests cover the mixups.
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

    /// @brief per-output propagation table built at construction and indexed
    /// by the node-local output ordinal. Hot-path access is pure array
    /// indexing.
    struct OutputResolved {
        /// @brief outgoing dataflow edges from this output.
        std::vector<OutEdge> edges;
        /// @brief marked_flags index for sequential-scope transitions
        /// sourced from this output, or NO_INDEX if none.
        size_t mark_handle_idx = NO_INDEX;
        /// @brief gated scopes whose activation handle is this output
        /// (dense Scheduler::scopes indices); activated by mark_changed
        /// when the source fires.
        std::vector<size_t> activates;
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
    /// An unresolved node key leaves both fields set to NO_INDEX and the
    /// member is skipped at execution.
    struct MemberState {
        /// @brief IR member key, used for `=> name` transition lookups.
        std::string key;
        /// @brief dense index into Scheduler::nodes for leaf-node targets;
        /// NO_INDEX for scope members and for unresolved node keys.
        size_t node = NO_INDEX;
        /// @brief dense index into Scheduler::scopes for nested scopes;
        /// NO_INDEX for leaf-node members.
        size_t scope = NO_INDEX;

        [[nodiscard]] bool is_node() const { return scope == NO_INDEX; }
    };

    /// @brief runtime mirror of an ir::Scope, holding activation bookkeeping
    /// for one scope.
    struct ScopeState {
        /// @brief static IR scope this state mirrors.
        ir::Scope ir;
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
        /// @brief transition_on_node[i] is the Scheduler::nodes index of
        /// transition i's `on`-handle source; NO_INDEX if unresolved.
        std::vector<size_t> transition_on_node;
        /// @brief transition_on_output_idx[i] is the output ordinal on
        /// transition_on_node[i] for transition i's `on`-handle, or
        /// NO_INDEX if unresolved. Paired with transition_on_node[i] so
        /// the truthy check is a direct virtual call.
        std::vector<size_t> transition_on_output_idx;
    };

    /// @brief owns every runtime Node wrapper. Dense: a size_t index
    /// addresses a slot directly. Sized once at construction.
    std::vector<Node> nodes;
    /// @brief owns every runtime ScopeState. scopes[0] is the program's
    /// always-live root scope; all other scopes are nested members reached
    /// from it. Dense: a size_t index addresses a slot directly. Sized
    /// once at construction.
    std::vector<ScopeState> scopes;
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
    /// @brief index into nodes of the node whose next is currently
    /// executing, cached so mark_changed / mark_self_changed callbacks
    /// know whom they came from. NO_INDEX between cycles.
    size_t curr_node = NO_INDEX;
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
    /// re-activates the root scope (scopes[0]). Used by the runtime when
    /// stopping a program so subsequent runs start from a clean state.
    void reset() {
        std::ranges::fill(this->changed_flags, 0);
        std::ranges::fill(this->self_changed_flags, 0);
        std::ranges::fill(this->marked_flags, 0);
        this->curr_node = NO_INDEX;
        for (auto &sc: this->scopes) {
            sc.active = false;
            sc.active_step = NO_INDEX;
        }
        for (auto &n: this->nodes)
            n.node->reset();
        this->activate_scope(this->scopes[0]);
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

        this->walk(this->scopes[0]);

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
        if (state.ir.mode == ir::ScopeMode::Sequential) {
            this->walk_sequential(state);
            return;
        }
        this->walk_parallel(state);
    }

    /// @brief runs every member of a parallel scope in stratum order.
    /// Members are stored stratum-flattened in state.members; state.ir.strata
    /// is read only for the stratum-index argument passed to execute_member.
    void walk_parallel(ScopeState &state) {
        const auto &strata = state.ir.strata;
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
        if (m.scope != NO_INDEX) {
            this->walk(this->scopes[m.scope]);
            return;
        }
        if (m.node == NO_INDEX) return;
        const size_t idx = m.node;
        const bool was_self_changed = this->self_changed_flags[idx] != 0;
        if (was_self_changed) this->self_changed_flags[idx] = 0;
        if (stratum_idx == 0 || this->changed_flags[idx] || was_self_changed) {
            this->curr_node = m.node;
            this->nodes[this->curr_node].node->next(this->ctx);
        }
    }

    /// @brief clears self-changed and calls reset on m's node. No-op if m is
    /// a scope member or an unresolved node-key.
    void reset_leaf_node(MemberState &m) {
        if (m.is_node() && m.node != NO_INDEX) {
            this->self_changed_flags[m.node] = 0;
            this->nodes[m.node].node->reset();
        }
    }

    /// @brief clears self-changed on m's node. No-op if m is a scope member
    /// or an unresolved node-key.
    void clear_leaf_node_self_changed(MemberState &m) {
        if (m.is_node() && m.node != NO_INDEX) this->self_changed_flags[m.node] = 0;
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
        const auto &transitions = state.ir.transitions;
        for (const size_t i: state.transitions_for_step[state.active_step]) {
            const size_t handle_idx = state.transition_on_idx[i];
            if (handle_idx == NO_INDEX || !this->marked_flags[handle_idx]) continue;
            if (!this->nodes[state.transition_on_node[i]].node->is_output_truthy(
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
        if (state.ir.mode == ir::ScopeMode::Sequential) {
            if (!state.members.empty()) this->activate_sequential_step(state, 0);
            return;
        }
        for (auto &m: state.members) {
            if (m.is_node()) {
                this->reset_leaf_node(m);
                continue;
            }
            if (m.scope == NO_INDEX) continue;
            auto &child = this->scopes[m.scope];
            if (child.ir.liveness == ir::Liveness::Gated &&
                !child.ir.activation.has_value())
                this->activate_scope(child);
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
        if (m.scope != NO_INDEX) this->activate_scope(this->scopes[m.scope]);
    }

    /// @brief clears self-changed for the step's node, or marks a nested
    /// scope inactive. Nested-scope state freezes and is overwritten on
    /// the next parent activation.
    void deactivate_step(MemberState &m) {
        if (m.scope != NO_INDEX) {
            this->deactivate_scope(this->scopes[m.scope]);
            return;
        }
        this->clear_leaf_node_self_changed(m);
    }

    /// @brief marks a scope inactive and clears self-changed on its direct
    /// leaf-node members. Does not recurse — nested scope state freezes
    /// until the next activation overwrites it.
    void deactivate_scope(ScopeState &state) {
        if (state.ir.mode == ir::ScopeMode::Sequential) state.active_step = NO_INDEX;
        for (auto &m: state.members)
            this->clear_leaf_node_self_changed(m);
        state.active = false;
    }

    void report_error(const x::errors::Error &e) const {
        LOG(ERROR) << "[arc.scheduler] node encountered error: " << e;
        this->error_handler(e);
    }

    /// @brief propagates the current node's output to downstream nodes,
    /// records a fresh transition mark when truthy, and fires any gated
    /// scope activations attached to this output. Conditional edges
    /// propagate only when the source is truthy. Only callable from
    /// inside a node's next() — curr_node is set by execute_member
    /// immediately before the call.
    void mark_changed(const size_t output_idx) {
        DCHECK(this->curr_node != NO_INDEX);
        const auto &n = this->nodes[this->curr_node];
        if (output_idx >= n.outputs.size()) return;
        const auto &out = n.outputs[output_idx];
        const bool truthy = n.node->is_output_truthy(output_idx);
        if (truthy && out.mark_handle_idx != NO_INDEX)
            this->marked_flags[out.mark_handle_idx] = 1;
        for (const auto &edge: out.edges)
            if (!edge.conditional || truthy) this->changed_flags[edge.target_idx] = 1;
        for (const size_t scope_idx: out.activates) {
            auto &scope = this->scopes[scope_idx];
            if (!scope.active) this->activate_scope(scope);
        }
    }

    void mark_self_changed() {
        DCHECK(this->curr_node != NO_INDEX);
        this->self_changed_flags[this->curr_node] = 1;
    }
};

namespace detail {

/// @brief assembles a Scheduler from a compiled IR. Owns every piece of
/// state needed only during wiring — the key → node-index map, the
/// per-node param-to-ordinal maps, and the running marked_flags handle
/// counter — so the resulting Scheduler holds only what the hot path
/// consults. Builder state is destroyed when build returns; the
/// Scheduler's long-lived references into its own vectors are dense
/// size_t indices, which stay valid regardless of vector reallocation.
class Builder {
public:
    void build(
        Scheduler &s,
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls
    ) {
        this->s = &s;
        this->populate_nodes(node_impls);
        this->wire_edges();
        s.scopes.reserve(count_scopes(s.prog.root));
        const size_t root_idx = this->build_scope_state(s.prog.root);
        this->register_scope(root_idx);
        s.marked_flags.assign(this->next_handle_idx, 0);
        s.activate_scope(s.scopes[root_idx]);
    }

private:
    /// @brief Scheduler being assembled; held as a raw pointer for the
    /// duration of build.
    Scheduler *s = nullptr;
    /// @brief node key to its dense index into Scheduler::nodes.
    std::unordered_map<std::string, size_t> nodes_by_key;
    /// @brief per-node "output name → ordinal" map used to find or
    /// allocate matching entries in Node::outputs while wiring. Indexed
    /// by Node::idx (dense [0, N)); kept off Scheduler::Node so the
    /// long-lived wrapper stays lean.
    std::vector<std::unordered_map<std::string, size_t>> output_by_param;
    /// @brief next index to assign to a freshly-discovered transition-source
    /// handle. After build, this is the final length of marked_flags.
    size_t next_handle_idx = 0;

    /// @brief allocates the flat nodes vector and builds the key → node
    /// index map. Node::idx equals the slot in Scheduler::nodes, so dense
    /// auxiliary structures (output_by_param, hot-path index arrays) can
    /// be indexed by it directly.
    void populate_nodes(
        std::unordered_map<std::string, std::unique_ptr<node::Node>> &node_impls
    ) {
        const size_t n = this->s->prog.nodes.size();
        this->s->nodes.resize(n);
        this->s->changed_flags.assign(n, 0);
        this->s->self_changed_flags.assign(n, 0);
        this->nodes_by_key.reserve(n);
        this->output_by_param.resize(n);

        for (size_t idx = 0; idx < n; ++idx) {
            const auto &ir_node = this->s->prog.nodes[idx];
            auto &wrapper = this->s->nodes[idx];
            wrapper.key = ir_node.key;
            wrapper.idx = idx;
            const auto impl_it = node_impls.find(ir_node.key);
            if (impl_it != node_impls.end()) wrapper.node = std::move(impl_it->second);
            // Pre-seed outputs in the IR's declared order. Node impls
            // hardcode integer ordinals that match positions in this list;
            // wiring passes below use get_or_create_output, which finds
            // these pre-seeded entries by name via the side map.
            wrapper.outputs.reserve(ir_node.outputs.size());
            auto &params = this->output_by_param[idx];
            params.reserve(ir_node.outputs.size());
            for (const auto &p: ir_node.outputs) {
                params[p.name] = wrapper.outputs.size();
                wrapper.outputs.push_back(Scheduler::OutputResolved{});
            }
            this->nodes_by_key[ir_node.key] = idx;
        }
    }

    /// @brief single pass over edges (O(N+E)). Resolves each edge's source
    /// param to its local output ordinal and the target node to its dense
    /// flag-slice index so mark_changed needs zero hash lookups per edge.
    void wire_edges() {
        for (const auto &edge: this->s->prog.edges) {
            const size_t src = this->lookup_node(edge.source.node);
            if (src == NO_INDEX) continue;
            const size_t tgt = this->lookup_node(edge.target.node);
            if (tgt == NO_INDEX) continue;
            auto &src_node = this->s->nodes[src];
            const size_t out_idx = this->get_or_create_output(
                src_node,
                edge.source.param
            );
            src_node.outputs[out_idx].edges.push_back(
                Scheduler::OutEdge{tgt, edge.kind == ir::EdgeKind::Conditional}
            );
        }
    }

    /// @brief counts every scope reachable from sc (including sc itself).
    /// Used to size Scheduler::scopes exactly once so the recursive build
    /// doesn't need to reallocate.
    static size_t count_scopes(const ir::Scope &sc) {
        size_t n = 1;
        const auto count_member = [&](const ir::Member &m) {
            if (m.scope) n += count_scopes(*m.scope);
        };
        if (sc.mode == ir::ScopeMode::Parallel) {
            for (const auto &stratum: sc.strata)
                for (const auto &m: stratum)
                    count_member(m);
        } else if (sc.mode == ir::ScopeMode::Sequential) {
            for (const auto &m: sc.steps)
                count_member(m);
        }
        return n;
    }

    /// @brief recursively constructs a ScopeState mirror of the IR scope
    /// tree into Scheduler::scopes and returns its index. The returned
    /// state is inert — activation runs only after build completes, via
    /// Scheduler::activate_scope on the root.
    size_t build_scope_state(const ir::Scope &sc) {
        const size_t this_idx{this->s->scopes.size()};
        this->s->scopes.emplace_back();
        this->s->scopes[this_idx].ir = sc;

        const auto append_member = [&](const ir::Member &m) {
            Scheduler::MemberState ms;
            ms.key = m.key();
            if (m.node_key.has_value()) {
                ms.node = this->lookup_node(*m.node_key);
            } else if (m.scope) {
                ms.scope = this->build_scope_state(*m.scope);
            }
            // Re-index rather than holding a reference across the recursion,
            // which may push onto scopes.
            this->s->scopes[this_idx].members.push_back(std::move(ms));
        };

        if (sc.mode == ir::ScopeMode::Parallel) {
            for (const auto &stratum: sc.strata)
                for (const auto &m: stratum)
                    append_member(m);
        } else if (sc.mode == ir::ScopeMode::Sequential) {
            for (const auto &m: sc.steps)
                append_member(m);
        }

        auto &state = this->s->scopes[this_idx];
        state.member_by_key.reserve(state.members.size());
        for (size_t i = 0; i < state.members.size(); ++i)
            state.member_by_key[state.members[i].key] = i;
        return this_idx;
    }

    /// @brief recursively wires per-output activation entries on source
    /// Nodes (so mark_changed needs zero scheduler-wide hash lookups) and
    /// resolves the per-sequential-scope transition tables. Takes a
    /// size_t so it can record it into OutputResolved::activates.
    void register_scope(const size_t idx) {
        auto &state = this->s->scopes[idx];
        if (state.ir.liveness == ir::Liveness::Gated &&
            state.ir.activation.has_value()) {
            if (const size_t src = this->lookup_node(state.ir.activation->node);
                src != NO_INDEX) {
                auto &src_node = this->s->nodes[src];
                const size_t out_idx = this->get_or_create_output(
                    src_node,
                    state.ir.activation->param
                );
                src_node.outputs[out_idx].activates.push_back(idx);
            }
        }
        if (state.ir.mode == ir::ScopeMode::Sequential)
            this->resolve_transitions(state);
        for (auto &m: state.members)
            if (m.scope != NO_INDEX) this->register_scope(m.scope);
    }

    /// @brief populates the parallel transition slices on state from the
    /// IR's transition list. Each transition's on-handle is resolved to a
    /// (node-index, output ordinal) pair so the hot-path truthy check is a
    /// direct virtual call. The handle is also assigned a dense
    /// marked_flags index for the fresh-mark check, and the source node
    /// is mapped back to its owning member so transitions originating in
    /// inactive siblings can be skipped. Transitions whose on-node is
    /// unknown get NO_INDEX/NO_INDEX throughout and are silently skipped
    /// at evaluation time.
    void resolve_transitions(Scheduler::ScopeState &state) {
        // Key: node index (dense [0, N)); value: owning member index.
        std::unordered_map<size_t, size_t> node_to_member;
        for (size_t i = 0; i < state.members.size(); ++i)
            collect_member_nodes(state.members[i], i, node_to_member);

        const auto &transitions = state.ir.transitions;
        state.transition_owner.assign(transitions.size(), NO_INDEX);
        state.transition_on_idx.assign(transitions.size(), NO_INDEX);
        state.transition_on_node.assign(transitions.size(), NO_INDEX);
        state.transition_on_output_idx.assign(transitions.size(), NO_INDEX);
        for (size_t i = 0; i < transitions.size(); ++i) {
            const size_t on = this->lookup_node(transitions[i].on.node);
            if (on == NO_INDEX) continue;
            auto &on_node = this->s->nodes[on];
            const size_t out_idx = this->get_or_create_output(
                on_node,
                transitions[i].on.param
            );
            auto &out = on_node.outputs[out_idx];
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
        auto &params = this->output_by_param[n.idx];
        const auto it = params.find(param);
        if (it != params.end()) return it->second;
        const size_t idx = n.outputs.size();
        n.outputs.push_back(Scheduler::OutputResolved{});
        params[param] = idx;
        return idx;
    }

    /// @brief returns the Scheduler::nodes index for the given node key,
    /// or NO_INDEX when the key is unknown.
    size_t lookup_node(const std::string &key) {
        const auto it = this->nodes_by_key.find(key);
        return it == this->nodes_by_key.end() ? NO_INDEX : it->second;
    }

    /// @brief adds every node owned (directly or transitively) by m to out,
    /// tagged with the member index. A leaf-node member owns one node; a
    /// nested-scope member owns every node reachable through its scope
    /// tree. Unresolved node keys are skipped. Keyed by node index.
    void collect_member_nodes(
        Scheduler::MemberState &m,
        const size_t idx,
        std::unordered_map<size_t, size_t> &out
    ) const {
        if (m.is_node()) {
            if (m.node != NO_INDEX) out[m.node] = idx;
            return;
        }
        if (m.scope != NO_INDEX)
            collect_scope_nodes(this->s->scopes[m.scope], idx, out);
    }

    void collect_scope_nodes(
        Scheduler::ScopeState &sc,
        const size_t idx,
        std::unordered_map<size_t, size_t> &out
    ) const {
        for (auto &inner: sc.members) {
            if (inner.is_node()) {
                if (inner.node != NO_INDEX) out[inner.node] = idx;
            } else if (inner.scope != NO_INDEX)
                collect_scope_nodes(this->s->scopes[inner.scope], idx, out);
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
