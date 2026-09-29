// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <system_error>

/// @brief settings that Asio's serial port options cannot express, applied through the
/// port's native handle. Each platform implements them in its own file, and this header
/// stays free of Asio because Linux's termios2 conflicts with <termios.h>.
namespace driver::serial::native {
/// @brief a file descriptor on POSIX or a HANDLE on Windows.
using Handle = std::intptr_t;

/// @brief a flow control mode.
enum class FlowControl { NONE, SOFTWARE, HARDWARE };

/// @brief sets the flow control mode. Asio cannot set hardware flow control on macOS.
std::error_code set_flow_control(Handle handle, FlowControl mode);

/// @brief sets a baud rate that is absent from the platform's table of standard rates.
std::error_code set_baud_rate(Handle handle, std::uint32_t rate);

/// @brief sets mark parity when mark is true, and space parity otherwise.
std::error_code set_mark_space_parity(Handle handle, bool mark);

/// @brief makes the driver assert RTS while sending, to turn an RS-485 transceiver
/// around.
std::error_code enable_rs485(Handle handle);
}
