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
#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/pointer.h"

namespace x::json {
/// @brief Template renders JSON documents whose members keep the order of the fields
/// that set them. Everything but the slots renders once, so a render costs only its
/// slots. It is safe for concurrent use.
class Template {
public:
    /// @brief Field is one value of the documents a Template renders.
    struct Field {
        /// @brief label names the field in errors, such as field "source".
        std::string label;
        /// @brief pointer places the value. Its tokens name object members, never
        /// array indexes.
        Pointer pointer;
        /// @brief literal is the JSON text of a value that every document holds, or
        /// nothing for a slot that each render fills.
        std::optional<std::string> literal;
    };

private:
    struct Segment {
        /// @brief prefix is the JSON text before the slot.
        std::string prefix;
        std::size_t field = 0;
    };

    std::vector<Segment> segments;
    /// @brief suffix is the JSON text after the last slot.
    std::string suffix;

public:
    /// @brief creates a Template that appends nothing. Assign one from create before
    /// use.
    Template() = default;

    /// @brief creates a Template for fields, which it creates the objects of.
    /// @returns VALIDATION when two pointers are equal or one holds the other, since a
    /// document can hold only one of them. The error names both fields in the order of
    /// fields. Also VALIDATION when a pointer token is not valid UTF-8.
    static std::pair<Template, errors::Error> create(const std::vector<Field> &fields);

    /// @brief appends a document to out. fill(field, out) appends the JSON value of a
    /// slot to out, where field is the position of the slot's field in the fields of
    /// create.
    /// @returns the first error that fill returns, appending nothing.
    template<typename Fill>
    [[nodiscard]] errors::Error append(std::string &out, const Fill &fill) const {
        const auto start = out.size();
        for (const auto &s: this->segments) {
            out += s.prefix;
            if (auto err = fill(s.field, out)) {
                out.resize(start);
                return err;
            }
        }
        out += this->suffix;
        return errors::NIL;
    }
};
}
