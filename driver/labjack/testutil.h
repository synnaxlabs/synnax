// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <cstdint>
#include <random>
#include <string>
#include <variant>

#include "gtest/gtest.h"

#include "client/cpp/labjack/device/device.h"
#include "client/cpp/synnax.h"
#include "x/cpp/testutil/testutil.h"

namespace driver::labjack {
/// @brief creates a LabJack device of the model on the rack, with a serial number that
/// no other test device has.
/// @returns the key that a task config names the device by.
inline std::string create_task_device(
    const synnax::Synnax &client,
    const synnax::rack::Key rack,
    const std::string &model = synnax::labjack::device::MODEL_T_7
) {
    namespace lj = synnax::labjack::device;
    static std::mt19937 gen{std::random_device{}()};
    lj::Ports ports = lj::T7Ports{};
    if (model == lj::MODEL_T_4) ports = lj::T4Ports{};
    if (model == lj::MODEL_T_8) ports = lj::T8Ports{};
    lj::Device dev{
        .rack = rack,
        .name = "my_device",
        .serial = std::uniform_int_distribution<std::uint32_t>{1, 4'000'000'000}(gen),
        .connection = lj::CONNECTION_USB,
        .model = model,
        .ports = ports,
    };
    EXPECT_NIL(client.labjack.devices.create(dev));
    return std::to_string(dev.serial);
}
}
