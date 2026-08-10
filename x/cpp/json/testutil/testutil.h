// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/json/json.h"

namespace x::json {
/// @brief returns an object nested 150 levels deep. The protobuf JSON parser accepts
/// 100 levels, so every conversion to a Struct rejects this object.
inline json::object_t deeply_nested_object() {
    json j = json::object();
    for (int i = 0; i < 150; i++)
        j = json{{"nested", j}};
    return j.get<json::object_t>();
}
}
