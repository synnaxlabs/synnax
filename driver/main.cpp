// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <csignal>
#include <fstream>
#include <iostream>
#include <thread>

#include "nlohmann/json.hpp"
#include "glog/logging.h"

#include "driver/driver.h"
#include "task/task.h"
#include "driver/opc/opc.h"
#include "driver/meminfo/meminfo.h"
#include "driver/ni/ni.h"

using json = nlohmann::json;

std::unique_ptr<driver::Driver> d;

std::pair<synnax::Rack, freighter::Error> retrieveDriverRack(
    const driver::Config &config,
    breaker::Breaker &breaker,
    const std::shared_ptr<synnax::Synnax> &client
) {
    std::pair<synnax::Rack, freighter::Error> res;
    if (config.rack_key != 0)
        res = client->hardware.retrieveRack(config.rack_key);
    else
        res = client->hardware.retrieveRack(config.rack_name);
    auto err = res.second;
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message()))
        return retrieveDriverRack(config, breaker, client);
    return res;
}

std::atomic<bool> stopped = false;

int main(int argc, char *argv[]) {
    FLAGS_logtostderr = 1;
    google::InitGoogleLogging(argv[0]);

    std::string config_path = "./synnax-driver-config.json";
    if (argc > 1) config_path = argv[1]; // Use the first argument as the config path if provided

    LOG(INFO) << "[main] starting up";

    auto cfg_json = driver::readConfig(config_path);
    if (cfg_json.empty())
        LOG(INFO) << "[main] no configuration found at " << config_path <<
                ". We'll just use the default configuration.";
    else {
        LOG(INFO) << "[main] loaded configuration from " << config_path;
    }
    auto [cfg, cfg_err] = driver::parseConfig(cfg_json);
    if (cfg_err) {
        LOG(FATAL) << "[main] failed to parse configuration: " << cfg_err;
        return 1;
    }
    LOG(INFO) << "[main] configuration parsed successfully";
    LOG(INFO) << "[main] connecting to Synnax at " << cfg.client_config.host << ":" <<
            cfg.client_config.port;


    auto client = std::make_shared<synnax::Synnax>(cfg.client_config);

    auto breaker = breaker::Breaker(cfg.breaker_config);
    breaker.start();
    LOG(INFO) << "[main] retrieving meta-data";
    auto [rack, rack_err] = retrieveDriverRack(cfg, breaker, client);
    breaker.stop();
    if (rack_err) {
        LOG(FATAL) <<
                "[main] failed to retrieve meta-data - can't proceed without it. Exiting."
                << rack_err;
        return 1;
    }

    std::unique_ptr<task::Factory> opc_factory = std::make_unique<opc::Factory>();
    std::unique_ptr<meminfo::Factory> meminfo_factory = std::make_unique<
        meminfo::Factory>();
    // std::unique_ptr<ni::Factory> ni_factory = std::make_unique<ni::Factory>();

    // std::vector<std::shared_ptr<task::Factory> > factories = {
    //     std::move(opc_factory), std::move(meminfo_factory), std::move(ni_factory)
    // };

    std::vector<std::shared_ptr<task::Factory> > factories = {
        std::move(opc_factory), std::move(meminfo_factory)
    };

    std::unique_ptr<task::Factory> factory = std::make_unique<task::MultiFactory>(
        std::move(factories)
    );

    d = std::make_unique<driver::Driver>(
        rack,
        client,
        std::move(factory),
        cfg.breaker_config
    );
    signal(SIGINT, [](int) {
        if (stopped) return;
        LOG(INFO) << "[main] received interrupt signal. shutting down";
        stopped = true;
        d->stop();
        LOG(INFO) << "[main] shutdown complete";
    });
    auto err = d->run();
    if (err)
        LOG(FATAL) << "[main] failed to start: " << err;
    return 0;
}
