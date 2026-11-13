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
#include <utility>

#include "x/cpp/errors/errors.h"

#include "driver/ni/syscfg/sugared.h"

namespace driver::ni {
/// @brief strips the delimiters around a resource name that NI System Configuration
/// reports, such as the quotes around a GUID.
/// @param raw the resource name as NI System Configuration reports it.
/// @returns the resource name that the Driver stores.
std::string trim_resource_name(const std::string &raw);

/// @brief finds the DAQmx name of the present device with the given resource name.
/// DAQmx calls take the device's NI-MAX alias, which a user can rename at any time,
/// so a task looks it up when it starts.
/// @param syscfg the NI System Configuration library.
/// @param resource_name the resource name of the device, as the Driver stores it.
/// @returns the alias, and an error that matches NOT_FOUND when no present device has
/// the resource name.
std::pair<std::string, x::errors::Error>
find_alias(syscfg::SugaredAPI &syscfg, const std::string &resource_name);
}
