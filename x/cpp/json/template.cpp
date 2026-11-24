// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <algorithm>
#include <cstddef>
#include <functional>
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/template.h"
#include "x/cpp/json/text.h"

namespace x::json {
namespace {
/// @brief Node is one value of the JSON document that a Template renders.
struct Node {
    /// @brief name is the member name of the node in its parent object.
    std::string name;
    /// @brief members are the members of an object, in the order fields create them.
    std::vector<Node> members;
    /// @brief field is the position of the field that sets the node, or nothing for an
    /// object that fields create.
    std::optional<std::size_t> field;
};

/// @brief returns the position of a field that sets n or a value inside it.
std::size_t overlapped(const Node &n) {
    const Node *c = &n;
    while (!c->field.has_value())
        c = &c->members.front();
    return c->field.value();
}

errors::Error overlap(
    const std::vector<Template::Field> &fields,
    const std::size_t a,
    const std::size_t b
) {
    return errors::Error(
        errors::VALIDATION,
        fields[a].label + " " + quote(fields[a].pointer.to_string()) + " and " +
            fields[b].label + " " + quote(fields[b].pointer.to_string()) + " overlap"
    );
}

/// @brief adds field i to the tree at root.
/// @returns VALIDATION when the pointer of field i overlaps one that root holds.
errors::Error
set(Node &root, const std::vector<Template::Field> &fields, const std::size_t i) {
    const auto &tokens = fields[i].pointer.tokens;
    Node *n = &root;
    for (std::size_t depth = 0; depth < tokens.size(); depth++) {
        if (n->field.has_value()) return overlap(fields, n->field.value(), i);
        const auto last = depth == tokens.size() - 1;
        const auto c = std::ranges::find(n->members, tokens[depth], &Node::name);
        if (c == n->members.end()) {
            n->members.push_back({.name = tokens[depth]});
            if (last) n->members.back().field = i;
            n = &n->members.back();
            continue;
        }
        if (last) return overlap(fields, overlapped(*c), i);
        n = &*c;
    }
    if (tokens.empty()) {
        if (n->field.has_value() || !n->members.empty())
            return overlap(fields, overlapped(*n), i);
        n->field = i;
    }
    return errors::NIL;
}

/// @brief appends n to text, which holds the JSON text since the last slot. render
/// passes the text before each slot to add_slot, with the position of the slot's
/// field, and then clears text.
/// @returns VALIDATION when a member name is not valid UTF-8.
errors::Error render(
    const std::vector<Template::Field> &fields,
    const Node &n,
    std::string &text,
    const std::function<void(std::string prefix, std::size_t field)> &add_slot
) {
    if (n.field.has_value()) {
        const auto field = n.field.value();
        if (const auto &literal = fields[field].literal; literal.has_value()) {
            text += literal.value();
            return errors::NIL;
        }
        add_slot(std::move(text), field);
        text.clear();
        return errors::NIL;
    }
    text += '{';
    for (std::size_t i = 0; i < n.members.size(); i++) {
        const auto &m = n.members[i];
        if (i > 0) text += ',';
        if (const auto err = append_quoted(m.name, text))
            return errors::Error(
                errors::VALIDATION,
                "pointer token " + quote(m.name) + ": " + err.data
            );
        text += ':';
        if (const auto err = render(fields, m, text, add_slot)) return err;
    }
    text += '}';
    return errors::NIL;
}
}

std::pair<Template, errors::Error> Template::create(const std::vector<Field> &fields) {
    Node root;
    for (std::size_t i = 0; i < fields.size(); i++)
        if (const auto err = set(root, fields, i)) return {Template(), err};
    Template t;
    std::string text;
    const auto add_slot = [&t](std::string prefix, const std::size_t field) {
        t.segments.push_back({std::move(prefix), field});
    };
    if (const auto err = render(fields, root, text, add_slot)) return {Template(), err};
    t.suffix = std::move(text);
    return {std::move(t), errors::NIL};
}
}
