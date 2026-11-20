// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <functional>
#include <map>
#include <optional>
#include <string>
#include <string_view>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/pointer.h"

namespace x::json {
/// @brief Extractor resolves a fixed set of pointers in one pass over a JSON document.
/// It neither copies nor decodes the values the pointers address. It is not safe for
/// concurrent use.
class Extractor {
    struct Node {
        /// @brief pointers are the positions of the pointers that end here.
        std::vector<std::size_t> pointers;
        /// @brief names holds the next node by object member name.
        std::map<std::string, std::size_t, std::less<>> names;
        /// @brief elements holds the next node by array index.
        std::map<std::size_t, std::size_t> elements;
    };

    std::vector<Pointer> pointers;
    /// @brief nodes holds the steps of the pointers, with the root first.
    std::vector<Node> nodes;
    std::string_view doc;
    std::size_t pos = 0;
    /// @brief depth is the nesting depth of the containers being read.
    std::size_t depth = 0;
    /// @brief containers holds the open containers of the value skip_value reads.
    std::vector<char> containers;
    /// @brief name is scratch space for an object member name holding an escape.
    std::string name;

    errors::Error
    walk(std::size_t node, std::vector<std::optional<std::string_view>> &values);
    errors::Error
    walk_object(std::size_t node, std::vector<std::optional<std::string_view>> &values);
    errors::Error
    walk_array(std::size_t node, std::vector<std::optional<std::string_view>> &values);
    errors::Error skip_value();
    errors::Error skip_scalar();
    errors::Error read_name(std::string_view &out);
    errors::Error enter();
    void leave();
    errors::Error expect(char c);
    /// @brief returns the byte at pos, or 0 at the end of the document.
    [[nodiscard]] char peek() const;
    void skip_whitespace();
    /// @brief returns the error for the byte at pos, which no JSON grammar rule takes.
    [[nodiscard]] errors::Error unexpected() const;
    [[nodiscard]] errors::Error fail(std::string_view problem) const;

public:
    /// @brief creates an Extractor for pointers.
    explicit Extractor(std::vector<Pointer> pointers);

    /// @brief resolves every pointer in doc into values, which it resizes to hold one
    /// entry for each pointer. Each value is the text of doc at the pointer, or nothing
    /// when doc holds no value there.
    /// @returns VALIDATION when doc is not one valid JSON document, or when a member
    /// name repeats on the path to an addressed value.
    errors::Error
    extract(std::string_view doc, std::vector<std::optional<std::string_view>> &values);
};
}
