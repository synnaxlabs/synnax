// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// module
#include "client/cpp/synnax.h"

/// internal
#include "driver/ni/daqmx/sugared.h"
#include "driver/ni/syscfg/sugared.h"
#include "driver/task/task.h"
#include "driver/task/common/sample_clock.h"

namespace ni {
const std::string MAKE = "NI";
const std::string INTEGRATION_NAME = "ni";
const std::string SCAN_TASK_TYPE = "ni_scanner";
const std::string ANALOG_READ_TASK_TYPE = "ni_analog_read";
const std::string DIGITAL_READ_TASK_TYPE = "ni_digital_read";
const std::string ANALOG_WRITE_TASK_TYPE = "ni_analog_write";
const std::string DIGITAL_WRITE_TASK_TYPE = "ni_digital_write";
const std::vector UNREACHABLE_ERRORS = {
    daqmx::DEVICE_DISCONNECTED,
    daqmx::RESOURCE_NOT_AVAILABLE
};

inline xerrors::Error translate_error(const xerrors::Error &err) {
    if (err.matches(UNREACHABLE_ERRORS))
        return daqmx::TEMPORARILY_UNREACHABLE;
    if (err.matches(daqmx::APPLICATION_TOO_SLOW))
        return {
            xerrors::Error(
                driver::CRITICAL_HARDWARE_ERROR,
                "the network cannot keep up with the stream rate specified. try making the sample rate a higher multiple of the stream rate"
            )
        };
    return err.skip(daqmx::ANALOG_WRITE_OUT_OF_BOUNDS);
}

/// @brief a factory for instantiating and operating NI data acquisition, control,
/// and device scanning tasks.
class Factory final : public task::Factory {
    /// @brief the daqmx library used to communicate with NI hardware.
    std::shared_ptr<daqmx::SugaredAPI> dmx;
    /// @brief the system configuration library used to get information
    /// about devices.
    std::shared_ptr<syscfg::SugaredAPI> syscfg;
    common::TimingConfig timing_cfg;

    /// @brief checks whether the factory is healthy and capable of creating tasks.
    /// If not, the factory will automatically send an error back through the
    /// task state and return false.
    [[nodiscard]] bool check_health(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;

public:
    Factory(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        const std::shared_ptr<syscfg::SugaredAPI> &syscfg,
        common::TimingConfig timing_cfg
    );

    /// @brief creates a new NI factory, loading the DAQmx and system configuration
    /// libraries.
    static std::unique_ptr<Factory> create(common::TimingConfig timing_cfg = common::TimingConfig{});

    /// @brief implements task::Factory to process task configuration requests.
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    /// @brief implements task::Factory to configure initial tasks such as the
    /// device scanner.
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;

    template<typename HardwareT, typename ConfigT, typename SourceSinkT, typename TaskT>
    std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        auto [cfg, cfg_err] = ConfigT::parse(ctx->client, task, this->timing_cfg);
        if (cfg_err) return {nullptr, cfg_err};
        TaskHandle handle;
        const std::string dmx_task_name = task.name + " (" + std::to_string(task.key) + ")";
        if (const auto err = this->dmx->CreateTask(dmx_task_name.c_str(), &handle))
            return {nullptr, err};
        // Very important that we instantiate the Hardware API here, as we pass
        // ownership over the lifecycle of the task handle to it. If we encounter any
        // errors when applying the configuration or cycling the task, we need to make
        // sure it gets cleared.
        auto hw = std::make_unique<HardwareT>(this->dmx, handle);
        if (const auto err = cfg.apply(this->dmx, handle)) return {nullptr, err};
        // NI will look for invalid configuration parameters internally, so we quickly
        // cycle the task in order to catch and communicate any errors as soon as
        // possible.
        if (const auto err = hw->start()) return {nullptr, err};
        if (const auto err = hw->stop()) return {nullptr, err};
        return {
            std::make_unique<TaskT>(
                task,
                ctx,
                breaker::default_config(task.name),
                std::make_unique<SourceSinkT>(std::move(cfg), std::move(hw))
            ),
            xerrors::NIL
        };
    }


    std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_scan(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );
};
} 
