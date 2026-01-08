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

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/xlib.h"

namespace driver {
const x::errors::Error BASE_ERROR = x::errors::SY.sub("driver");
const x::errors::Error HARDWARE_ERROR = BASE_ERROR.sub("hardware");
const x::errors::Error CRITICAL_HARDWARE_ERROR = HARDWARE_ERROR.sub("critical");
const x::errors::Error TEMPORARY_HARDWARE_ERROR = HARDWARE_ERROR.sub("temporary");
const x::errors::Error CONFIGURATION_ERROR = BASE_ERROR.sub("configuration");

/// Vendor library definitions
struct LibraryInfo {
    std::string name;
    std::string url;
};

/// Standardized missing library error
inline x::errors::Error missing_lib(const LibraryInfo &lib) {
    std::string message = lib.name + " library is not installed.";
    if (!lib.url.empty()) {
        message += " Download here: " + lib.url +
                   ". Restart Driver after installation.";
    }
    return x::errors::Error(x::lib::LOAD_ERROR, message);
}

/// @brief wraps an error with channel name and hardware location context for easier
/// debugging. The hardware location is integration-specific (e.g., node_id for OPC UA,
/// port for LabJack, physical_channel for NI, address for Modbus).
inline x::errors::Error wrap_channel_error(
    const x::errors::Error &err,
    const std::string &channel_name,
    const std::string &hardware_location
) {
    return x::errors::Error(
        err,
        channel_name + " (" + hardware_location + "): " + err.data
    );
}

}
