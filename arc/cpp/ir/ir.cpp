// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <sstream>
#include <stdexcept>

#include "arc/cpp/ir/ir.h"

namespace arc::ir {

std::string Handle::to_string() const {
    return this->node + "." + this->param;
}

std::ostream &operator<<(std::ostream &os, const Handle &h) {
    return os << h.to_string();
}

std::string Edge::to_string() const {
    std::ostringstream ss;
    ss << this->source.to_string();
    ss << (this->kind == EdgeKind::Conditional ? " => " : " -> ");
    ss << this->target.to_string();
    if (this->kind == EdgeKind::Continuous)
        ss << " (continuous)";
    else if (this->kind == EdgeKind::Conditional)
        ss << " (conditional)";
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Edge &e) {
    return os << e.to_string();
}

namespace {

/// @brief returns the short label used in tree output for a scope's mode.
const char *scope_mode_label(const ScopeMode m) {
    switch (m) {
        case ScopeMode::Parallel:
            return "parallel";
        case ScopeMode::Sequential:
            return "sequential";
        default:
            return "unspecified";
    }
}

/// @brief returns the short label used in tree output for a scope's liveness.
const char *liveness_label(const Liveness l) {
    switch (l) {
        case Liveness::Always:
            return "always";
        case Liveness::Gated:
            return "gated";
        default:
            return "unspecified";
    }
}

}

std::string Transition::to_string() const {
    std::ostringstream ss;
    ss << "on " << this->on.node << "/" << this->on.param << " ";
    if (this->target.member_key.has_value())
        ss << "=> " << *this->target.member_key;
    else if (this->target.exit.has_value() && *this->target.exit)
        ss << "=> exit";
    else
        ss << "=> ?";
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Transition &t) {
    return os << t.to_string();
}

std::string Member::to_string() const {
    return this->to_string_with_prefix("");
}

std::string Member::to_string_with_prefix(const std::string &prefix) const {
    if (this->node_ref.has_value()) {
        std::ostringstream ss;
        if (!this->key.empty() && this->key != this->node_ref->key)
            ss << this->key << " -> " << this->node_ref->key;
        else
            ss << this->node_ref->key;
        return ss.str();
    }
    if (this->scope) return this->scope->to_string_with_prefix(prefix);
    return "(empty member)";
}

std::ostream &operator<<(std::ostream &os, const Member &m) {
    return os << m.to_string();
}

std::string Phase::to_string() const {
    return this->to_string_with_prefix("");
}

std::string Phase::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    for (size_t i = 0; i < this->members.size(); ++i) {
        const bool last = (i == this->members.size() - 1);
        ss << prefix << tree_prefix(last);
        ss << this->members[i].to_string_with_prefix(prefix + tree_indent(last));
        if (!last) ss << "\n";
    }
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Phase &p) {
    return os << p.to_string();
}

std::string Scope::to_string() const {
    return this->to_string_with_prefix("");
}

std::string Scope::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    const std::string label = this->key.empty() ? std::string("(scope)") : this->key;
    ss << label << " [" << scope_mode_label(this->mode) << ", "
       << liveness_label(this->liveness) << "]";

    const bool is_parallel = this->mode == ScopeMode::Parallel;
    const bool has_phases = is_parallel && !this->phases.empty();
    const bool has_members = !is_parallel && !this->members.empty();
    const bool has_transitions = !this->transitions.empty();

    if (has_phases) {
        for (size_t i = 0; i < this->phases.size(); ++i) {
            const bool last = (i == this->phases.size() - 1) && !has_transitions;
            ss << "\n" << prefix << tree_prefix(last) << "phase " << i;
            if (!this->phases[i].members.empty()) {
                ss << "\n";
                ss << this->phases[i].to_string_with_prefix(prefix + tree_indent(last));
            }
        }
    }
    if (has_members) {
        for (size_t i = 0; i < this->members.size(); ++i) {
            const bool last = (i == this->members.size() - 1) && !has_transitions;
            ss << "\n" << prefix << tree_prefix(last);
            ss << this->members[i].to_string_with_prefix(prefix + tree_indent(last));
        }
    }
    if (has_transitions) {
        for (size_t i = 0; i < this->transitions.size(); ++i) {
            const bool last = (i == this->transitions.size() - 1);
            ss << "\n" << prefix << tree_prefix(last);
            ss << this->transitions[i].to_string();
        }
    }
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Scope &s) {
    return os << s.to_string();
}

namespace {
void write_params_section(
    std::ostringstream &ss,
    const std::string &prefix,
    const std::string &label,
    const ::arc::types::Params &params,
    bool last
) {
    ss << "\n" << prefix << tree_prefix(last) << label << ": ";
    ss << format_params(params);
}

void write_channels_section(
    std::ostringstream &ss,
    const std::string &prefix,
    const ::arc::types::Channels &channels,
    bool last
) {
    ss << "\n" << prefix << tree_prefix(last) << "channels: ";
    ss << format_channels(channels);
}
}

std::string Node::to_string() const {
    return this->to_string_with_prefix("");
}

