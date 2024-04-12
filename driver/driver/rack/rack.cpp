// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <fstream>
#include <vector>
#include <latch>
#include <glog/logging.h>

#include "nlohmann/json.hpp"
#include "driver/rack/rack.h"

using json = nlohmann::json;

device::Rack::Rack(
    synnax::RackKey key,
    std::uint32_t generation,
    const std::shared_ptr<synnax::Synnax>& client,
    std::unique_ptr<module::Factory> factory,
    breaker::Breaker breaker
): modules(key, client, std::move(factory), breaker), heartbeat(key, generation, client, breaker) {
}

freighter::Error device::Rack::run() {
    LOG(INFO) << "Starting Node " << key.node_key() << "Rack " << key.value;
    std::latch rack_latch{1};
    LOG(INFO) << "Starting modules";
    auto err = modules.start(rack_latch);
    if (err) {
        LOG(ERROR) << "Failed to start modules: " << err.message();
        return err;
    }
    LOG(INFO) << "Modules started successfully. Starting heartbeat.";
    err = heartbeat.start(rack_latch);
    if (err) {
        LOG(ERROR) << "Failed to start heartbeat: " << err.message();
        modules.stop();
        return err;
    }
    LOG(INFO) << "Rack started successfully. Waiting for shutdown.";
    rack_latch.wait();
    auto modules_err = modules.stop();
    auto hb_err = heartbeat.stop();
    if (modules_err) return modules_err;
    return hb_err;
}