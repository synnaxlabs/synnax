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

#include "x/cpp/json/json.h"

namespace x::filter {
/// @brief describes the records that a filter tree selects as a short sentence.
/// @param tree the JSON form of the filter tree.
/// @returns a sentence such as `active = true and name starts with "press"`, or an
/// empty string when the tree sets no conditions.
[[nodiscard]] std::string describe(const x::json::json &tree);
}
