// Copyright 2024 Synnax Labs, Inc.
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
#include <utility>
#include <memory>
#include <thread>

#include "nidaqmx/nidaqmx_api.h"
#include "nidaqmx/nidaqmx.h"
#include "nisyscfg/nisyscfg.h"
#include "nisyscfg/nisyscfg_api.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"

#include "driver/task/task.h"
#include "driver/breaker/breaker.h"
#include "driver/config/config.h"
#include "driver/errors/errors.h"
#include "driver/loop/loop.h"

namespace ni {
class Factory final : public task::Factory {
public:
    Factory(
        const std::shared_ptr<DAQmx> &dmx,
        const std::shared_ptr<SysCfg> &syscfg
    );

    bool check_health(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );


    static std::shared_ptr<ni::Factory> create();

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
    bool dlls_present = false;
    std::shared_ptr<DAQmx> dmx;
    std::shared_ptr<SysCfg> syscfg;
};

const std::string INTEGRATION_NAME = "ni";
} // namespace ni
