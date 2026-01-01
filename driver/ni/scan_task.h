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

#include "driver/ni/ni.h"
#include "driver/ni/syscfg/nisyscfg.h"
#include "driver/ni/syscfg/sugared.h"
#include "driver/task/common/scan_task.h"

namespace ni {
const std::string SCAN_LOG_PREFIX = "[" + INTEGRATION_NAME + ".scan_task] ";
const std::string RESET_DEVICE_CMD = "reset_device";

struct ResetDeviceCommandArgs {
    std::vector<std::string> device_keys;

    explicit ResetDeviceCommandArgs(xjson::Parser &parser):
        device_keys(parser.field<std::vector<std::string>>("device_keys")) {}
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
    ):
        synnax::Device(device),
        resource_name(std::move(resource_name)),
        is_simulated(is_simulated) {}

    /// @brief returns the synnax device representation along with json serialized
    /// properties.
    synnax::Device to_synnax() {
        auto dev = synnax::Device(
            this->key,
            this->name,
            this->rack,
            this->location,
            this->make,
            this->model,
            nlohmann::to_string(
                json{
                    {"is_simulated", this->is_simulated},
                    {"resource_name", this->resource_name}
                }
            )
        );
        dev.status = this->status;
        return dev;
    }
};

/// @brief the default pattern for ignoring certain models.
const std::vector<std::string> DEFAULT_IGNORED_MODELS = {"^cRIO.*", "^nown.*"};
/// @brief configuration for opening a scan task.
struct ScanTaskConfig : common::ScanTaskConfig {
    /// @brief a set of regex patterns to ignore certain devices when scanning.
    std::vector<std::regex> ignored_models;

    explicit ScanTaskConfig(xjson::Parser &cfg): common::ScanTaskConfig(cfg) {
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
    /// @returns the device and an SKIP_DEVICE_ERR error if the device should be
    /// skipped.
    /// @returns an empty device and an error if the device could not be parsed.
    std::pair<ni::Device, xerrors::Error>
    parse_device(NISysCfgResourceHandle resource) const;

    common::ScannerConfig config() const override {
        return common::ScannerConfig{.make = MAKE, .log_prefix = SCAN_LOG_PREFIX};
    }

public:
    explicit Scanner(
        const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
        ScanTaskConfig cfg,
        synnax::Task task
    );

    xerrors::Error start() override;

    std::pair<std::vector<synnax::Device>, xerrors::Error>
    scan(const common::ScannerContext &ctx) override;

    xerrors::Error stop() override;
};
}
