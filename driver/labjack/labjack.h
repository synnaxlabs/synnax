// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/task/task.h"
#include "driver/labjack/dll_check_windows.h"
#include "driver/labjack/util.h"

namespace labjack {
const std::string INTEGRATION_NAME = "labjack";

class Factory final : public task::Factory {
public:
    Factory() = default;

    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override;

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override;

private:
    std::shared_ptr<labjack::DeviceManager> device_manager = std::make_shared<labjack::DeviceManager>();
};

static inline bool dlls_available() {
    std::vector<std::string> dlls = {
        "LabjackM.dll",
        "LabJackWUSB.dll",
    };

    bool all_present = true;
    for (const auto &dll: dlls)
        if (!labjack::does_dll_exist(dll.c_str()))
            all_present = false;

    if (!all_present)
        LOG(ERROR) << "[labjack] Required Labjack DLLs not found.";

    return all_present;
} // dlls_available
} // namespace labjack
