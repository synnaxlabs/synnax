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
        RackKey rack_key,
        const std::shared_ptr<Synnax> &client,
        std::unique_ptr<task::Factory> factory,
        breaker::Breaker breaker
    );

    freighter::Error start(std::latch &latch);

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
    void run(std::latch &latch);
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
        breaker::Breaker breaker
    );

    freighter::Error start(std::latch &latch);

    freighter::Error stop();

private:
    // Synnax
    RackKey rack_key;
    const std::shared_ptr<Synnax> client;

    Channel channel;

    // Heartbeat
    std::uint32_t generation;
    std::uint32_t version;

    // Breaker
    breaker::Breaker breaker;

    // Threading
    std::atomic<bool> running;
    std::thread exec_thread;

    void run();
};

class Driver {
public:
    Driver(
        RackKey key,
        const std::shared_ptr<Synnax> &client,
        std::unique_ptr<task::Factory> task_factory,
        const breaker::Breaker &brk
    );

    void run();

    void stop();

private:
    RackKey key{};
    TaskManager task_manager;
    Heartbeat heartbeat;
};
}
