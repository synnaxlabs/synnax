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
#include <utility>
#include <vector>

#include "x/cpp/errors/errors.h"

#include "driver/serial/list.h"

namespace driver::serial {
/// @brief lists the serial ports in the sysfs tree at sys, as list does.
/// @param dev the device tree that holds the nodes of the ports and their serial/by-id
/// links.
std::pair<std::vector<Entry>, x::errors::Error>
list(const std::filesystem::path &sys, const std::filesystem::path &dev);
}
