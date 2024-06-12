// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <unordered_map>
#include <thread>
#include "nlohmann/json.hpp"
#include "client/cpp/synnax.h"
#include "driver/breaker/breaker.h"
#include "heartbeat/heartbeat.h"
#include "task/task.h"

using json = nlohmann::json;

namespace driver {
struct Config {
    synnax::RackKey rack_key;
    std::string rack_name;
    synnax::Config client_config;
    breaker::Config breaker_config;
    std::vector<std::string> integrations;
};

std::pair<Config, freighter::Error> parseConfig(const json &content);

json readConfig(std::string path);

class Driver {
public:
    Driver(
        Rack rack,
        const std::shared_ptr<Synnax> &client,
        std::unique_ptr<task::Factory> task_factory,
        const breaker::Config &breaker_config
    );

    freighter::Error run();

    void stop();
private:
    task::Manager task_manager;
    heartbeat::Heartbeat heartbeat;
    std::atomic<bool> done = false;
};
}