std::string Node::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    ss << this->key << " (type: " << this->type << ")";
    bool has_channels = !this->channels.read.empty() || !this->channels.write.empty();
    bool has_config = !this->config.empty();
    bool has_inputs = !this->inputs.empty();
    bool has_outputs = !this->outputs.empty();
    if (has_channels)
        write_channels_section(
            ss,
            prefix,
            this->channels,
            !has_config && !has_inputs && !has_outputs
        );
    if (has_config)
        write_params_section(
            ss,
            prefix,
            "config",
            this->config,
            !has_inputs && !has_outputs
        );
    if (has_inputs)
        write_params_section(ss, prefix, "inputs", this->inputs, !has_outputs);
    if (has_outputs) write_params_section(ss, prefix, "outputs", this->outputs, true);
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Node &n) {
    return os << n.to_string();
}

std::string Function::to_string() const {
    return this->to_string_with_prefix("");
}

std::string Function::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    ss << this->key;
    bool has_channels = !this->channels.read.empty() || !this->channels.write.empty();
    bool has_config = !this->config.empty();
    bool has_inputs = !this->inputs.empty();
    bool has_outputs = !this->outputs.empty();
    if (has_channels)
        write_channels_section(
            ss,
            prefix,
            this->channels,
            !has_config && !has_inputs && !has_outputs
        );
    if (has_config)
        write_params_section(
            ss,
            prefix,
            "config",
            this->config,
            !has_inputs && !has_outputs
        );
    if (has_inputs)
        write_params_section(ss, prefix, "inputs", this->inputs, !has_outputs);
    if (has_outputs) write_params_section(ss, prefix, "outputs", this->outputs, true);
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Function &f) {
    return os << f.to_string();
}

const Node &IR::node(const std::string &key) const {
    for (const auto &n: this->nodes)
        if (n.key == key) return n;
    throw std::runtime_error("node not found: " + key);
}

const Function &IR::function(const std::string &key) const {
    for (const auto &f: this->functions)
        if (f.key == key) return f;
    throw std::runtime_error("function not found: " + key);
}

std::optional<Edge> IR::edge_to(const Handle &target) const {
    for (const auto &e: this->edges)
        if (e.target == target) return e;
    return std::nullopt;
}

std::unordered_map<std::string, std::vector<Edge>>
IR::edges_from(const std::string &node_key) const {
    std::unordered_map<std::string, std::vector<Edge>> result;
    for (const auto &e: this->edges)
        if (e.source.node == node_key) result[e.source.param].push_back(e);
    return result;
}

std::vector<Edge> IR::edges_to(const std::string &node_key) const {
    std::vector<Edge> result;
    for (const auto &e: this->edges)
        if (e.target.node == node_key) result.push_back(e);
    return result;
}

namespace {

/// @brief reports whether a scope carries any non-zero state. Used to decide
/// whether the Root section should appear in an IR's tree output.
bool scope_is_zero(const Scope &s) {
    return s.key.empty() && s.mode == ScopeMode::Unspecified &&
           s.liveness == Liveness::Unspecified && !s.activation.has_value() &&
           s.phases.empty() && s.members.empty() && s.transitions.empty();
}

}

std::string IR::to_string() const {
    return this->to_string_with_prefix("");
}

std::string IR::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    ss << "IR";

    const bool has_functions = !this->functions.empty();
    const bool has_nodes = !this->nodes.empty();
    const bool has_edges = !this->edges.empty();
    const bool has_root = !scope_is_zero(this->root);

    if (has_functions) {
        bool last = !has_nodes && !has_edges && !has_root;
        ss << "\n" << prefix << tree_prefix(last) << "Functions";
        std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < this->functions.size(); ++i) {
            bool fn_last = (i == this->functions.size() - 1);
            ss << "\n" << child_prefix << tree_prefix(fn_last);
            ss << this->functions[i].to_string_with_prefix(
                child_prefix + tree_indent(fn_last)
            );
        }
    }

    if (has_nodes) {
        bool last = !has_edges && !has_root;
        ss << "\n" << prefix << tree_prefix(last) << "Nodes";
        std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < this->nodes.size(); ++i) {
            bool n_last = (i == this->nodes.size() - 1);
            ss << "\n" << child_prefix << tree_prefix(n_last);
            ss << this->nodes[i].to_string_with_prefix(
                child_prefix + tree_indent(n_last)
            );
        }
    }

    if (has_edges) {
        bool last = !has_root;
        ss << "\n" << prefix << tree_prefix(last) << "Edges";
        std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < this->edges.size(); ++i) {
            bool e_last = (i == this->edges.size() - 1);
            ss << "\n" << child_prefix << tree_prefix(e_last);
            ss << this->edges[i].to_string();
        }
    }

    if (has_root) {
        ss << "\n" << prefix << tree_prefix(true) << "Root ";
        ss << this->root.to_string_with_prefix(prefix + tree_indent(true));
    }

    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const IR &ir) {
    return os << ir.to_string();
}
}
