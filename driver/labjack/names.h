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

namespace driver::labjack {
/// @brief make of LabJack devices.
const std::string MAKE = "LabJack";
/// @brief labjack integration name.
const std::string INTEGRATION_NAME = "labjack";
const std::string SCAN_TASK_TYPE = "labjack_scan";
const std::string READ_TASK_TYPE = "labjack_read";
const std::string WRITE_TASK_TYPE = "labjack_write";
}
