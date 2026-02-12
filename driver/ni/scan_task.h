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
// Disable GCC 13 false positive warning in <regex> header
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wmaybe-uninitialized"
#endif
#include <regex>
#if defined(__GNUC__) && !defined(__clang__)
#pragma GCC diagnostic pop
#endif
#include <string>
#include <thread>
#include <vector>

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"

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
        auto dev = synnax::device::Device(
            this->key,
            this->name,
            this->rack,
            this->location,
            this->make,
            this->model,
            nlohmann::to_string(
                x::json::json{
                    {"is_simulated", this->is_simulated},
                    {"resource_name", this->resource_name}
                },
            .status = this->status,
        };
        return dev;
    }
};

/// @brief the default pattern for ignoring certain models.
const std::vector<std::string> DEFAULT_IGNORED_MODELS = {"^cRIO.*", "^nown.*"};
/// @brief configuration for opening a scan task.
struct ScanTaskConfig : driver::task::common::ScanTaskConfig {
    /// @brief a set of regex patterns to ignore certain devices when scanning.
    std::vector<std::regex> ignored_models;

    explicit ScanTaskConfig(x::json::Parser &cfg): common::ScanTaskConfig(cfg) {
        const auto i = cfg.field<std::vector<std::string>>(
            "ignored_models",
            DEFAULT_IGNORED_MODELS
        );
        for (const auto &pattern: i)
            ignored_models.emplace_back(pattern);
    }

    /// @brief returns if the device with the given model should be ignored.
    [[nodiscard]] bool should_ignore(const std::string &model) const {
        for (const auto &pattern: this->ignored_models)
            if (std::regex_match(model, pattern)) return true;
        return false;
    }
};

/// @brief a task that scans for NI devices.
class Scanner final : public driver::task::common::Scanner {
    /// @brief configuration for the scan task.
    const ScanTaskConfig cfg;
    const synnax::task::Task task;
    /// @brief the NI system configuration library.
    std::shared_ptr<syscfg::SugaredAPI> syscfg;
    /// @brief ni system configuration session handle.
    NISysCfgSessionHandle session = nullptr;
    /// @brief ni filter we use to only find certain ni devices;
    NISysCfgFilterHandle filter = nullptr;

    /// @brief parses the device located at the specified resource handle.
    /// @returns the parsed device and x::errors::NIL error if successful.
    /// @returns the device and an SKIP_DEVICE_ERR error if the device should be
    /// skipped.
    /// @returns an empty device and an error if the device could not be parsed.
    std::pair<ni::Device, x::errors::Error>
    parse_device(NISysCfgResourceHandle resource) const;

    driver::task::common::ScannerConfig config() const override {
        return driver::task::common::ScannerConfig{
            .make = MAKE,
            .log_prefix = SCAN_LOG_PREFIX
        };
    }

public:
    explicit Scanner(
        const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
        ScanTaskConfig cfg,
        synnax::task::Task task
    );

    x::errors::Error start() override;

    std::pair<std::vector<synnax::device::Device>, x::errors::Error>
    scan(const common::ScannerContext &ctx) override;

    x::errors::Error stop() override;
};
}
