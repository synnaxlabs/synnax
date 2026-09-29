// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <filesystem>
#include <string>
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"

namespace driver::serial {
/// @brief a failure to list the serial ports.
const x::errors::Error SCAN_ERROR = driver::errors::HARDWARE_ERROR.sub("serial_scan");

/// @brief a serial port found by scan.
struct Info {
    /// @brief the path that opens the port, such as /dev/serial/by-id/usb-FTDI_...,
    /// /dev/cu.usbserial-1, or COM3.
    std::string path;
    /// @brief a readable name for the port.
    std::string name;
};

/// @brief lists the serial ports on the host: /dev/serial/by-id links and ttyS, ttyUSB,
/// and ttyACM devices on Linux, /dev/cu.* on macOS, and the SERIALCOMM registry key on
/// Windows. A Linux device reached through a by-id link is listed only by the link.
/// @param root the directory under which to look for dev and sys. Windows ignores it.
/// @returns the ports sorted by path, or SCAN_ERROR when the listing fails.
std::pair<std::vector<Info>, x::errors::Error>
scan(const std::filesystem::path &root = "/");
}
