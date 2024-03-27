// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


/// std.
#include <fstream>
#include <latch>

// external.
#include <glog/logging.h>
#include "nlohmann/json.hpp"

// internal.
#include "driver/driver/driver.h"

using json = nlohmann::json;

driver::Driver::Driver(
    RackKey key,
    const std::shared_ptr<Synnax>& client,
    std::unique_ptr<TaskFactory> factory,
    const breaker::Breaker& brk
): task_manager(key, client, std::move(factory), brk), heartbeat(key, client, brk) {
}

freighter::Error driver::Driver::run() {
    LOG(INFO) << "Starting Node " << key.node_key() << "Rack " << key.value;
    std::latch rack_latch{1};
    LOG(INFO) << "Starting task manager";
    auto err = task_manager.start(rack_latch);
    if (err) {
        LOG(ERROR) << "Failed to start task manager: " << err.message();
        return err;
    }
    LOG(INFO) << "Task manager started successfully. Starting heartbeat.";
    err = heartbeat.start(rack_latch);
    if (err) {
        LOG(ERROR) << "Failed to start heartbeat: " << err.message();
        task_manager.stop();
        return err;
    }
    LOG(INFO) << "Rack started successfully. Waiting for shutdown.";
    auto modules_err = task_manager.stop();
    auto hb_err = heartbeat.stop();
    rack_latch.wait();
    if (modules_err) return modules_err;
    return hb_err;
}