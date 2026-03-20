// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

#include "core/pkg/distribution/ontology/pb/ontology.pb.h"

namespace synnax::ontology {
/// @brief An ontology ID is a composite identifier consisting of a type and key.
/// The type represents the resource type (e.g., "channel", "user", "group"),
/// and the key represents the unique identifier within that type.
/// String representation: "type:key" (colon-separated)
/// Example: "channel:42", "group:748d31e2-5732-4cb5-8bc9-64d4ad51efe8"
struct ID {
    /// @brief The resource type (e.g., "channel", "user", "group", "rack", "device",
    /// "task")
    std::string type;
    /// @brief The unique identifier within the resource type
    std::string key;

    /// @brief Returns the string representation of the ID in "type:key" format.
    /// @returns A string in the format "type:key".
    [[nodiscard]] std::string string() const;

    /// @brief Parses a string in "type:key" format into an ID.
    /// @param s The string to parse.
    /// @returns A pair containing the parsed ID and an error. If parsing fails,
    /// the error will have ok() == false.
    [[nodiscard]] static std::pair<ID, x::errors::Error> parse(const std::string &s);

    /// @brief Returns true if both type and key are empty.
    [[nodiscard]] bool is_zero() const { return type.empty() && key.empty(); }

    /// @brief Parses an ID from a JSON parser.
    static ID parse(x::json::Parser parser);

    /// @brief Serializes the ID to JSON.
    [[nodiscard]] x::json::json to_json() const;

    /// @brief Constructs an ID from its protobuf representation.
    using proto_type = ::distribution::ontology::pb::ID;
    static std::pair<ID, x::errors::Error>
    from_proto(const ::distribution::ontology::pb::ID &pb);

    /// @brief Serializes the ID to its protobuf representation.
    [[nodiscard]] std::pair<::distribution::ontology::pb::ID, x::errors::Error>
    to_proto() const;

    /// @brief Equality operator.
    bool operator==(const ID &other) const;

    /// @brief Inequality operator.
    bool operator!=(const ID &other) const;
};

/// @brief The root ID used as the top-level parent in the ontology hierarchy.
const ID ROOT_ID{.type = "builtin", .key = "root"};

/// @brief Parses a vector of strings into a vector of IDs.
/// @param strs The strings to parse, each in "type:key" format.
/// @returns A pair containing the parsed IDs and an error. If any parse fails,
/// the error will have ok() == false and indicate the first failure.
[[nodiscard]] std::pair<std::vector<ID>, x::errors::Error>
parse_ids(const std::vector<std::string> &strs);

/// @brief Converts a vector of IDs to a vector of strings.
/// @param ids The IDs to convert.
/// @returns A vector of strings in "type:key" format.
[[nodiscard]] std::vector<std::string> ids_to_strings(const std::vector<ID> &ids);
}
