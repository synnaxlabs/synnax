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

#include "google/protobuf/any.pb.h"
#include "nlohmann/json.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/struct.h"
#include "x/cpp/json/value.h"

namespace x::json {
/// @brief Converts json to a google::protobuf::Any that holds a Value.
/// @param j The JSON to convert. Every JSON type is valid.
/// @returns A pair containing the Any and an error if one occurred.
inline std::pair<google::protobuf::Any, errors::Error> to_any(const json &j) {
    auto [v, err] = to_value(j);
    if (err) return {{}, err};
    google::protobuf::Any any;
    if (!any.PackFrom(v))
        return {{}, errors::Error(errors::INTERNAL, "failed to pack Value into Any")};
    return {any, errors::NIL};
}

/// @brief Converts a google::protobuf::Any that holds a Value or a Struct to json.
/// @param any The Any to convert. An unset Any converts to null.
/// @returns A pair containing the JSON and an error if one occurred.
inline std::pair<nlohmann::json, errors::Error>
from_any(const google::protobuf::Any &any) {
    if (any.type_url().empty()) return {json(), errors::NIL};
    if (google::protobuf::Value v; any.UnpackTo(&v)) return from_value(v);
    // Peers on releases before value packing send a Struct.
    if (google::protobuf::Struct s; any.UnpackTo(&s)) return from_struct(s);
    return {
        {},
        errors::Error(
            errors::VALIDATION,
            "failed to unpack Any of type " + any.type_url()
        )
    };
}
}
