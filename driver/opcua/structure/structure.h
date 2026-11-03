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
#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "open62541/types.h"

#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/telem.h"

#include "driver/opcua/connection/connection.h"
#include "driver/opcua/types/types.h"

namespace driver::opcua::structure {
/// @brief a path of member names into a structure, such as Motor.Speed.
using Path = std::vector<std::string>;

/// @brief parses a field: member names joined by '.', where '&' escapes the next
/// character, so "a&.b" names the member "a.b".
/// @returns x::errors::VALIDATION if a name is empty or the field ends in '&'.
std::pair<Path, x::errors::Error> parse(const std::string &field);

/// @returns the field of a path, the inverse of parse.
std::string to_string(const Path &path);

/// @brief the server gives no definition of a data type, so the Driver cannot read its
/// values as structures.
const x::errors::Error UNDEFINED = x::errors::VALIDATION.sub("undefined");

/// @brief a member of a structure type.
struct Member {
    /// @brief the name of the member in its structure.
    std::string name;
    /// @brief the data type of the member.
    const UA_DataType *type;
    /// @brief whether the member holds an array of its type.
    bool is_array;
};

/// @brief where a member lies in the values of a structure type, found once so that
/// reads of the member do not search the type.
class Field {
    friend class Types;
    struct Step {
        std::size_t offset;
        bool optional;
        std::string name;
    };
    const UA_DataType *root = nullptr;
    std::vector<Step> steps;
    const UA_DataType *member = nullptr;

public:
    /// @returns the data type of the member.
    [[nodiscard]] const UA_DataType &type() const { return *this->member; }
};

/// @brief the structure and enumeration data types of one server, built from their
/// definitions on the server, together with the types the client already knows. It
/// decodes values that the client cannot decode alone. Namespace indexes in the types
/// are those of the connection that read them.
class Types {
    struct Entry {
        UA_DataTypeArray array{};
        UA_DataType type{};

        Entry() = default;
        Entry(const Entry &) = delete;
        Entry &operator=(const Entry &) = delete;
        ~Entry() { UA_DataType_clear(&this->type); }
    };

    /// @brief each entry's array links to the previous entry's, so the last entry heads
    /// a list of every type. The entries never move, since types point at the types of
    /// their members.
    std::vector<std::unique_ptr<Entry>> entries;
    /// @brief why each data type that the server defined does not build.
    std::vector<std::pair<types::NodeId, std::string>> failures;

    [[nodiscard]] const UA_DataTypeArray *head() const;

public:
    /// @brief reads the definitions of the data types that the client does not know,
    /// with the types their members use. A type that does not build keeps the reason,
    /// which field returns.
    /// @returns the error of a request, such as errors::UNREACHABLE when the connection
    /// is lost.
    static std::pair<Types, x::errors::Error> read(
        const connection::Pool::Connection &conn,
        const std::vector<UA_NodeId> &data_types
    );

    /// @returns the type with the data type ID, or nullptr if neither the client nor
    /// this set knows it.
    [[nodiscard]] const UA_DataType *find(const UA_NodeId &data_type) const;

    /// @returns the members of a structure type, or nothing for another kind of type.
    [[nodiscard]] static std::vector<Member> members(const UA_DataType &type);

    /// @returns the member at path in values of the data type. An empty path names the
    /// whole value.
    /// @returns UNDEFINED if the server gives no definition of the data type.
    /// @returns x::errors::VALIDATION if the type does not build, or a member does not
    /// exist or is an array.
    [[nodiscard]] std::pair<Field, x::errors::Error>
    field(const UA_NodeId &data_type, const Path &path) const;

    /// @returns the structure in value, decoded when the client could not decode it.
    /// The result borrows from value when the client decoded it.
    /// @returns x::errors::VALIDATION if the value is not one structure that decodes.
    [[nodiscard]] std::pair<types::Variant, x::errors::Error>
    decode(const UA_Variant &value) const;

    /// @returns the member at field of a structure that decode returned, converted to
    /// the data type.
    /// @returns x::errors::VALIDATION if the structure is of another type, lacks an
    /// optional member, or the member does not convert.
    [[nodiscard]] static std::pair<x::telem::SampleValue, x::errors::Error> sample(
        const UA_Variant &structure,
        const Field &field,
        const x::telem::DataType &dt
    );
};
}
