// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// Windows-specific headers
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

/// LabJack only supported on Windows.
#include "driver/labjack/labjack.h"
#endif

/// std
#include <fstream>
#include <iostream>
#include <thread>
#include <condition_variable>
#include <mutex>
#include <array>

/// external
#include "nlohmann/json.hpp"
#include "glog/logging.h"

/// internal
#include "driver/config.h"
#include "driver/task/task.h"
#include "driver/opc/opc.h"
#include "driver/meminfo/meminfo.h"
#include "driver/heartbeat/heartbeat.h"
#include "driver/ni/ni.h"
#include "driver/sequence/task.h"

using json = nlohmann::json;

std::mutex mtx;
std::condition_variable cv;
bool should_stop = false;


std::string get_hostname() {
    std::array<char, 256> hostname{};
#ifdef _WIN32
    DWORD size = hostname.size();
    if (GetComputerNameA(hostname.data(), &size) == 0) {
        LOG(WARNING) << "[driver] Failed to get hostname";
        return "unknown";
    }
#else
    if (gethostname(hostname.data(), hostname.size()) != 0) {
        LOG(WARNING) << "[driver] Failed to get hostname";
        return "unknown";
    }
#endif
    return {hostname.data()};
}

std::pair<synnax::Rack, freighter::Error> retrieve_driver_rack(
    configd::Config &config,
    breaker::Breaker &breaker,
    const std::shared_ptr<synnax::Synnax> &client) {
    std::pair<synnax::Rack, freighter::Error> res;
    if (config.rack_key != 0) {
        LOG(INFO) << "existing rack key found in configuration: " << config.rack_key;
        res = client->hardware.retrieve_rack(config.rack_key);
    } else {
        LOG(INFO) << "no existing rack key found in configuration. Creating a new rack";
        res = client->hardware.create_rack(get_hostname());
    }
    const auto err = res.second;
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message()))
        return retrieve_driver_rack(config, breaker, client);
    if (err.matches(synnax::NOT_FOUND)) {
        config.rack_key = 0;
        return retrieve_driver_rack(config, breaker, client);
    }
    LOG(INFO) << "[driver] retrieved rack: " << res.first.key << " - " << res.first.name;
    return res;
}

const std::string STOP_COMMAND = "STOP";

void input_listener() {
    std::string input;
    while (std::getline(std::cin, input)) {
        if (input == STOP_COMMAND) {
            {
                std::lock_guard lock(mtx);
                should_stop = true;
            }
            cv.notify_one();
            break;
        }
    }
}

void configure_opc(
    const configd::Config &config,
    std::vector<std::shared_ptr<task::Factory> > &factories) {
    if (!config.integration_enabled(opc::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] OPC integration disabled";
        return;
    }
    factories.push_back(std::make_shared<opc::Factory>());
}

void configure_ni(
    const configd::Config &config,
    std::vector<std::shared_ptr<task::Factory> > &factories) {
    if (!config.integration_enabled(ni::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] NI integration disabled";
        return;
    }
    const auto ni_factory = ni::Factory::create();
    factories.push_back(ni_factory);
}

void configure_sequences(
    const configd::Config &config,
    std::vector<std::shared_ptr<task::Factory> > &factories) {
    if (!config.integration_enabled(sequence::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] Sequence integration disabled";
        return;
    }
    factories.push_back(std::make_shared<sequence::Factory>());
}

void configure_labjack(
    const configd::Config &config,
    std::vector<std::shared_ptr<task::Factory> > &factories
) {
#ifdef _WIN32
    if (
        !config.integration_enabled(labjack::INTEGRATION_NAME) ||
        !labjack::dlls_available()
    ) {
        LOG(INFO) << "[driver] LabJack integration disabled";
        return;
    }
    auto labjack_factory = std::make_shared<labjack::Factory>();
    factories.push_back(labjack_factory);
    return;
#endif
    LOG(INFO) << "[driver] LabJack integration not available on this platform";
}


int main(int argc, char *argv[]) {
    std::string config_path = "./synnax-driver-config.json";
    if (argc > 1)
        config_path = argv[1];

    auto cfg_json = configd::read(config_path);
    LOG(INFO) << "[driver] reading configuration from " << config_path;
    if (cfg_json.empty())
        LOG(INFO) << "[driver] no configuration found at " << config_path <<
                ". We'll just use the default configuration";
    else
        LOG(INFO) << "[driver] loaded configuration from " << config_path;
    auto [cfg, cfg_err] = configd::parse(cfg_json);
    if (cfg_err) {
        LOG(FATAL) << "[driver] failed to parse configuration: " << cfg_err;
        return 1;
    }
    VLOG(1) << "[driver] configuration parsed successfully";

    auto [persisted_state, state_err] = configd::load_persisted_state();
    if (state_err) {
        LOG(WARNING) << "[driver] failed to load persisted state: " << state_err;
    } else if (persisted_state.rack_key != 0 && cfg.rack_key == 0) {
        VLOG(1) << "[driver] using persisted rack key: " << persisted_state.rack_key;
        cfg.rack_key = persisted_state.rack_key;
    }

    LOG(INFO) << "[driver] starting up";

    FLAGS_logtostderr = true;
    if (cfg.debug)
        FLAGS_v = 1;
    google::InitGoogleLogging(argv[0]);

    VLOG(1) << "[driver] connecting to Synnax at " << cfg.client_config.host << ":"
            << cfg.client_config.port;

    auto client = std::make_shared<synnax::Synnax>(cfg.client_config);

    auto breaker = breaker::Breaker(cfg.breaker_config);
    breaker.start();
    VLOG(1) << "[driver] retrieving meta-data";
    auto [rack, rack_err] = retrieve_driver_rack(cfg, breaker, client);
    breaker.stop();
    if (rack_err) {
        LOG(FATAL) <<
                "[driver] failed to retrieve meta-data - can't proceed without it. Exiting."
                << rack_err;
        return 1;
    }

    if (auto err = configd::save_persisted_state({.rack_key = rack.key}))
        LOG(WARNING) << "[driver] failed to save persisted state: " << err;

    auto hb_factory = std::make_shared<heartbeat::Factory>();
    std::vector<std::shared_ptr<task::Factory> > factories{hb_factory};
    configure_opc(cfg, factories);
    configure_ni(cfg, factories);
    configure_sequences(cfg, factories);
    configure_labjack(cfg, factories);

    LOG(INFO) << "[driver] starting task manager";

    auto factory = std::make_unique<task::MultiFactory>(std::move(factories));
    auto task_manager = std::make_unique<task::Manager>(
        rack,
        client,
        std::move(factory),
        cfg.breaker_config);

    std::thread listener(input_listener);

    if (auto err = task_manager->start()) {
        LOG(FATAL) << "[driver] failed to start: " << err;
        return 1;
    } {
        std::unique_lock lock(mtx);
        cv.wait(lock, [] { return should_stop; });
    }

    LOG(INFO) << "[driver] received stop command. Shutting down";
    task_manager->stop();
    listener.join();
    LOG(INFO) << "[driver] shutdown complete";
    return 0;
}
