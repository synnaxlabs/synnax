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
#include "x/cpp/loop/loop.h"

/// internal
#include "driver/task/task.h"
#include "driver/ni/ni.h"
#include "driver/ni/syscfg/nisyscfg.h"
#include "driver/ni/syscfg/sugared.h"

namespace ni {
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

const auto DEFAULT_SCAN_RATE = telem::Rate(telem::SECOND * 5);
const std::vector<std::string> DEFAULT_IGNORED_MODELS = {"^O.*", "^cRIO.*", "^nown.*"};

/// @brief configuration for opening a scan task.
struct ScanTaskConfig {
    const telem::Rate rate;
    const bool enabled;
    std::vector<std::regex> ignored_models;

    explicit ScanTaskConfig(
        xjson::Parser &cfg
    ): rate(telem::Rate(cfg.optional<double>("rate", DEFAULT_SCAN_RATE.hz()))),
       enabled(cfg.optional<bool>("enabled", true)) {
        const auto ignored = cfg.optional_array<std::string>(
            "ignored_models",
            DEFAULT_IGNORED_MODELS
        );
        for (const auto &pattern: ignored)
            ignored_models.emplace_back(pattern);
    }
};

class ScanTask final : public task::Task {
    /// @brief the raw synnax task configuration.
    const synnax::Task task;
    /// @brief configuration for the scan task.
    const ScanTaskConfig cfg;
    /// @brief the breaker for managing the lifecycle of threads.
    breaker::Breaker breaker;
    /// @brief the scanner used to scan for devices.
    loop::Timer timer;
    /// @brief the task context to communicate state updates and device changes.
    std::shared_ptr<task::Context> ctx;
    /// @brief the scan thread that will scan for devices.
    std::shared_ptr<std::thread> thread;
    /// @brief the current list of scanned devices.
    std::unordered_map<std::string, ni::Device> devices;
    /// @brief the NI system configuration library.
    std::shared_ptr<syscfg::SugaredAPI> syscfg;
    /// @brief ni system configuration session handle.
    NISysCfgSessionHandle session = nullptr;
    /// @brief ni filter we use to only find certain ni devices;
    NISysCfgFilterHandle filter = nullptr;
    /// @brief the current task state.
    task::State state;

    /// @brief parses the device located at the specified resource handle.
    /// @returns the parsed device and xerrors::NIL error if successful.
    /// @returns the device and an SKIP_DEVICE_ERR error if the device should be skipped.
    /// @returns an empty device and an error if the device could not be parsed.
    std::pair<ni::Device, xerrors::Error> parse_device(
        NISysCfgResourceHandle resource) const;

    /// @brief scans the hardware for devices.
    xerrors::Error find_devices();

    /// @brief updates the list of devices in the synnax cluster.
    xerrors::Error update_remote();

    void run();
public:
    explicit ScanTask(
        const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        ScanTaskConfig cfg
    );

    void exec(task::Command &cmd) override;

    xerrors::Error initialize_syscfg_session();

    void stop(bool will_reconfigure) override;

    xerrors::Error start();

    xerrors::Error scan();

    std::string name() override { return task.name; }

    static std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure(
        const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );
}; // class ScannerTask
} // namespace ni
