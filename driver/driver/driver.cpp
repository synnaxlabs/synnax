// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <fstream>
#include <latch>
#include <glog/logging.h>
#include "nlohmann/json.hpp"
#include "driver/driver/driver.h"

using json = nlohmann::json;

driver::Driver::Driver(
    Rack rack,
    const std::shared_ptr<Synnax> &client,
    std::unique_ptr<task::Factory> factory,
    breaker::Config breaker_config
): key(rack.key),
   task_manager(rack, client, std::move(factory), breaker_config.child("task_manager")),
   heartbeat(rack.key, client, breaker_config.child("heartbeat")) {
}

const std::string VERSION = "0.1.0";

freighter::Error driver::Driver::run() {
    std::atomic done = false;
    auto err = task_manager.start(done);
    if (err) return err;
    err = heartbeat.start(done);
    if (err) {
        task_manager.stop();
        return err;
    }
    LOG(INFO) << "[Driver] started successfully. waiting for shutdown.";
    done.wait(false);
    task_manager.stop();
    heartbeat.stop();
    return freighter::NIL;
}

void driver::Driver::stop() {
    const auto tm_err = task_manager.stop();
    const auto hb_err = heartbeat.stop();
    if (tm_err) {
        LOG(ERROR) << "Failed to stop task manager: " << tm_err.message();
    }
    if (hb_err) {
        LOG(ERROR) << "Failed to stop heartbeat: " << hb_err.message();
    }
}
