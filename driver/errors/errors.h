// Copyright 2025 Synnax Labs, Inc.
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
const xerrors::Error HARDWARE_ERROR = BASE_ERROR.sub("hardware");
const xerrors::Error CRITICAL_HARDWARE_ERROR = HARDWARE_ERROR.sub("critical");
const xerrors::Error TEMPORARY_HARDWARE_ERROR = HARDWARE_ERROR.sub("temporary");
const xerrors::Error CONFIGURATION_ERROR = BASE_ERROR.sub("configuration");

/// Vendor library definitions
struct LibraryInfo {
    std::string name;
    std::string url;
};

namespace lib {
const LibraryInfo LABJACK_LJM = {
    "LabJack LJM shared",
    "https://support.labjack.com/docs/ljm-software-installer-downloads-t4-t7-t8-digit"
};
const LibraryInfo NI_DAQMX = {
    "National Instruments NI-DAQmx shared",
    "https://www.ni.com/en/support/downloads/drivers/download.ni-daq-mx.html"
};
const LibraryInfo NI_SYSCFG = {
    "National Instruments System Configuration",
    "https://www.ni.com/en/support/downloads/drivers/download.system-configuration.html"
};
}

/// Standardized missing library error
inline xerrors::Error missing_lib(const LibraryInfo &lib) {
    std::string message = lib.name + " library is not installed.";
    if (!lib.url.empty()) {
        message += " Download here: " + lib.url + ". Restart Driver after installation.";
    }
    return xerrors::Error(xlib::LOAD_ERROR, message);
}

}
