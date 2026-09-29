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
#include <system_error>

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::transport {
/// @brief a device that cannot be opened or reached, or a connection that dropped.
const x::errors::Error UNREACHABLE_ERROR = driver::errors::TEMPORARY_HARDWARE_ERROR.sub(
    "unreachable"
);
/// @brief device properties that are invalid or that the platform cannot apply.
const x::errors::Error CONFIG_ERROR = driver::errors::CONFIGURATION_ERROR.sub(
    "transport"
);

/// @returns an error of the given type whose message is context followed by ec's
/// message.
inline x::errors::Error error(
    const x::errors::Error &type,
    const std::string &context,
    const std::error_code &ec
) {
    return x::errors::Error(type, context + ": " + ec.message());
}
}
