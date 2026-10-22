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

namespace driver::serial {
/// @brief maps a native error from a serial port to a transport error whose message is
/// context followed by the error's message.
/// @returns UNREACHABLE_ERROR when the port is missing, gone, closed, timed out, or in
/// use by another program. CONFIG_ERROR when the Driver may not use the port or the
/// port rejects a setting. CRITICAL_ERROR when the host runs out of resources or the
/// error is unknown.
x::errors::Error translate(const std::string &context, const std::error_code &ec);
}
