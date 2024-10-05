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
#include "LabJackM.h"
#include "LJM_Utilities.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"

#include "driver/errors/errors.h"
#include "driver/breaker/breaker.h"
#include "driver/task/task.h"

namespace labjack{

const std::string SCAN_CMD_TYPE = "scan";

///////////////////////////////////////////////////////////////////////////////////
//                               Scanner Task                                    //
///////////////////////////////////////////////////////////////////////////////////
class ScannerTask final : public task::Task {
public:
    explicit ScannerTask() = default;
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

    void stop() override{

    };

    void scan();

private:
    json devices;
    std::set<std::string> device_keys;
    synnax::Task task;
    std::shared_ptr<task::Context> ctx;
    std::shared_ptr<std::thread> scan_thread = nullptr;
};


};

/*
 Supported device types
 T4
 T7
 T8
 DIGIT (thought it is eol)
 */