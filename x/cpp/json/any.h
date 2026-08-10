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

namespace x::json {
/// @brief Converts json to a google::protobuf::Any that holds a Struct.
/// @param j The JSON to convert. It must be an object or null. A null value converts to
/// an empty object. Any other value returns a VALIDATION error.
/// @returns A pair containing the Any and an error if one occurred.
inline std::pair<google::protobuf::Any, errors::Error> to_any(const json &j) {
    if (!j.is_null() && !j.is_object())
        return {
            {},
            errors::Error(
                errors::VALIDATION,
                std::string("expected a JSON object, got ") + j.type_name()
            )
        };
    google::protobuf::Struct s;
    if (j.is_object())
        if (const auto err = to_struct(j, &s)) return {{}, err};
    google::protobuf::Any any;
    if (!any.PackFrom(s))
        return {{}, errors::Error(errors::INTERNAL, "failed to pack Struct into Any")};
    return {any, errors::NIL};
}

inline std::pair<nlohmann::json, errors::Error>
from_any(const google::protobuf::Any &any) {
    // Handle empty Any (no type_url set) - return empty JSON object
    if (any.type_url().empty()) return {json::object(), errors::NIL};
    google::protobuf::Struct s;
    if (!any.UnpackTo(&s))
        return {
            {},
            errors::Error(errors::VALIDATION, "failed to unpack Any to Struct")
        };
    return from_struct(s);
}
}
