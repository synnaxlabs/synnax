// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <optional>
#include <string>
#include <thread>
#include <unordered_set>
#include <vector>

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"
#include "x/cpp/lib/lazy.h"

#include "driver/common/scan_task.h"
#include "driver/ni/ni.h"
#include "driver/ni/syscfg/nisyscfg.h"
#include "driver/ni/syscfg/sugared.h"

namespace driver::ni {
const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task] ";
const std::string RESET_DEVICE_CMD = "reset_device";

struct ResetDeviceCommandArgs {
    std::vector<std::string> device_keys;

    explicit ResetDeviceCommandArgs(x::json::Parser &parser):
        device_keys(parser.field<std::vector<std::string>>("device_keys")) {}
};

/// @brief an extension of the default synnax device that also includes NI related
/// properties.
struct Device : synnax::device::Device {
    /// @brief the raw NI resource name.
    std::string resource_name;
    /// @brief whether the device is simulated.
    bool is_simulated = false;
    /// @brief whether this device is a chassis.
    bool is_chassis = false;
    /// @brief link name this device connects to (modules connect to chassis links).
    std::string connects_to_link_name;
    /// @brief link name this device provides (chassis provide links for modules).
    std::string provides_link_name;

    Device() = default;

    explicit Device(
        const synnax::device::Device &device,
        std::string resource_name,
        const bool is_simulated
    ):
        synnax::device::Device(device),
        resource_name(std::move(resource_name)),
        is_simulated(is_simulated) {}

    /// @brief returns the synnax device representation along with json serialized
    /// properties.
    synnax::device::Device to_synnax() {
        auto dev = synnax::device::Device{
            .key = this->key,
            .rack = this->rack,
            .location = this->location,
            .make = this->make,
            .model = this->model,
            .name = this->name,
            .properties =
                x::json::json{
                    {"is_simulated", this->is_simulated},
                    {"resource_name", this->resource_name},
                    {"is_chassis", this->is_chassis},
                }
                    .get<x::json::json::object_t>(),
        };
        dev.status = this->status;
        return dev;
    }
};

/// @brief configuration for opening a scan task.
struct ScanTaskConfig : common::ScanTaskConfig {
    explicit ScanTaskConfig(x::json::Parser &cfg): common::ScanTaskConfig(cfg) {}
};

/// @brief a task that scans for NI devices.
class Scanner final : public common::Scanner {
    /// @brief configuration for the scan task.
    const ScanTaskConfig cfg;
    const synnax::task::Task task;
    /// @brief loads the NI System Configuration library.
    std::shared_ptr<x::lib::Lazy<syscfg::SugaredAPI>> lib;
    /// @brief the NI System Configuration library, set by the first open.
    std::shared_ptr<syscfg::SugaredAPI> syscfg;
    /// @brief ni system configuration session handle.
    NISysCfgSessionHandle session = nullptr;
    /// @brief ni filter we use to only find certain ni devices. Null while closed.
    NISysCfgFilterHandle filter = nullptr;
    /// @brief keys of devices already logged as ignored by the model filter.
    std::unordered_set<std::string> ignored_keys;

    /// @brief parses the device located at the specified resource handle.
    /// @returns the parsed device and x::errors::NIL error if successful.
    /// @returns the device and an SKIP_DEVICE_ERR error if the device should be
    /// skipped.
    /// @returns an empty device and an error if the device could not be parsed.
    std::pair<ni::Device, x::errors::Error>
    parse_device(NISysCfgResourceHandle resource);

    /// @brief loads the library, then opens the session and the device filter.
    x::errors::Error open();

    /// @brief closes the filter and the session that are open.
    x::errors::Error close();

    [[nodiscard]] common::ScannerConfig config() const override {
        return common::ScannerConfig{.make = MAKE, .log_prefix = SCAN_LOG_PREFIX};
    }

public:
    explicit Scanner(
        std::shared_ptr<x::lib::Lazy<syscfg::SugaredAPI>> lib,
        ScanTaskConfig cfg,
        synnax::task::Task task
    );

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override;

    x::errors::Error stop() override;
};
}
