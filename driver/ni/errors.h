// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/errors/errors.h"

namespace ni {
const driver::LibraryInfo NI_DAQMX = {
    "National Instruments NI-DAQmx shared",
    "https://www.ni.com/en/support/downloads/drivers/download.ni-daq-mx.html"
};
const driver::LibraryInfo NI_SYSCFG = {
    "National Instruments System Configuration",
    "https://www.ni.com/en/support/downloads/drivers/download.system-configuration.html"
};

const xerrors::Error END_OF_ENUM("ni.end_of_enum");
}
