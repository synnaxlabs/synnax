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

#include "driver/config.h"
#include "task/task.h"
#include "driver/opc/opc.h"
#include "driver/meminfo/meminfo.h"
#include "driver/heartbeat/heartbeat.h"
#include "driver/ni/ni.h"
#ifdef _WIN32
#include "driver/labjack/labjack.h"
#endif

using json = nlohmann::json;

std::unique_ptr<task::Manager> task_manager;

std::pair<synnax::Rack, freighter::Error> retrieveDriverRack(
    const config::Config &config,
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
    std::string config_path = "./synnax-driver-config.json";
    // Use the first argument as the config path if provided
    if (argc > 1) config_path = argv[1];

    auto cfg_json = config::read(config_path);
    if (cfg_json.empty())
        LOG(INFO) << "[driver] no configuration found at " << config_path <<
                ". We'll just use the default configuration";
    else {
        LOG(INFO) << "[driver] loaded configuration from " << config_path;
    }
    auto [cfg, cfg_err] = config::parse(cfg_json);
    if (cfg_err) {
        LOG(FATAL) << "[driver] failed to parse configuration: " << cfg_err;
        return 1;
    }
    VLOG(1) << "[driver] configuration parsed successfully";

    LOG(INFO) << "[driver] starting up";
    FLAGS_logtostderr = 1;
    if (cfg.debug) FLAGS_v = 1;
    google::InitGoogleLogging(argv[0]);

    VLOG(1) << "[driver] connecting to Synnax at " << cfg.client_config.host << ":" <<
            cfg.client_config.port;

    auto client = std::make_shared<synnax::Synnax>(cfg.client_config);

    auto breaker = breaker::Breaker(cfg.breaker_config);
    breaker.start();
    VLOG(1) << "[driver] retrieving meta-data";
    auto [rack, rack_err] = retrieveDriverRack(cfg, breaker, client);
    breaker.stop();
    if (rack_err) {
        LOG(FATAL) <<
                "[driver] failed to retrieve meta-data - can't proceed without it. Exiting."
                << rack_err;
        return 1;
    }

    // auto meminfo_factory = std::make_unique<meminfo::Factory>();
    auto heartbeat_factory = std::make_unique<heartbeat::Factory>();

    std::vector<std::shared_ptr<task::Factory> > factories = {
        // std::move(meminfo_factory),
        std::move(heartbeat_factory)
    };

    auto opc_enabled = std::find(cfg.integrations.begin(), cfg.integrations.end(), opc::INTEGRATION_NAME);
    if (opc_enabled != cfg.integrations.end()) {
        auto opc_factory = std::make_unique<opc::Factory>();
        factories.push_back(std::move(opc_factory));
    } else
        LOG(INFO) << "[driver] OPC integration is not enabled";

#ifdef USE_NI
    auto ni_enabled = std::find(cfg.integrations.begin(), cfg.integrations.end(), ni::INTEGRATION_NAME);
    if( ni_enabled != cfg.integrations.end() && ni::dlls_available()) {
        std::unique_ptr<ni::Factory>  ni_factory = std::make_unique<ni::Factory>();
        factories.push_back(std::move(ni_factory));
    } else
        LOG(INFO) << "[driver] NI integration is not enabled or the required DLLs are not available";
#endif

#ifdef _WIN32
    auto labjack_enabled = std::find(cfg.integrations.begin(), cfg.integrations.end(), labjack::INTEGRATION_NAME);
    if(labjack_enabled != cfg.integrations.end() && labjack::dlls_available()) {
        std::unique_ptr<labjack::Factory> labjack_factory = std::make_unique<labjack::Factory>();
        factories.push_back(std::move(labjack_factory));
    } else{
        LOG(INFO) << "[driver] LabJack integration is not enabled or the required DLLs are not available";
    }
#else
    LOG(INFO) << "[driver] LabJack integration is not available on this platform";
#endif



    auto factory = std::make_unique<task::MultiFactory>(std::move(factories));
    task_manager = std::make_unique<task::Manager>(
        rack,
        client,
        std::move(factory),
        cfg.breaker_config
    );
    signal(SIGINT, [](int) {
        if (stopped) return;
        LOG(INFO) << "[driver] received interrupt signal. shutting down";
        stopped = true;
        task_manager->stop();
    });
    std::atomic<bool> running = false;
    auto err = task_manager->start(running);
    running.wait(false);
    if (err)
        LOG(FATAL) << "[driver] failed to start: " << err;
    LOG(INFO) << "[driver] shutdown complete";
    return 0;
}
