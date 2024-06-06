// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>

namespace driver {
    const std::string ERROR_PREFIX = "sy.driver.";
    const std::string TYPE_CRITICAL_HARDWARE_ERROR = ERROR_PREFIX + "hardware.critical";
    const std::string TEMPORARY_HARDWARE_ERROR = ERROR_PREFIX + "hardware.temporary";
    const std::string TYPE_CONFIGURATION_ERROR = ERROR_PREFIX + "configuration";
}