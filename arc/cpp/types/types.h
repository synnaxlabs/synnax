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

#include "arc/cpp/types/json.gen.h"
#include "arc/cpp/types/proto.gen.h"
#include "arc/cpp/types/types.gen.h"

namespace arc::types {
using ChannelKey = std::uint32_t;

/// @brief Converts a JSON value to a SampleValue based on the given type.
std::optional<x::telem::SampleValue>
to_sample_value(const x::json::json &value, const Type &type);

/// @brief returns what to multiply a variable by to get the unit of the input that
/// reads it, and what to divide it by. A variable in s read by an input in ns gives 1e9
/// and 1e-9.
std::pair<double, double> input_scale(const Type &type);
}
