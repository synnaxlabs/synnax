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
#include <optional>
#include <span>
#include <string>
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

/// @brief keeps other programs from opening the port while the Driver holds it.
/// @returns UNREACHABLE_ERROR when another program holds the port.
x::errors::Error lock(Handle handle, const std::string &port);

/// @brief lets other programs open the port once the Driver closes it.
void unlock(Handle handle);

/// @brief applies the line settings in props to the open port. props must pass the
/// checks that Port::open makes.
/// @returns CONFIG_ERROR when the platform or the port cannot apply a setting.
x::errors::Error configure(Handle handle, const synnax::serial::Properties &props);

/// @returns the number of written bytes that the port has not yet sent.
std::pair<std::size_t, std::error_code> queued_output(Handle handle);

/// @brief drops the written bytes that the port has not yet sent. A close on Linux
/// waits up to 30 s for them otherwise.
void discard_output(Handle handle);

/// @brief Line is the line errors of an open port, carried from one read to the next.
struct Line {
    /// @brief marked is how many bytes of an error mark ended the last read.
    std::uint8_t marked = 0;
    /// @brief overruns is the overrun count that the port last reported, if it counts
    /// overruns.
    std::optional<std::uint32_t> overruns;
};

/// @brief Received is what one read received once its line errors are removed.
struct Received {
    /// @brief size is the number of bytes left at the front of the read.
    std::size_t size = 0;
    /// @brief errors is the number of bytes received with a parity or framing error.
    std::size_t errors = 0;
    /// @brief overruns is the number of receive overruns since the last read.
    std::size_t overruns = 0;
};

/// @brief starts counting line errors from the port's current state.
std::error_code watch(Handle handle, Line &line);

/// @brief removes the bytes with line errors from data, which the port just read, and
/// counts the errors and overruns. When the overrun count fails, it still returns the
/// bytes with the error.
std::pair<Received, std::error_code>
receive(Handle handle, Line &line, std::span<std::uint8_t> data);
}
