// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in
// the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business
// Source License, use of this software will be governed by the Apache License,
// Version 2.0, included in the file licenses/APL.txt.

#include "arc/cpp/runtime/scheduler.h"

namespace arc {

Scheduler::Scheduler(State *state) : state_(*state) {}

xerrors::Error Scheduler::register_node(std::string node_id,
                                       std::unique_ptr<Node> node,
                                       size_t stratum) {
    // Check for duplicate registration
    if (nodes_.find(node_id) != nodes_.end()) {
        return xerrors::Error("arc.scheduler.duplicate_node");
    }

    // Expand strata if needed
    if (stratum >= strata_.size()) {
        strata_.resize(stratum + 1);
    }

    // Add to stratum
    strata_[stratum].push_back(node_id);

    // Store node
    nodes_[node_id] = std::move(node);

    // Store stratum mapping
    node_stratum_[node_id] = stratum;

    return xerrors::NIL;
}

xerrors::Error Scheduler::next() {
    // 1. Process input queue from I/O thread
    state_.process_input_queue();

    // 2. Create NodeContext with callbacks
    NodeContext ctx;
    ctx.mark_changed = [this](const std::string &output_param) {
        this->mark_output_changed(this->current_executing_node_, output_param);
    };
    ctx.report_error = [](const xerrors::Error &err) {
        // TODO: Implement error reporting/logging
    };

    // 3. Execute stratum 0 (always execute - sources/inputs)
    if (!strata_.empty()) {
        for (const auto &node_id : strata_[0]) {
            auto it = nodes_.find(node_id);
            if (it != nodes_.end()) {
                current_executing_node_ = node_id;
                if (auto err = it->second->execute(ctx)) {
                    return err;
                }
            }
        }
    }

    // 4. Execute higher strata (only if changed)
    for (size_t i = 1; i < strata_.size(); i++) {
        for (const auto &node_id : strata_[i]) {
            if (changed_.find(node_id) != changed_.end()) {
                auto it = nodes_.find(node_id);
                if (it != nodes_.end()) {
                    current_executing_node_ = node_id;
                    if (auto err = it->second->execute(ctx)) {
                        return err;
                    }
                }
            }
        }
    }

    // 5. Clear changed set for next cycle
    changed_.clear();

    return xerrors::NIL;
}

void Scheduler::mark_changed(const std::string &node_id) {
    changed_.insert(node_id);
}

void Scheduler::mark_downstream_changed(const std::string &node_id) {
    // Find this node's stratum
    auto stratum_it = node_stratum_.find(node_id);
    if (stratum_it == node_stratum_.end()) return;

    const size_t source_stratum = stratum_it->second;

    // Mark all nodes in higher strata as changed
    // (Simplified: marks ALL downstream nodes. In full impl, would use dependency graph)
    for (size_t i = source_stratum + 1; i < strata_.size(); i++) {
        for (const auto &downstream_id : strata_[i]) {
            changed_.insert(downstream_id);
        }
    }
}

size_t Scheduler::get_stratum(const std::string &node_id) const {
    auto it = node_stratum_.find(node_id);
    if (it == node_stratum_.end()) return 0;
    return it->second;
}

bool Scheduler::has_node(const std::string &node_id) const {
    return nodes_.find(node_id) != nodes_.end();
}

void Scheduler::register_outgoing_edge(const std::string &source_node,
                                       const std::string &source_param,
                                       const std::string &target_node) {
    outgoing_edges_[source_node].push_back(
        OutgoingEdge{source_param, target_node});
}

void Scheduler::mark_output_changed(const std::string &node_id,
                                    const std::string &output_param) {
    auto it = outgoing_edges_.find(node_id);
    if (it == outgoing_edges_.end()) return;

    // Mark only downstream nodes that depend on this specific output
    for (const auto &edge : it->second) {
        if (edge.source_param == output_param) {
            changed_.insert(edge.target_node);
        }
    }
}

}  // namespace arc
