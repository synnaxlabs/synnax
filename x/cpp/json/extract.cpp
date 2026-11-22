// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <charconv>
#include <cstddef>
#include <optional>
#include <string>
#include <string_view>
#include <system_error>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/extract.h"
#include "x/cpp/json/literal.h"
#include "x/cpp/json/pointer.h"

namespace x::json {
namespace {
/// @brief bounds the nesting depth of a document, as the Core does.
constexpr std::size_t MAX_DEPTH = 10000;

/// @brief returns the array index that token addresses, or nothing when it addresses no
/// element.
std::optional<std::size_t> array_index(const std::string_view token) {
    if (token.empty() || (token.size() > 1 && token[0] == '0')) return std::nullopt;
    std::size_t i = 0;
    const auto *end = token.data() + token.size();
    const auto [ptr, ec] = std::from_chars(token.data(), end, i);
    if (ec != std::errc() || ptr != end) return std::nullopt;
    return i;
}

std::string quote_char(const char c) {
    const auto u = static_cast<unsigned char>(c);
    if (u >= 0x20 && u < 0x7F && c != '\'' && c != '\\')
        return std::string{'\'', c, '\''};
    const char *const hex = "0123456789abcdef";
    return std::string{'\'', '\\', 'x', hex[u >> 4], hex[u & 0xF], '\''};
}
}

Extractor::Extractor(std::vector<Pointer> pointers):
    pointers(std::move(pointers)), nodes(1) {
    for (std::size_t p = 0; p < this->pointers.size(); p++) {
        std::size_t n = 0;
        for (const auto &token: this->pointers[p].tokens) {
            const auto found = this->nodes[n].names.find(token);
            if (found != this->nodes[n].names.end()) {
                n = found->second;
                continue;
            }
            const auto child = this->nodes.size();
            this->nodes.emplace_back();
            this->nodes[n].names.emplace(token, child);
            if (const auto i = array_index(token); i.has_value())
                this->nodes[n].elements.emplace(*i, child);
            n = child;
        }
        this->nodes[n].pointers.push_back(p);
    }
}

errors::Error Extractor::extract(
    const std::string_view doc,
    std::vector<std::optional<std::string_view>> &values
) {
    values.assign(this->pointers.size(), std::nullopt);
    this->doc = doc;
    this->pos = 0;
    this->depth = 0;
    this->containers.clear();
    if (const auto err = this->walk(0, values)) return err;
    this->skip_whitespace();
    if (this->pos != this->doc.size())
        return errors::Error(
            errors::VALIDATION,
            "unexpected data after the top-level value"
        );
    return errors::NIL;
}

errors::Error Extractor::walk(
    const std::size_t node,
    std::vector<std::optional<std::string_view>> &values
) {
    this->skip_whitespace();
    const auto start = this->pos;
    const auto &n = this->nodes[node];
    const char c = this->peek();
    errors::Error err;
    if (c == '{' && !n.names.empty())
        err = this->walk_object(node, values);
    else if (c == '[' && !n.elements.empty())
        err = this->walk_array(node, values);
    else
        err = this->skip_value();
    if (err) return err;
    for (const auto p: n.pointers) {
        if (values[p].has_value())
            return errors::Error(
                errors::VALIDATION,
                "duplicate value at " + this->pointers[p].to_string()
            );
        values[p] = this->doc.substr(start, this->pos - start);
    }
    return errors::NIL;
}

errors::Error Extractor::walk_object(
    const std::size_t node,
    std::vector<std::optional<std::string_view>> &values
) {
    if (const auto err = this->enter()) return err;
    this->skip_whitespace();
    if (this->peek() == '}') {
        this->leave();
        return errors::NIL;
    }
    while (true) {
        std::string_view member;
        if (const auto err = this->read_name(member)) return err;
        const auto &names = this->nodes[node].names;
        const auto child = names.find(member);
        const auto value_err = child != names.end() ? this->walk(child->second, values)
                                                    : this->skip_value();
        if (value_err) return value_err;
        this->skip_whitespace();
        if (this->peek() == '}') {
            this->leave();
            return errors::NIL;
        }
        if (const auto err = this->expect(',')) return err;
    }
}

errors::Error Extractor::walk_array(
    const std::size_t node,
    std::vector<std::optional<std::string_view>> &values
) {
    if (const auto err = this->enter()) return err;
    this->skip_whitespace();
    if (this->peek() == ']') {
        this->leave();
        return errors::NIL;
    }
    for (std::size_t i = 0;; i++) {
        const auto &elements = this->nodes[node].elements;
        const auto child = elements.find(i);
        const auto value_err = child != elements.end()
                                 ? this->walk(child->second, values)
                                 : this->skip_value();
        if (value_err) return value_err;
        this->skip_whitespace();
        if (this->peek() == ']') {
            this->leave();
            return errors::NIL;
        }
        if (const auto err = this->expect(',')) return err;
    }
}

errors::Error Extractor::skip_value() {
    const auto base = this->containers.size();
    while (true) {
        this->skip_whitespace();
        const char c = this->peek();
        if (c == '{' || c == '[') {
            if (const auto err = this->enter()) return err;
            this->containers.push_back(c);
            this->skip_whitespace();
            if (this->peek() != (c == '{' ? '}' : ']')) {
                std::string_view member;
                if (c == '{')
                    if (const auto err = this->read_name(member)) return err;
                continue;
            }
            this->leave();
            this->containers.pop_back();
        } else if (const auto err = this->skip_scalar())
            return err;
        while (this->containers.size() > base) {
            this->skip_whitespace();
            const char open = this->containers.back();
            if (this->peek() == (open == '{' ? '}' : ']')) {
                this->leave();
                this->containers.pop_back();
                continue;
            }
            if (const auto err = this->expect(',')) return err;
            std::string_view member;
            if (open == '{')
                if (const auto err = this->read_name(member)) return err;
            break;
        }
        if (this->containers.size() == base) return errors::NIL;
    }
}

errors::Error Extractor::skip_scalar() {
    const char c = this->peek();
    if (c == '"') {
        const auto scan = scan_literal(this->doc.substr(this->pos), nullptr);
        this->pos += scan.end;
        if (!scan.problem.empty()) return this->fail(scan.problem);
        return errors::NIL;
    }
    for (const std::string_view literal: {"true", "false", "null"}) {
        if (c != literal[0]) continue;
        for (const char want: literal) {
            if (this->pos == this->doc.size() || this->doc[this->pos] != want)
                return this->unexpected();
            this->pos++;
        }
        return errors::NIL;
    }
    if (c != '-' && (c < '0' || c > '9')) return this->unexpected();
    const auto digits = [this]() -> errors::Error {
        const auto start = this->pos;
        while (this->peek() >= '0' && this->peek() <= '9')
            this->pos++;
        return this->pos > start ? errors::NIL : this->unexpected();
    };
    if (c == '-') this->pos++;
    if (this->peek() == '0')
        this->pos++;
    else if (const auto err = digits())
        return err;
    if (this->peek() == '.') {
        this->pos++;
        if (const auto err = digits()) return err;
    }
    if (this->peek() == 'e' || this->peek() == 'E') {
        this->pos++;
        if (this->peek() == '+' || this->peek() == '-') this->pos++;
        if (const auto err = digits()) return err;
    }
    return errors::NIL;
}

errors::Error Extractor::read_name(std::string_view &out) {
    this->skip_whitespace();
    if (this->pos == this->doc.size() || this->doc[this->pos] != '"')
        return this->unexpected();
    const auto rest = this->doc.substr(this->pos);
    const auto scan = scan_literal(rest, nullptr);
    this->pos += scan.end;
    if (!scan.problem.empty()) return this->fail(scan.problem);
    const auto literal = rest.substr(0, scan.end);
    out = literal.substr(1, literal.size() - 2);
    if (out.contains('\\')) {
        this->name.clear();
        scan_literal(literal, &this->name);
        out = this->name;
    }
    this->skip_whitespace();
    return this->expect(':');
}

errors::Error Extractor::enter() {
    if (++this->depth > MAX_DEPTH) return this->fail("exceeded max depth");
    this->pos++;
    return errors::NIL;
}

void Extractor::leave() {
    this->pos++;
    this->depth--;
}

errors::Error Extractor::expect(const char c) {
    if (this->pos == this->doc.size() || this->doc[this->pos] != c)
        return this->unexpected();
    this->pos++;
    return errors::NIL;
}

char Extractor::peek() const {
    return this->pos < this->doc.size() ? this->doc[this->pos] : '\0';
}

void Extractor::skip_whitespace() {
    while (this->pos < this->doc.size()) {
        const char c = this->doc[this->pos];
        if (c != ' ' && c != '\t' && c != '\n' && c != '\r') return;
        this->pos++;
    }
}

errors::Error Extractor::unexpected() const {
    if (this->pos == this->doc.size()) return this->fail("unexpected EOF");
    return this->fail("invalid character " + quote_char(this->doc[this->pos]));
}

errors::Error Extractor::fail(const std::string_view problem) const {
    return errors::Error(
        errors::VALIDATION,
        std::string(problem) + " after offset " + std::to_string(this->pos)
    );
}
}
