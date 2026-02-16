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

std::string Stage::to_string() const {
    std::ostringstream ss;
    ss << "Stage(" << this->key << ")";
    if (!this->nodes.empty()) {
        ss << " nodes=[";
        for (size_t i = 0; i < this->nodes.size(); ++i) {
            if (i > 0) ss << ", ";
            ss << this->nodes[i];
        }
        ss << "]";
    }
    return ss.str();
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
    std::ostringstream ss;
    ss << "Sequence(" << this->key << ") [";
    for (size_t i = 0; i < this->stages.size(); ++i) {
        if (i > 0) ss << " -> ";
        ss << this->stages[i].key;
    }
    ss << "]";
    return ss.str();
}

const Node &IR::node(const std::string &key) const {
    for (const auto &n: nodes)
        if (n.key == key) return n;
    throw std::runtime_error("node not found: " + key);
}

const Function &IR::function(const std::string &key) const {
    for (const auto &f: functions)
        if (f.key == key) return f;
    throw std::runtime_error("function not found: " + key);
}

std::optional<Edge> IR::edge_to(const Handle &target) const {
    for (const auto &e: edges)
        if (e.target == target) return e;
    return std::nullopt;
}

std::unordered_map<std::string, std::vector<Edge>>
IR::edges_from(const std::string &node_key) const {
    std::unordered_map<std::string, std::vector<Edge>> result;
    for (const auto &e: edges)
        if (e.source.node == node_key) result[e.source.param].push_back(e);
    return result;
}

const Sequence &IR::sequence(const std::string &key) const {
    for (const auto &s: sequences)
        if (s.key == key) return s;
    throw std::runtime_error("sequence not found: " + key);
}

[[nodiscard]] std::vector<Edge> IR::edges_to(const std::string &node_key) const {
    std::vector<Edge> result;
    for (const auto &e: edges)
        if (e.target.node == node_key) result.push_back(e);
    return result;
}
}
