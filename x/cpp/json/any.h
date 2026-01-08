// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <utility>

#include "google/protobuf/any.pb.h"
#include "nlohmann/json.hpp"

#include "x/cpp/errors/errors.h"
#include "x/cpp/json/struct.h"

namespace x::json {
inline google::protobuf::Any to_any(const json &j) {
    google::protobuf::Any any;
    auto [s, err] = to_struct(j);
    if (err) return any;
    any.PackFrom(s);
    return any;
}

inline std::pair<nlohmann::json, errors::Error>
from_any(const google::protobuf::Any &any) {
    google::protobuf::Struct s;
    if (!any.UnpackTo(&s))
        return {
            {},
            errors::Error(errors::VALIDATION, "failed to unpack Any to Struct")
        };
    return from_struct(s);
}
}
