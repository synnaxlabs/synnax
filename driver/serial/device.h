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

#include "asio/serial_port.hpp"

#include "x/cpp/errors/errors.h"

/// @brief opens the device of a port. Each platform implements it in its own file.
namespace driver::serial::device {
/// @brief opens device at port and locks it against other programs before it changes
/// any setting. device stays closed on failure.
/// @returns UNREACHABLE_ERROR when the port is missing or in use.
x::errors::Error open(asio::serial_port &device, const std::string &port);
}
