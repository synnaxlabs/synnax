// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "open62541/types.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/opcua/connection/connection.h"

namespace driver::opcua::structure {
/// @brief a path of member names into a structure, such as Motor.Speed.
using Path = std::vector<std::string>;

/// @brief parses a field: member names joined by '.', where '&' escapes the next
/// character, so "a&.b" names the member "a.b".
/// @returns x::errors::VALIDATION if a name is empty or the field ends in '&'.
std::pair<Path, x::errors::Error> parse(const std::string &field);

/// @returns the field of a path, the inverse of parse.
std::string to_string(const Path &path);

/// @brief a member of a structure type.
struct Member {
    std::string name;
    const UA_DataType *type;
    bool is_array;
};

/// @brief the structure and enumeration data types of one server, built from their
/// definitions on the server, together with the types the client already knows. It
/// decodes values that the client cannot decode alone. Namespace indexes in the types
/// are those of the connection that read them.
class Types {
    struct Entry {
        UA_DataTypeArray array{};
        UA_DataType type{};
    };

    /// @brief each entry's array links to the previous entry's, so the last entry
    /// heads a list of every type. The entries never move, since types point at the
    /// types of their members.
    std::vector<std::unique_ptr<Entry>> entries;

    [[nodiscard]] const UA_DataTypeArray *head() const;

    /// @brief calls fn with the member at path of a scalar value, decoded when it is a
    /// structure that the client does not know.
    template<typename R, typename F>
    std::pair<R, x::errors::Error>
    visit(const UA_Variant &value, const Path &path, const F &fn) const;

public:
    Types() = default;
    ~Types();
    Types(Types &&other) noexcept;
    Types &operator=(Types &&other) noexcept;
    Types(const Types &) = delete;
    Types &operator=(const Types &) = delete;

    /// @brief reads the data types of the nodes, and the definitions of the types the
    /// client does not know, with the types their members use. A type whose definition
    /// the server does not give is left out, so its values do not decode.
    /// @returns the error of a request, such as errors::UNREACHABLE when the
    /// connection is lost.
    static std::pair<Types, x::errors::Error>
    read(const connection::Pool::Connection &conn, const std::vector<UA_NodeId> &nodes);

    /// @returns the type with the data type ID, or nullptr if neither the client nor
    /// this set knows it.
    [[nodiscard]] const UA_DataType *find(const UA_NodeId &data_type) const;

    /// @returns the members of a structure type, or nothing for another kind of type.
    [[nodiscard]] static std::vector<Member> members(const UA_DataType &type);

    /// @returns the type of the member at path in a structure type.
    /// @returns x::errors::VALIDATION if a member does not exist or is an array.
    [[nodiscard]] static std::pair<const UA_DataType *, x::errors::Error>
    member_type(const UA_DataType &type, const Path &path);

    /// @returns the member at path of a structure value, converted to the data type.
    /// @returns x::errors::VALIDATION if the value does not decode, or has no such
    /// member, or the member does not convert.
    [[nodiscard]] std::pair<x::telem::SampleValue, x::errors::Error> sample(
        const UA_Variant &value,
        const Path &path,
        const x::telem::DataType &dt
    ) const;

    /// @returns the member at path of a value as JSON text, or the whole value when
    /// path is empty. An array is a JSON array of its elements.
    /// @returns x::errors::VALIDATION if the value does not decode or has no such
    /// member.
    [[nodiscard]] std::pair<std::string, x::errors::Error>
    print(const UA_Variant &value, const Path &path) const;
};
}
