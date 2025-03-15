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

#include "driver/labjack/device/device.h"
#include "driver/task/task.h"
#include "x/cpp/xjson/xjson.h"

namespace modbus {


const std::string TEST_CONNECTION_CMD_TYPE = "test_connection";

class ScanTask final: public task::Task {
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    std::shared_ptr<device::Manager> devices;

    void test_connection(const task::Command &cmd) const;
public:
    explicit ScanTask(
        const std::shared_ptr<task::Context> &context,
        const synnax::Task &task,
        const std::shared_ptr<device::Manager> &devices
    ): ctx(context), task(task), devices(devices) {
    }

    void exec(task::Command &cmd) override {
        this->devices->acquire()

    }

    std::string name() override { return this->task.name; }
};
}
