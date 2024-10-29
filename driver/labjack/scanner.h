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
#include <map>
#include <thread>

#include <stdio.h>
#include "LJM_Utilities.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"

#include "driver/errors/errors.h"
#include "driver/breaker/breaker.h"
#include "driver/task/task.h"

// Currently supports: T7, T4, T5, Digit products.

namespace labjack{

const std::string SCAN_CMD_TYPE = "scan";
const std::string STOP_CMD_TYPE = "stop";

///////////////////////////////////////////////////////////////////////////////////
//                               Scanner Task                                    //
///////////////////////////////////////////////////////////////////////////////////
class ScannerTask final : public task::Task {
public:
    explicit ScannerTask() = default;
    ~ScannerTask();
    explicit ScannerTask (
            const std::shared_ptr<task::Context> &ctx,
            const synnax::Task &task
    );

    static std::unique_ptr<task::Task> configure(
            const std::shared_ptr<task::Context> &ctx,
            const synnax::Task &task
    );

    std::string name() override { return task.name; }

    void exec(task::Command &cmd) override;

    void stop() override;

    void scan();

    void run();

    void create_devices();

    json get_devices();

    int check_err(int err);

private:
    json devices;
    std::set<int> device_keys;
    synnax::Task task;
    std::shared_ptr<task::Context> ctx;
    std::shared_ptr<std::thread> thread = nullptr;
    breaker::Breaker breaker;
    synnax::Rate scan_rate = synnax::Rate(5);
};
};
