// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>
#include <memory>
#include <optional>
#include <string>
#include <vector>

#include "client/cpp/channel/channel.h"
#include "client/cpp/ni/chassis/types.gen.h"
#include "client/cpp/ni/device/types.gen.h"
#include "client/cpp/rack/key.h"
#include "x/cpp/errors/errors.h"
#include "x/cpp/telem/series.h"

#include "driver/common/sync.h"
#include "driver/ni/scan_task.h"

namespace driver::ni {
/// @brief returns the chassis among the scanned hardware.
std::vector<synnax::ni::chassis::Chassis>
found_chassis(const std::vector<Device> &scanned);

/// @brief returns the key that matches scanned hardware to its stored resource: the
/// serial number, or the resource name when the hardware has no serial number.
std::string match_key(const Device &scanned);

/// @brief returns the key of the chassis of a module.
/// @param module the match key of the module.
/// @param chassis the match key of the scanned chassis whose link the module connects
/// to.
/// @returns nothing when the module sits under its rack.
using ChassisKey = std::function<std::optional<
    synnax::ni::chassis::Key>(const std::string &module, const std::string &chassis)>;

/// @brief returns the modules and standalone devices among the scanned hardware.
std::vector<synnax::ni::device::Device>
found_devices(const std::vector<Device> &scanned, const ChassisKey &chassis_key);

/// @brief keeps the NI chassis and devices of a rack in step with the hardware that a
/// scan finds. Resources is safe for concurrent use.
class Resources {
    common::Sync<synnax::ni::chassis::Chassis> chassis;
    common::Sync<synnax::ni::device::Device> devices;

public:
    Resources(
        std::unique_ptr<common::Store<synnax::ni::chassis::Chassis>> chassis,
        std::unique_ptr<common::Store<synnax::ni::device::Device>> devices,
        synnax::rack::Key rack
    );

    /// @brief the channels whose frames go to on_signal.
    static const std::vector<std::string> SIGNALS;

    /// @brief saves the scanned hardware: the chassis first, then the devices under
    /// them.
    /// @returns the error of the chassis sync, which skips the device sync, or the
    /// error of the device sync.
    x::errors::Error sync(const std::vector<Device> &scanned);

    /// @brief returns the stored hardware of the scanned device, if the Driver holds
    /// the device.
    [[nodiscard]] std::optional<synnax::ni::device::Hardware>
    stored_hardware(const Device &scanned);

    /// @brief tracks the chassis and devices that the series on ch sets or deletes.
    /// @throws std::invalid_argument when ch is not in SIGNALS.
    void on_signal(const synnax::channel::Channel &ch, const x::telem::Series &series);
};
}
