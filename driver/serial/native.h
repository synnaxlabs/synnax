// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstddef>
#include <cstdint>
#include <system_error>
#include <utility>

#include "client/cpp/serial/types.gen.h"
#include "x/cpp/errors/errors.h"

/// @brief port operations on the native handle. Each platform implements them in its
/// own file, and this header stays free of Asio because Linux's termios2 conflicts with
/// <termios.h>.
namespace driver::serial::native {
/// @brief a file descriptor on POSIX or a HANDLE on Windows.
using Handle = std::intptr_t;

/// @brief applies the line settings in props to the open port. props must pass the
/// checks that Port::open makes.
/// @returns CONFIG_ERROR when the platform or the port cannot apply a setting.
x::errors::Error configure(Handle handle, const synnax::serial::Properties &props);

/// @returns the number of written bytes that the port has not yet sent.
std::pair<std::size_t, std::error_code> queued_output(Handle handle);
}
