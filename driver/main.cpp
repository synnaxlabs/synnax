// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif

#ifndef NOMINMAX
#define NOMINMAX
#endif

#include <winsock2.h>
#include <ws2tcpip.h>
#include <windows.h>
#endif

// Removed: #include <csignal>
#include <fstream>
#include <iostream>
#include <thread>
#include <condition_variable>
#include <mutex>

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

std::mutex mtx;
std::condition_variable cv;
bool should_stop = false;

std::pair<synnax::Rack, freighter::Error> retrieve_driver_rack(
    const configd::Config &config,
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
        return retrieve_driver_rack(config, breaker, client);
    return res;
}

const std::string STOP_COMMAND = "STOP";

void input_listener() {
    std::string input;
    while (std::getline(std::cin, input)) {
        if (input == STOP_COMMAND) {
            {
                std::lock_guard<std::mutex> lock(mtx);
                should_stop = true;
            }
            cv.notify_one();
            break;
        }
    }
}

int main(int argc, char *argv[]) {
    std::string config_path = "./synnax-driver-config.json";
    // Use the first argument as the config path if provided
    if (argc > 1) config_path = argv[1];

    // json cfg_json;
    LOG(INFO) << config_path;
    auto cfg_json = configd::read(config_path);
    if (cfg_json.empty())
        LOG(INFO) << "[driver] no configuration found at " << config_path <<
                ". We'll just use the default configuration";
    else {
        LOG(INFO) << "[driver] loaded configuration from " << config_path;
    }
    auto [cfg, cfg_err] = configd::parse(cfg_json);
    if (cfg_err) {
        LOG(FATAL) << "[driver] failed to parse configuration: " << cfg_err;
        return 1;
    }
    VLOG(1) << "[driver] configuration parsed successfully";

    LOG(INFO) << "[driver] starting up";
    FLAGS_logtostderr = 1;
    if (cfg.debug) FLAGS_v = 1;
    google::InitGoogleLogging(argv[0]);

    VLOG(1) << "[driver] connecting to Synnax at " << cfg.client_config.host << ":"
            << cfg.client_config.port;

    auto client = std::make_shared<synnax::Synnax>(cfg.client_config);

    auto breaker = breaker::Breaker(cfg.breaker_config);
    breaker.start();
    VLOG(1) << "[driver] retrieving metadata";
    auto [rack, rack_err] = retrieve_driver_rack(cfg, breaker, client);
    breaker.stop();
    if (rack_err) {
        LOG(FATAL) << "[driver] failed to retrieve metadata - can't proceed without it. Exiting."
                << rack_err;
        return 1;
    }

    // auto meminfo_factory = std::make_unique<meminfo::Factory>();
    auto heartbeat_factory = std::make_unique<heartbeat::Factory>();

    std::vector<std::shared_ptr<task::Factory> > factories = {
        // std::move(meminfo_factory),
        std::move(heartbeat_factory)
    };

    auto opc_enabled = std::find(cfg.integrations.begin(), cfg.integrations.end(),
                                 opc::INTEGRATION_NAME);
    if (opc_enabled != cfg.integrations.end()) {
        auto opc_factory = std::make_unique<opc::Factory>();
        factories.push_back(std::move(opc_factory));
    } else
        LOG(INFO) << "[driver] OPC integration is not enabled";

#ifdef USE_NI

    auto ni_enabled = std::find(
        cfg.integrations.begin(),
        cfg.integrations.end(),
        ni::INTEGRATION_NAME
    );

    if (ni_enabled != cfg.integrations.end() && ni::dlls_available()) {
        std::unique_ptr<ni::Factory> ni_factory = std::make_unique<ni::Factory>();
        factories.push_back(std::move(ni_factory));
    } else
        LOG(INFO)
            << "[driver] NI integration is not enabled or the required DLLs are not available";

#endif

#ifdef _WIN32
    auto labjack_enabled = std::find(
        cfg.integrations.begin(),
        cfg.integrations.end(),
        labjack::INTEGRATION_NAME
    );

    if (labjack_enabled != cfg.integrations.end() && labjack::dlls_available()) {
        std::unique_ptr<labjack::Factory> labjack_factory = std::make_unique<labjack::Factory>();
        factories.push_back(std::move(labjack_factory));
    } else {
        LOG(INFO) << "[driver] LabJack integration is not enabled or the required DLLs are not available";
    }
#else
    LOG(INFO) << "[driver] LabJack integration is not available on this platform";
#endif

    LOG(INFO) << "[driver] starting task manager";

    auto factory = std::make_unique<task::MultiFactory>(std::move(factories));
    auto task_manager = std::make_unique<task::Manager>(
        rack,
        client,
        std::move(factory),
        cfg.breaker_config
    );

    std::thread listener(input_listener);

    auto err = task_manager->start();
    if (err) {
        LOG(FATAL) << "[driver] failed to start: " << err;
        return 1;
    } {
        std::unique_lock<std::mutex> lock(mtx);
        cv.wait(lock, [] { return should_stop; });
    }

    LOG(INFO) << "[driver] received stop command. Shutting down";
    task_manager->stop();
    listener.join();
    LOG(INFO) << "[driver] shutdown complete";
    return 0;
}
