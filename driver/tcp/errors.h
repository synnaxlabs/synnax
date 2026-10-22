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

namespace driver::tcp {
/// @brief maps a native error from a TCP connection to a transport error whose message
/// is context followed by the error's message.
/// @returns UNREACHABLE_ERROR when the peer refuses, resets, closes, or times out, its
/// name does not resolve, or the network is down. CONFIG_ERROR when the host forbids
/// the connection or rejects a setting. CRITICAL_ERROR when the host runs out of
/// resources or the error is unknown.
x::errors::Error translate(const std::string &context, const std::error_code &ec);
}
