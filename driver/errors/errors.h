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

#include "x/cpp/xerrors/errors.h"
#include "x/cpp/xlib/xlib.h"

namespace driver {
const xerrors::Error BASE_ERROR = xerrors::SY.sub("driver");
/// @brief a general hardware error for a device.
const xerrors::Error HARDWARE_ERROR = BASE_ERROR.sub("hardware");
/// @brief a critical hardware error for a device that should not be retried.
const xerrors::Error CRITICAL_HARDWARE_ERROR = HARDWARE_ERROR.sub("critical");
/// @brief a temporary hardware error for a device that should be retried.
const xerrors::Error TEMPORARY_HARDWARE_ERROR = HARDWARE_ERROR.sub("temporary");
/// @brief a configuration error for a device, task, integration, etc.
const xerrors::Error CONFIGURATION_ERROR = BASE_ERROR.sub("configuration");
/// @brief sentinel indicating expected shutdown, not an error condition.
const xerrors::Error NOMINAL_SHUTDOWN_ERROR = BASE_ERROR.sub("nominal_shutdown");

/// Vendor library definitions
struct LibraryInfo {
    std::string name;
    std::string url;
};

/// Standardized missing library error
inline xerrors::Error missing_lib(const LibraryInfo &lib) {
    std::string message = lib.name + " library is not installed.";
    if (!lib.url.empty()) {
        message += " Download here: " + lib.url +
                   ". Restart Driver after installation.";
    }
    return xerrors::Error(xlib::LOAD_ERROR, message);
}

/// @brief wraps an error with channel name and hardware location context for easier
/// debugging. The hardware location is integration-specific (e.g., node_id for OPC UA,
/// port for LabJack, physical_channel for NI, address for Modbus).
inline xerrors::Error wrap_channel_error(
    const xerrors::Error &err,
    const std::string &channel_name,
    const std::string &hardware_location
) {
    return xerrors::Error(
        err,
        channel_name + " (" + hardware_location + "): " + err.data
    );
}

}
