// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "absl/log/log.h"

#include "x/cpp/lib/lazy.h"

#include "driver/common/common.h"
#include "driver/common/sample_clock.h"
#include "driver/ni/daqmx/sugared.h"
#include "driver/ni/names.h"
#include "driver/ni/syscfg/sugared.h"
#include "driver/task/task.h"

namespace driver::ni {
const std::vector UNREACHABLE_ERRORS = {
    daqmx::DEVICE_DISCONNECTED,
    daqmx::RESOURCE_NOT_AVAILABLE,
    daqmx::DEVICE_DISCONNECTED_2,
    daqmx::ADC_CONVERSION_ERROR
};
const std::vector REQUIRES_RESTART_ERRORS = {
    daqmx::RESOURCE_RESERVED,
    daqmx::ROUTING_ERROR
};

inline x::errors::Error translate_error(const x::errors::Error &err) {
    if (!err) return err;
    LOG(WARNING) << "[ni] task encountered error: " << err;
    if (err.matches(UNREACHABLE_ERRORS)) return daqmx::TEMPORARILY_UNREACHABLE;
    if (err.matches(REQUIRES_RESTART_ERRORS)) return daqmx::REQUIRES_RESTART;
    if (err.matches(daqmx::APPLICATION_TOO_SLOW))
        return {x::errors::Error(
            errors::CRITICAL_HARDWARE_ERROR,
            "The network cannot keep up with the stream rate. Make the sample rate a "
            "higher multiple of the stream rate."
        )};
    return err.skip(daqmx::ANALOG_WRITE_OUT_OF_BOUNDS);
}

/// @brief a factory for instantiating and operating NI data acquisition, control,
/// and device scanning tasks.
class Factory final : public task::Factory {
    /// @brief the DAQmx library that read and write tasks use.
    std::shared_ptr<x::lib::Lazy<daqmx::SugaredAPI>> dmx;
    /// @brief the System Configuration library that the scanner uses.
    std::shared_ptr<x::lib::Lazy<syscfg::SugaredAPI>> syscfg;
    common::TimingConfig timing_cfg;

public:
    Factory(
        x::lib::Lazy<daqmx::SugaredAPI>::Loader load_dmx,
        x::lib::Lazy<syscfg::SugaredAPI>::Loader load_syscfg,
        common::TimingConfig timing_cfg = common::TimingConfig{}
    );

    /// @brief creates a new NI factory that loads the DAQmx and System Configuration
    /// libraries on first use.
    static std::unique_ptr<Factory>
    create(common::TimingConfig timing_cfg = common::TimingConfig{});

    /// @brief implements task::Factory to process task configuration requests.
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override;

    /// @brief implements task::Factory to configure initial tasks such as the
    /// device scanner.
    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
    ) override;

    template<typename HardwareT, typename ConfigT, typename SourceSinkT, typename TaskT>
    std::pair<common::ConfigureResult, x::errors::Error> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) {
        common::ConfigureResult result;
        auto [cfg, cfg_err] = ConfigT::parse(ctx->client, task, this->timing_cfg);
        if (cfg_err) return {std::move(result), cfg_err};
        result.auto_start = cfg.auto_start;
        auto [dmx, dmx_err] = this->dmx->get();
        if (dmx_err) return {std::move(result), dmx_err};
        const std::string dmx_task_name = task.name + " (" + task.key.to_string() + ")";
        using Hardware = SourceSinkT::Hardware;
        typename SourceSinkT::MakeHardware make_hw =
            [dmx, dmx_task_name](
                const ConfigT &run_cfg
            ) -> std::pair<std::unique_ptr<Hardware>, x::errors::Error> {
            TaskHandle handle = nullptr;
            if (const auto err = dmx->CreateTask(dmx_task_name.c_str(), &handle))
                return {nullptr, err};
            // The hardware owns the handle from here, so a failed apply still
            // clears it.
            auto hw = std::make_unique<HardwareT>(dmx, handle);
            if (const auto err = run_cfg.apply(dmx, handle)) return {nullptr, err};
            return {std::move(hw), x::errors::NIL};
        };
        result.task = std::make_unique<TaskT>(
            task,
            ctx,
            x::breaker::default_config(task.name),
            std::make_unique<SourceSinkT>(std::move(cfg), std::move(make_hw))
        );
        return {std::move(result), x::errors::NIL};
    }

    std::string name() override { return INTEGRATION_NAME; }

    std::pair<common::ConfigureResult, x::errors::Error> configure_scan(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    );
};
}
