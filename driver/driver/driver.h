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
#include <latch>
#include <memory>
#include "nlohmann/json.hpp"
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/breaker/breaker.h"
#include "task/task.h"

using json = nlohmann::json;

namespace driver {
/// @brief TaskManager is responsible for configuring, executing, and commanding data
/// acqusition and control tasks.
class TaskManager {
public:
    TaskManager(
        Rack rack,
        const std::shared_ptr<Synnax> &client,
        std::unique_ptr<task::Factory> factory,
        breaker::Config breaker
    );

    freighter::Error start(std::atomic<bool> &done);

    freighter::Error stop();

private:
    RackKey rack_key;
    Rack internal;
    std::shared_ptr<task::Context> ctx;
    std::unique_ptr<Streamer> streamer;
    std::unique_ptr<task::Factory> factory;
    std::unordered_map<std::uint64_t, std::unique_ptr<task::Task> > tasks{};

    Channel task_set_channel;
    Channel task_delete_channel;
    Channel task_cmd_channel;
    Channel task_state_channel;

    breaker::Breaker breaker;

    std::thread run_thread;
    freighter::Error run_err;

    void run(std::atomic<bool> &done);

    freighter::Error runGuarded();

    freighter::Error startGuarded();

    void processTaskSet(const Series &series);

    void processTaskDelete(const Series &series);

    void processTaskCmd(const Series &series);
};

class Heartbeat {
public:
    Heartbeat(
        RackKey rack_key,
        std::shared_ptr<Synnax> client,
        breaker::Config breaker_config
    );

    freighter::Error start(std::atomic<bool> &done);

    freighter::Error stop();

private:
    // Synnax
    RackKey rack_key;
    const std::shared_ptr<Synnax> client;

    Channel channel;

    // Heartbeat
    std::uint32_t version;

    // Breaker
    breaker::Breaker breaker;

    // Threading
    std::atomic<bool> running;
    std::thread run_thread;
    freighter::Error run_err;

    void run(std::atomic<bool> &done);

    freighter::Error runGuarded();

    freighter::Error startGuarded();
};

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
        breaker::Config breaker_config
    );

    freighter::Error run();

    void stop();

private:
    RackKey key{};
    TaskManager task_manager;
    Heartbeat heartbeat;
};
}
