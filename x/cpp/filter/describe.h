// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <string>

#include "x/cpp/json/json.h"

namespace x::filter {
/// @returns a short, readable description of the records that the JSON filter tree
/// selects, such as `name starts with "press" and active = true`. An empty tree returns
/// an empty string.
[[nodiscard]] std::string describe(const x::json::json &tree);

/// @returns a short, readable description of the records that where selects, or an
/// empty string when where is empty or absent.
template<typename W>
[[nodiscard]] std::string describe(const std::optional<W> &where) {
    if (!where.has_value()) return "";
    return describe(where->to_json());
}
}
