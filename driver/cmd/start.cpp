// Copyright 2025 Synnax Labs, Inc.
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
#include <filesystem>

/// external
#include "nlohmann/json.hpp"
#include "glog/logging.h"

/// internal
#include "cmd.h"
#include "driver/task/task.h"
#include "driver/opc/opc.h"
#include "driver/heartbeat/heartbeat.h"
#include "driver/ni/ni.h"
#include "driver/sequence/sequence.h"
#include "driver/config/config.h"

const std::string STOP_COMMAND = "STOP";

std::mutex mtx;
std::condition_variable cv;
bool should_stop = false;

typedef std::vector<std::unique_ptr<task::Factory> > FactoryList;

namespace fs = std::filesystem;


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

void configure_opc(const driver::Config &config, FactoryList &factories) {
    if (!config.integration_enabled(opc::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] OPC integration disabled";
        return;
    }
    factories.push_back(std::make_unique<opc::Factory>());
}

void configure_ni(const driver::Config &config, FactoryList &factories) {
    if (!config.integration_enabled(ni::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] NI integration disabled";
        return;
    }
    factories.push_back(ni::Factory::create());
}

void configure_sequences(const driver::Config &config, FactoryList &factories) {
    if (!config.integration_enabled(sequence::INTEGRATION_NAME)) {
        LOG(INFO) << "[driver] Sequence integration disabled";
        return;
    }
    factories.push_back(std::make_unique<sequence::Factory>());
}

void configure_labjack(const driver::Config &config, FactoryList &factories) {
#ifdef _WIN32
    if (
        !config.integration_enabled(labjack::INTEGRATION_NAME) ||
        !labjack::dlls_available()
    ) {
        LOG(INFO) << "[driver] LabJack integration disabled";
        return;
    }
    factories.push_back(std::make_unique<labjack::Factory>());
    return;
#endif
    LOG(INFO) << "[driver] LabJack integration not available on this platform";
}

int cmd::sub::start(int argc, char *argv[]) {
    const auto [cfg, cfg_err] = driver::Config::load(argc, argv);
    if (cfg_err) {
        LOG(FATAL) << cfg_err;
        return 1;
    }

    auto hb_factory = std::make_unique<heartbeat::Factory>();
    FactoryList factories{};
    configure_opc(cfg, factories);
    configure_ni(cfg, factories);
    configure_sequences(cfg, factories);
    configure_labjack(cfg, factories);

    LOG(INFO) << "[driver] starting task manager";

    const auto client = std::make_shared<synnax::Synnax>(cfg.connection);
    auto factory = std::make_unique<task::MultiFactory>(std::move(factories));
    task::Manager task_manager(
        cfg.rack_key,
        cfg.cluster_key,
        [](const synnax::RackKey &rack, const std::string &cluster_key) {
            return driver::save_remote_info(rack, cluster_key);
        },
        client,
        std::move(factory),
        cfg.breaker_config
    );
    std::thread listener(input_listener);
    if (auto err = task_manager.start()) {
        LOG(FATAL) << "[driver] failed to start: " << err;
        return 1;
    } {
        std::unique_lock lock(mtx);
        cv.wait(lock, [] { return should_stop; });
    }
    LOG(INFO) << "[driver] received stop command. Shutting down";
    task_manager.stop();
    listener.join();
    LOG(INFO) << "[driver] shutdown complete";
    return 0;
}
