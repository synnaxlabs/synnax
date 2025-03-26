// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>
#include <cstdlib>
#include <type_traits>

/// external
#include "glog/logging.h"

namespace xenv {
template<typename T>
T load(const std::string& name, const T& default_value) {
    const char* value = std::getenv(name.c_str());
    if (value == nullptr) return default_value;
    VLOG(1) << "Loaded " << name << " from environment variable.";
    try {
        if constexpr (std::is_same_v<T, std::string>)
            return std::string(value);
        if constexpr (std::is_floating_point_v<T>)
            return static_cast<T>(std::stold(value));
        if constexpr (std::is_unsigned_v<T>)
            return static_cast<T>(std::stoull(value));
        if constexpr (std::is_integral_v<T>)
            return static_cast<T>(std::stoll(value));
        return default_value;
    } catch (const std::exception&) {
        return default_value;
    }
}

} // namespace xenv
