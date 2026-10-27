// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <array>
#include <string>
#include <string_view>
#include <utility>
#include <vector>

#include "x/cpp/filter/describe.h"

namespace x::filter {
namespace {
constexpr std::array<std::pair<std::string_view, std::string_view>, 10> OPERATORS{{
    {"eq", "="},
    {"in", "in"},
    {"contains", "contains"},
    {"starts_with", "starts with"},
    {"regex", "matches"},
    {"gt", ">"},
    {"gte", ">="},
    {"lt", "<"},
    {"lte", "<="},
    {"any", "has any of"},
}};

constexpr std::string_view CASE_INSENSITIVE = "case_insensitive";

/// @brief one of the terms that a node joins with `and`.
struct Conjunct {
    std::string text;
    bool disjunction = false;
};

bool is_node_key(const std::string &key) {
    return key == "and" || key == "or" || key == "not";
}

std::string literal(const x::json::json &value) {
    if (!value.is_array()) return value.dump();
    std::string out = "[";
    for (std::size_t i = 0; i < value.size(); ++i) {
        if (i > 0) out += ", ";
        out += literal(value[i]);
    }
    return out + "]";
}

bool is_operator(const std::string &key) {
    return key == CASE_INSENSITIVE ||
           std::ranges::any_of(OPERATORS, [&](const auto &op) {
               return op.first == key;
           });
}

bool is_operators(const x::json::json &value) {
    if (!value.is_object()) return false;
    for (const auto &item: value.items())
        if (!is_operator(item.key())) return false;
    return true;
}

std::vector<std::string> clauses(const std::string &field, const x::json::json &value) {
    if (!is_operators(value)) return {field + " = " + literal(value)};
    const auto ci = value.find(std::string(CASE_INSENSITIVE));
    const std::string suffix = ci != value.end() && ci->is_boolean() && ci->get<bool>()
                                 ? " (ignoring case)"
                                 : "";
    std::vector<std::string> out;
    for (const auto &[op, word]: OPERATORS) {
        const auto it = value.find(std::string(op));
        if (it == value.end() || it->is_null()) continue;
        const auto operand = op == "regex" ? "/" + it->get<std::string>() + "/"
                                           : literal(*it);
        out.push_back(field + " " + std::string(word) + " " + operand + suffix);
    }
    return out;
}

std::string join(const std::vector<Conjunct> &terms) {
    std::string out;
    for (std::size_t i = 0; i < terms.size(); ++i) {
        if (i > 0) out += " and ";
        const auto &[text, disjunction] = terms[i];
        out += disjunction && terms.size() > 1 ? "(" + text + ")" : text;
    }
    return out;
}

std::vector<Conjunct> conjuncts(const x::json::json &node);

std::string disjunct(const std::vector<Conjunct> &terms) {
    if (terms.empty()) return "true";
    if (terms.size() == 1) return terms.front().text;
    return "(" + join(terms) + ")";
}

std::vector<Conjunct> conjuncts(const x::json::json &node) {
    std::vector<std::string> fields;
    for (const auto &item: node.items())
        if (!is_node_key(item.key()) && !item.value().is_null())
            fields.push_back(item.key());
    std::ranges::sort(fields);
    std::vector<Conjunct> terms;
    for (const auto &field: fields)
        for (auto &text: clauses(field, node.at(field)))
            terms.push_back({std::move(text)});
    if (const auto it = node.find("and"); it != node.end() && it->is_array())
        for (const auto &child: *it)
            for (auto &term: conjuncts(child))
                terms.push_back(std::move(term));
    if (const auto it = node.find("or"); it != node.end() && it->is_array()) {
        if (it->empty())
            terms.push_back({"false"});
        else if (it->size() == 1)
            for (auto &term: conjuncts(it->front()))
                terms.push_back(std::move(term));
        else {
            std::string text;
            for (std::size_t i = 0; i < it->size(); ++i) {
                if (i > 0) text += " or ";
                text += disjunct(conjuncts((*it)[i]));
            }
            terms.push_back({std::move(text), true});
        }
    }
    if (const auto it = node.find("not"); it != node.end() && it->is_object()) {
        const auto inner = conjuncts(*it);
        terms.push_back({"not (" + (inner.empty() ? "true" : join(inner)) + ")"});
    }
    return terms;
}
}

std::string describe(const x::json::json &tree) {
    return join(conjuncts(tree));
}
}
