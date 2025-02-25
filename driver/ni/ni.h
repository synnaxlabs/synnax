// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>
#include <thread>


#include "client/cpp/synnax.h"
#include "daqmx/sugared.h"

#include "driver/ni/syscfg/syscfg.h"
#include "driver/ni/daqmx/daqmx.h"

#include "driver/task/task.h"

namespace ni {
class Factory final : public task::Factory {
    /// @brief the daqmx library used to communicate with NI hardware.
    std::shared_ptr<SugaredDAQmx> dmx;
    /// @brief the system configuration library used to get information
    /// about devices.
    std::shared_ptr<SysCfg> syscfg;

    [[nodiscard]] bool check_health(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) const;
public:
    Factory(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        const std::shared_ptr<SysCfg> &syscfg
    );

    /// @brief creates a new NI factory, loading the DAQmx and system configuration
    /// libraries.
    static std::unique_ptr<Factory> create();

    /// @brief implements task::Factory to process task configuration requests.
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    /// @brief implements task::Factory to configure initial tasks such as the
    /// device scanner.
    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;
};

const std::string INTEGRATION_NAME = "ni";
} // namespace ni
