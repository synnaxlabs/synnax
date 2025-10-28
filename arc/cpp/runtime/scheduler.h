// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <string>
#include <unordered_map>
#include <unordered_set>
#include <vector>

#include "arc/cpp/runtime/state.h"
#include "x/cpp/xerrors/errors.h"

namespace arc {

/// @brief Context passed to nodes during execution.
///
/// Provides callback mechanisms for nodes to mark their outputs as changed
/// and report errors. This matches the Go runtime's node.Context pattern.
struct NodeContext {
    /// @brief Callback to mark a specific output parameter as changed.
    ///
    /// When a node produces new output, it calls this callback with the
    /// output parameter name. The scheduler then marks downstream nodes
    /// that depend on this output for re-execution.
    ///
    /// @param output_param Name of the output parameter that changed.
    std::function<void(const std::string &output_param)> mark_changed;

    /// @brief Callback to report errors during node execution.
    /// @param err Error to report.
    std::function<void(const xerrors::Error &)> report_error;
};

/// @brief Abstract node interface for executable units in the scheduler.
///
/// Nodes represent compiled Arc stages (WASM functions, operators, etc.)
/// that can be executed by the scheduler.
class Node {
public:
    virtual ~Node() = default;

    /// @brief Execute this node.
    /// @param ctx Node context with callbacks for change tracking.
    /// @return Error status (NIL on success).
    /// @note Must be RT-safe if used in RT thread.
    virtual xerrors::Error execute(NodeContext &ctx) = 0;

    /// @brief Get node identifier.
    /// @return Node ID string.
    virtual std::string id() const = 0;
};

/// @brief Outgoing edge from a node output to downstream node.
struct OutgoingEdge {
    std::string source_param;  ///< Source node's output parameter name
    std::string target_node;   ///< Target (downstream) node ID
};

/// @brief Stratified scheduler for reactive Arc execution.
///
/// Implements Arc's stratified execution model:
/// - Stratum 0: Always executes (source nodes, channel readers)
/// - Stratum N: Executes only if marked as "changed" by upstream nodes
///
/// The scheduler maintains a pre-computed topological ordering (stratification)
/// and tracks which nodes need re-execution via a "changed" set.
class Scheduler {
    /// Stratified execution order (pre-computed)
    /// strata_[i] contains node IDs for stratum i
    std::vector<std::vector<std::string>> strata_;

    /// Node registry (node_id -> node instance)
    std::unordered_map<std::string, std::unique_ptr<Node>> nodes_;

    /// Stratum lookup (node_id -> stratum index)
    std::unordered_map<std::string, size_t> node_stratum_;

    /// Changed node tracking for reactive execution
    std::unordered_set<std::string> changed_;

    /// State reference (non-owning)
    State& state_;

    /// Outgoing edges per node (source_node_id -> list of outgoing edges)
    /// Used for per-output change propagation (matches Go runtime behavior)
    std::unordered_map<std::string, std::vector<OutgoingEdge>> outgoing_edges_;

    /// Currently executing node (used for NodeContext callbacks)
    std::string current_executing_node_;

public:
    /// @brief Construct scheduler with state reference.
    /// @param state Pointer to global state (not owned).
    explicit Scheduler(State *state);

    /// @brief Register a node at a specific stratum.
    /// @param node_id Node identifier (must be unique).
    /// @param node Node instance (ownership transferred).
    /// @param stratum Stratum index (0 = always execute, >0 = reactive).
    /// @return Error status (NIL on success).
    /// @note Must be called during initialization, not in RT loop.
    xerrors::Error register_node(std::string node_id,
                                 std::unique_ptr<Node> node,
                                 size_t stratum);

    /// @brief Execute one scheduler cycle (RT-safe).
    ///
    /// Execution order:
    /// 1. Process input queue (update channel data from I/O thread)
    /// 2. Execute stratum 0 (always)
    /// 3. Execute higher strata (only if changed)
    /// 4. Clear changed set for next cycle
    ///
    /// @return Error status (NIL on success).
    /// @note RT-safe: no allocations, bounded execution time.
    xerrors::Error next();

    /// @brief Mark a node as changed (triggers downstream re-execution).
    /// @param node_id Node identifier.
    /// @note Called by nodes when they produce new outputs.
    void mark_changed(const std::string &node_id);

    /// @brief Mark downstream nodes as changed.
    /// @param node_id Source node that produced output.
    /// @note Marks all nodes in higher strata that depend on this node.
    void mark_downstream_changed(const std::string &node_id);

    /// @brief Get stratum for a node.
    /// @param node_id Node identifier.
    /// @return Stratum index, or 0 if not found.
    size_t get_stratum(const std::string &node_id) const;

    /// @brief Get number of strata.
    /// @return Number of strata in the scheduler.
    size_t num_strata() const { return strata_.size(); }

    /// @brief Get number of registered nodes.
    /// @return Total number of nodes.
    size_t num_nodes() const { return nodes_.size(); }

    /// @brief Check if a node is registered.
    /// @param node_id Node identifier.
    /// @return true if node exists.
    bool has_node(const std::string &node_id) const;

    /// @brief Register an outgoing edge from a node's output to downstream node.
    /// @param source_node Source node identifier.
    /// @param source_param Source node's output parameter name.
    /// @param target_node Target (downstream) node identifier.
    /// @note Must be called during initialization, not in RT loop.
    void register_outgoing_edge(const std::string &source_node,
                                const std::string &source_param,
                                const std::string &target_node);

    /// @brief Mark downstream nodes that depend on a specific output parameter.
    ///
    /// Called by nodes via NodeContext.mark_changed callback when they
    /// produce new output. Only marks downstream nodes that have edges
    /// from the specified output parameter.
    ///
    /// @param node_id Source node identifier.
    /// @param output_param Output parameter name that changed.
    /// @note RT-safe: bounded lookup in outgoing_edges_ map.
    void mark_output_changed(const std::string &node_id,
                             const std::string &output_param);
};

}  // namespace arc
