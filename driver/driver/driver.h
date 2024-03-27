// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <string>
#include <unordered_map>
#include <thread>
#include <latch>

#include "client/cpp/synnax/synnax.h"
#include "driver/driver/breaker/breaker.h"
#include "nlohmann/json.hpp"

#pragma once

using json = nlohmann::json;

namespace driver {
class Task {
private:
    synnax::Task internal;
public:
    Task(synnax::Task task) : internal(std::move(task)) {}

    Task() = default;

    virtual void start() {};

    virtual void stop() {};

    virtual ~Task() = default;
};

class TaskFactory {
public:
    virtual std::unique_ptr<Task> createTask(const std::shared_ptr<synnax::Synnax> &client,
                                               const synnax::Task &task,
                                               bool &valid_config,
                                               json &config_err) = 0;

    virtual ~TaskFactory() = default;
};

class TaskManager {
public:
    [[maybe_unused]] TaskManager(
            synnax::RackKey rack_key,
            const std::shared_ptr<synnax::Synnax> &client,
            std::unique_ptr<TaskFactory> factory,
            breaker::Breaker breaker
    );

    freighter::Error start(std::latch &latch);

    freighter::Error stop();

    void processTaskSet(const synnax::Series &series, synnax::Writer &comms);

    void processModuleDelete(const synnax::Series &series);

private:
    RackKey rack_key;
    Rack internal;

    const std::shared_ptr<synnax::Synnax> client;
    std::unique_ptr<TaskFactory> factory;
    std::unique_ptr<Streamer> streamer;


    std::unordered_map<std::uint64_t, std::unique_ptr<Task>> tasks;

    Channel task_set_channel;
    Channel task_delete_channel;
    Channel task_state_channel;

    std::thread exec_thread;
    freighter::Error exit_err;
    breaker::Breaker breaker;

    void run(std::latch &latch);

    freighter::Error runInternal();

    freighter::Error startInternal();

};

class Heartbeat {
public:
    Heartbeat(
            synnax::RackKey rack_key,
            std::uint32_t generation,
            std::shared_ptr<synnax::Synnax> client,
            breaker::Breaker breaker
    );

    freighter::Error start(std::latch &latch);

    freighter::Error stop();

private:
    // Synnax
    synnax::RackKey rack_key;
    const std::shared_ptr<synnax::Synnax> client;
    synnax::Channel rack_heartbeat_channel;

    // Heartbeat
    std::uint32_t generation;
    std::uint32_t version;

    // Breaker
    breaker::Breaker breaker;

    // Threading
    std::atomic<bool> running;
    freighter::Error exit_err;
    std::thread exec_thread;

    void run();
};

class Rack {
public:
    Rack(
            synnax::RackKey key,
            std::uint32_t generation,
            const std::shared_ptr<synnax::Synnax> &client,
            std::unique_ptr<TaskFactory> module_factory,
            breaker::Breaker breaker
    );

    freighter::Error run();

private:
    synnax::RackKey key;
    TaskManager modules;
    Heartbeat heartbeat;
};
}