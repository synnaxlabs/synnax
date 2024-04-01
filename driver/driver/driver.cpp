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
    const RackKey key,
    const std::shared_ptr<Synnax>& client,
    std::unique_ptr<task::Factory> factory,
    const breaker::Breaker& brk
): key(key), task_manager(key, client, std::move(factory), brk), heartbeat(key, client, brk) {
}

void driver::Driver::run() {
    LOG(ERROR) << "Starting Rack " << key;
    std::latch rack_latch{2};
    LOG(ERROR) << "Starting task manager";
    auto err = task_manager.start(rack_latch);
    if (err) {
        LOG(ERROR) << "Failed to start task manager: " << err.message();
        return;
    }
    LOG(INFO) << "Task manager started successfully. Starting heartbeat.";
    err = heartbeat.start(rack_latch);
    if (err) {
        LOG(ERROR) << "Failed to start heartbeat: " << err.message();
        task_manager.stop();
        return;
    }
    LOG(INFO) << "Rack started successfully. Waiting for shutdown.";
    rack_latch.wait();
}

void driver::Driver::stop() {
    auto modules_err = task_manager.stop();
    auto hb_err = heartbeat.stop();
    if (modules_err) {
        LOG(ERROR) << "Failed to stop task manager: " << modules_err.message();
    }
    if (hb_err) {
        LOG(ERROR) << "Failed to stop heartbeat: " << hb_err.message();
    }
}