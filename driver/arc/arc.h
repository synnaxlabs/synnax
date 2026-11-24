// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <string>
#include <utility>
#include <vector>

#include "client/cpp/synnax.h"
#include "x/cpp/telem/telem.h"
#include "x/cpp/thread/rt/rt.h"

#include "driver/common/common.h"
#include "driver/task/task.h"

namespace driver::arc {
/// @brief integration name for arc runtime.
const std::string INTEGRATION_NAME = "arc";
/// @brief task type for arc runtime tasks.
const std::string TASK_TYPE = "arc";

/// @brief the Driver settings of the Arc integration.
struct Config {
    /// @brief how long a task waits after a calculated channel changes before it
    /// loads the calculations again, so that changes close in time load once.
    x::telem::TimeSpan calculation_reload_delay = 100 * x::telem::MILLISECOND;

    template<typename Parser>
    void override(Parser &p) {
        const auto delay_s = p.field(
            "calculation_reload_delay",
            this->calculation_reload_delay.seconds()
        );
        this->calculation_reload_delay = x::telem::TimeSpan(
            static_cast<int64_t>(delay_s * 1e9)
        );
    }
};

/// @brief factory for creating arc runtime tasks.
class Factory final : public task::Factory {
    std::shared_ptr<x::thread::rt::Manager> rt_manager;
    Config config;

    /// @brief configures an arc runtime task.
    std::pair<common::ConfigureResult, x::errors::Error> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    );

public:
    explicit Factory(
        std::shared_ptr<x::thread::rt::Manager> rt_manager = nullptr,
        Config config = {}
    ):
        rt_manager(std::move(rt_manager)), config(config) {}

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task,
        const std::string &cmd_key,
        x::breaker::Breaker &cancel
    ) override;

    [[nodiscard]] std::string name() override;
};
}
