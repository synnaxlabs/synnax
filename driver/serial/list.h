// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

namespace driver::serial {
/// @brief Entry is a serial port of this computer.
struct Entry {
    /// @brief path is the path that opens the port: its /dev/serial/by-id link or else
    /// its /dev/tty node on Linux, its /dev/cu node on macOS, or its COM name on
    /// Windows.
    std::string path;
    /// @brief description is the name that the operating system gives the device, such
    /// as its USB product string. It is absent when the operating system gives none.
    std::optional<std::string> description;

    bool operator==(const Entry &) const = default;
};

/// @brief lists the serial ports of this computer, sorted by path. On Linux, it leaves
/// out a port with no node in /dev, such as a port of the host in a container, and the
/// UART ports in which the kernel found no UART.
/// @returns the error of the operating system when it cannot list the ports.
std::pair<std::vector<Entry>, x::errors::Error> list();
}
