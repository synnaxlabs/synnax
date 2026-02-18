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
    ss << (this->kind == EdgeKind::OneShot ? " => " : " -> ");
    ss << this->target.to_string();
    if (this->kind == EdgeKind::Continuous)
        ss << " (continuous)";
    else if (this->kind == EdgeKind::OneShot)
        ss << " (oneshot)";
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Edge &e) {
    return os << e.to_string();
}

std::string Stage::to_string() const {
    return this->to_string_with_prefix("");
}

std::string Stage::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    ss << this->key << ": [";
    for (size_t i = 0; i < this->nodes.size(); ++i) {
        if (i > 0) ss << ", ";
        ss << this->nodes[i];
    }
    ss << "]";
    if (!this->strata.empty()) {
        ss << "\n" << this->strata.to_string_with_prefix(prefix);
    }
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Stage &s) {
    return os << s.to_string();
}

std::string Strata::to_string() const {
    return this->to_string_with_prefix("");
}

std::string Strata::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    for (size_t i = 0; i < this->size(); ++i) {
        bool last = (i == this->size() - 1);
        ss << prefix << tree_prefix(last) << "[" << i << "]: ";
        const auto &stratum = (*this)[i];
        for (size_t j = 0; j < stratum.size(); ++j) {
            if (j > 0) ss << ", ";
            ss << stratum[j];
        }
        if (!last) ss << "\n";
    }
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Strata &s) {
    return os << s.to_string();
}

const Stage &Sequence::operator[](const size_t idx) const {
    return this->stages[idx];
}

const Stage &Sequence::next(const std::string &stage_key) const {
    for (size_t i = 0; i < this->stages.size(); ++i)
        if (this->stages[i].key == stage_key) {
            if (i + 1 >= this->stages.size())
                throw std::runtime_error("no next stage after: " + stage_key);
            return this->stages[i + 1];
        }
    throw std::runtime_error("stage not found: " + stage_key);
}

std::string Sequence::to_string() const {
    return this->to_string_with_prefix("");
}

std::string Sequence::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    ss << this->key;
    for (size_t i = 0; i < this->stages.size(); ++i) {
        bool last = (i == this->stages.size() - 1);
        ss << "\n" << prefix << tree_prefix(last);
        ss << this->stages[i].to_string_with_prefix(prefix + tree_indent(last));
    }
    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const Sequence &s) {
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

const Sequence &IR::sequence(const std::string &key) const {
    for (const auto &s: this->sequences)
        if (s.key == key) return s;
    throw std::runtime_error("sequence not found: " + key);
}

std::vector<Edge> IR::edges_to(const std::string &node_key) const {
    std::vector<Edge> result;
    for (const auto &e: this->edges)
        if (e.target.node == node_key) result.push_back(e);
    return result;
}

std::string IR::to_string() const {
    return this->to_string_with_prefix("");
}

std::string IR::to_string_with_prefix(const std::string &prefix) const {
    std::ostringstream ss;
    ss << "IR";

    bool has_functions = !this->functions.empty();
    bool has_nodes = !this->nodes.empty();
    bool has_edges = !this->edges.empty();
    bool has_strata = !this->strata.empty();
    bool has_sequences = !this->sequences.empty();

    if (has_functions) {
        bool last = !has_nodes && !has_edges && !has_strata && !has_sequences;
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
        bool last = !has_edges && !has_strata && !has_sequences;
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
        bool last = !has_strata && !has_sequences;
        ss << "\n" << prefix << tree_prefix(last) << "Edges";
        std::string child_prefix = prefix + tree_indent(last);
        for (size_t i = 0; i < this->edges.size(); ++i) {
            bool e_last = (i == this->edges.size() - 1);
            ss << "\n" << child_prefix << tree_prefix(e_last);
            ss << this->edges[i].to_string();
        }
    }

    if (has_strata) {
        bool last = !has_sequences;
        ss << "\n" << prefix << tree_prefix(last) << "Strata";
        std::string child_prefix = prefix + tree_indent(last);
        ss << "\n" << this->strata.to_string_with_prefix(child_prefix);
    }

    if (has_sequences) {
        ss << "\n" << prefix << tree_prefix(true) << "Sequences";
        std::string child_prefix = prefix + tree_indent(true);
        for (size_t i = 0; i < this->sequences.size(); ++i) {
            bool s_last = (i == this->sequences.size() - 1);
            ss << "\n" << child_prefix << tree_prefix(s_last);
            ss << this->sequences[i].to_string_with_prefix(
                child_prefix + tree_indent(s_last)
            );
        }
    }

    return ss.str();
}

std::ostream &operator<<(std::ostream &os, const IR &ir) {
    return os << ir.to_string();
}
}
