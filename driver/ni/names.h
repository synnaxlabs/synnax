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

namespace driver::ni {
const std::string MAKE = "NI";
const std::string INTEGRATION_NAME = "ni";
const std::string SCAN_TASK_TYPE = "ni_scanner";
const std::string ANALOG_READ_TASK_TYPE = "ni_analog_read";
const std::string ANALOG_WRITE_TASK_TYPE = "ni_analog_write";
const std::string COUNTER_READ_TASK_TYPE = "ni_counter_read";
const std::string DIGITAL_READ_TASK_TYPE = "ni_digital_read";
const std::string DIGITAL_WRITE_TASK_TYPE = "ni_digital_write";
}
