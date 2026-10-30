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

#include "client/cpp/ni/types.gen.h"
#include "x/cpp/errors/errors.h"

#include "driver/ni/daqmx/sugared.h"

namespace driver::ni {
/// @brief reads the ports, rates, and ranges of a device from DAQmx. An attribute that
/// the device does not support reads as empty or absent. To find what Cfg_Default gives
/// on an analog input port, it creates a task with that port, reads the configuration
/// back, and clears the task without starting it. A failed readback leaves that default
/// absent and does not fail the probe; is_complete then returns false.
/// @param dmx the DAQmx library.
/// @param device the DAQmx name of the device, such as Dev1.
/// @returns the hardware of the device, or the first error that DAQmx returns.
std::pair<::synnax::ni::Hardware, x::errors::Error>
probe(daqmx::SugaredAPI &dmx, const std::string &device);

/// @brief returns whether the hardware holds every value that probe reads, which is
/// false when a default terminal configuration failed to read back.
bool is_complete(const ::synnax::ni::Hardware &hw);
}
