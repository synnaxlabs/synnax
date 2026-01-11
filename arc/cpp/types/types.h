// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <optional>

#include "x/cpp/telem/telem.h"

#include "arc/cpp/types/json.gen.h"
#include "arc/cpp/types/proto.gen.h"
#include "arc/cpp/types/types.gen.h"

namespace arc::types {
using ChannelKey = std::uint32_t;

/// @brief Finds a param in a Params vector by name.
/// @param params The Params vector to search.
/// @param name The name of the param to find.
/// @returns The param with the given name, or std::nullopt if not found.
[[nodiscard]] inline std::optional<std::reference_wrapper<const Param>>
find_param(const Params &params, const std::string &name) {
    for (const auto &param : params) {
        if (param.name == name) return std::cref(param);
    }
    return std::nullopt;
}

/// @brief Converts a JSON value to a SampleValue based on the given Type.
/// @param value The JSON value to convert.
/// @param type The Arc type that determines how to interpret the JSON value.
/// @returns The converted SampleValue, or std::nullopt if the value is null.
[[nodiscard]] std::optional<x::telem::SampleValue>
to_sample_value(const x::json::json &value, const Type &type);
}