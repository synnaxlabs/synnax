// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>
#include <vector>
#include <thread>
#include <regex>

/// external
#include "nlohmann/json.hpp"

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"

/// internal
#include "driver/task/task.h"
#include "driver/ni/ni.h"
#include "driver/ni/syscfg/nisyscfg.h"
#include "driver/ni/syscfg/sugared.h"
#include "driver/task/common/scan_task.h"

namespace ni {
const std::string RESET_DEVICE_CMD = "reset_device";

struct ResetDeviceCommandArgs {
    std::vector<std::string> device_keys;

    explicit ResetDeviceCommandArgs(xjson::Parser &parser)
        : device_keys(parser.required_vec<std::string>("device_keys")) {
    }
};


/// @brief an extension of the default synnax device that also includes NI related
/// properties.
struct Device : synnax::Device {
    /// @brief the raw NI resource name.
    std::string resource_name;
    /// @brief whether the device is simulated.
    bool is_simulated = false;

    Device() = default;

    explicit Device(
        const synnax::Device &device,
        std::string resource_name,
        const bool is_simulated
    ): synnax::Device(device),
       resource_name(std::move(resource_name)),
       is_simulated(is_simulated) {
    }

    /// @brief returns the synnax device representation along with json serialized
    /// properties.
    synnax::Device to_synnax() {
        return synnax::Device(
            this->key,
            this->name,
            this->rack,
            this->location,
            this->identifier,
            this->make,
            this->model,
            nlohmann::to_string(json{
                {"is_simulated", this->is_simulated},
                {"resource_name", this->resource_name}
            })
        );
    }
};

/// @brief the default rate for scanning for devices.
const auto DEFAULT_SCAN_RATE = telem::Rate(telem::SECOND * 5);
/// @brief the default pattern for ignoring certain models.
const std::vector<std::string> DEFAULT_IGNORED_MODELS = {"^O.*", "^cRIO.*", "^nown.*"};

/// @brief configuration for opening a scan task.
struct ScanTaskConfig {
    /// @brief the rate at which we'll can for devices.
    const telem::Rate rate;
    /// @brief whether the scan task is enabled.
    const bool enabled;
    /// @brief a set of regex patterns to ignore certain devices when scanning.
    std::vector<std::regex> ignored_models;

    explicit ScanTaskConfig(xjson::Parser &cfg):
        rate(telem::Rate(cfg.optional<double>("rate", DEFAULT_SCAN_RATE.hz()))),
        enabled(cfg.optional<bool>("enabled", true)) {
        const auto i = cfg.optional_vec<std::string>(
            "ignored_models",
            DEFAULT_IGNORED_MODELS
        );
        for (const auto &pattern: i) ignored_models.emplace_back(pattern);
    }

    /// @brief returns if the device with the given model should be ignored.
    bool should_ignore(const std::string &model) const {
        for (const auto &pattern: this->ignored_models)
            if (std::regex_match(model, pattern)) return true;
        return false;
    }
};

/// @brief a task that scans for NI devices.
class Scanner final : public common::Scanner {
    /// @brief configuration for the scan task.
    const ScanTaskConfig cfg;
    const synnax::Task task;
    /// @brief the NI system configuration library.
    std::shared_ptr<syscfg::SugaredAPI> syscfg;
    /// @brief ni system configuration session handle.
    NISysCfgSessionHandle session = nullptr;
    /// @brief ni filter we use to only find certain ni devices;
    NISysCfgFilterHandle filter = nullptr;

    /// @brief parses the device located at the specified resource handle.
    /// @returns the parsed device and xerrors::NIL error if successful.
    /// @returns the device and an SKIP_DEVICE_ERR error if the device should be skipped.
    /// @returns an empty device and an error if the device could not be parsed.
    std::pair<ni::Device, xerrors::Error> parse_device(
        NISysCfgResourceHandle resource
    ) const;
public:
    explicit Scanner(
        const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
        ScanTaskConfig cfg,
        const synnax::Task &task
    );

    xerrors::Error start() override;

    std::pair<std::vector<synnax::Device>, xerrors::Error> scan(const common::ScannerContext &ctx) override;

    xerrors::Error stop() override;

}; 
} 
