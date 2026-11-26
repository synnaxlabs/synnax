// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#if defined(__linux__)
#include <asm/termbits.h>
#elif defined(__APPLE__)
#include <termios.h>
#endif

#include "client/cpp/serial/types.gen.h"
#include "x/cpp/errors/errors.h"

/// @brief the mapping from serial properties onto the platform's line settings.
namespace driver::serial::line {
#if defined(__linux__)
/// @brief the kernel's termios2, which carries any baud rate. It conflicts with
/// <termios.h>, so a file that includes this header cannot include that one.
using Settings = ::termios2;
#elif defined(__APPLE__)
using Settings = ::termios;
#endif

/// @brief writes the baud rate, character size, parity, stop bits, and flow control in
/// props onto settings, and keeps its other fields. On macOS, a baud rate outside the
/// termios table leaves the speeds unchanged, and the port sets the rate after it
/// applies settings. props must hold data bits from 5 to 8 and a parity, stop bits, and
/// flow control that the serial schema defines.
/// @returns CONFIG_ERROR when the platform cannot express a setting.
x::errors::Error apply(const synnax::serial::Properties &props, Settings &settings);
}
