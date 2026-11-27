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

#include "client/cpp/device/device.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/json/json.h"

namespace driver::bus {
/// @brief Device is a device that bus tasks run on, with the Properties of its
/// transport.
template<typename Properties>
struct Device {
    /// @brief key is the key of the device.
    std::string key;
    /// @brief name is the name of the device.
    std::string name;
    /// @brief props is the properties of the transport of the device.
    Properties props;
};

/// @brief retrieves the device with key.
/// @returns the error of the retrieve, or VALIDATION when the properties of the device
/// are not valid Properties.
template<typename Properties>
std::pair<Device<Properties>, x::errors::Error>
retrieve_device(const synnax::device::Client &devices, const std::string &key) {
    auto [dev, err] = devices.retrieve(key);
    if (err) return {Device<Properties>{}, err};
    x::json::Parser parser(x::json::json(dev.properties));
    auto props = Properties::parse(parser);
    return {
        Device<Properties>{.key = dev.key, .name = dev.name, .props = std::move(props)},
        parser.error(),
    };
}
}
