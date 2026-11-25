// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <algorithm>
#include <cerrno>
#include <initializer_list>
#include <string>
#include <system_error>

#include "asio/error.hpp"
#include "asio/error_code.hpp"

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::transport {
/// @brief UNREACHABLE_ERROR is a device that cannot be opened or reached, or a
/// connection that dropped. Reopening the device can fix it.
const x::errors::Error UNREACHABLE_ERROR = driver::errors::TEMPORARY_HARDWARE_ERROR.sub(
    "unreachable"
);
/// @brief CONFIG_ERROR is device properties that are invalid or that the platform
/// cannot apply.
const x::errors::Error CONFIG_ERROR = driver::errors::CONFIGURATION_ERROR.sub(
    "transport"
);
/// @brief CRITICAL_ERROR is a failure that reopening the device cannot fix, such as
/// running out of file descriptors.
const x::errors::Error CRITICAL_ERROR = driver::errors::CRITICAL_HARDWARE_ERROR.sub(
    "transport"
);

/// @brief HOST_DOWN is the native error of a host on the local network that stops
/// answering, which no std::errc names.
#ifdef _WIN32
inline const std::error_code HOST_DOWN(WSAEHOSTDOWN, std::system_category());
#else
inline const std::error_code HOST_DOWN(EHOSTDOWN, std::system_category());
#endif

/// @returns true when ec equals one of codes.
inline bool
matches(const std::error_code &ec, const std::initializer_list<std::errc> codes) {
    return std::ranges::any_of(codes, [&ec](const std::errc code) {
        return ec == code;
    });
}

/// @returns ec in std::system_category when Asio gives it in its own category, which
/// compares equal to few std::errc values.
inline std::error_code from_asio(const std::error_code &ec) {
    if (ec.category() != asio::system_category()) return ec;
    return {ec.value(), std::system_category()};
}

/// @returns an error of type whose message is context, then the message of ec.
inline x::errors::Error error(
    const x::errors::Error &type,
    const std::string &context,
    const std::error_code &ec
) {
    return x::errors::Error(type, context + ": " + ec.message());
}
}
