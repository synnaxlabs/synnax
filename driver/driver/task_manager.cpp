// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// Std.
#include <latch>
#include <utility>

/// External.
#include "nlohmann/json.hpp"
#include <glog/logging.h>

/// Internal.
#include "driver/driver/driver.h"

using json = nlohmann::json;

driver::TaskManager::TaskManager(
    RackKey rack_key,
    const std::shared_ptr<Synnax>& client,
    std::unique_ptr<task::Factory> factory,
    breaker::Breaker breaker
) : rack_key(rack_key),
    client(client),
    ctx(std::make_shared<task::Context>(client)),
    factory(std::move(factory)),
    exit_err(freighter::NIL),
    breaker(std::move(breaker)),
    internal(rack_key, "") {
}

const std::string TASK_SET_CHANNEL = "sy_task_set";
const std::string TASK_DELETE_CHANNEL = "sy_task_delete";
const std::string TASK_CMD_CHANNEL = "sy_task_cmd";

freighter::Error driver::TaskManager::start(std::latch& latch) {
    LOG(ERROR) << "Starting task manager";
    auto err = startInternal();
    if (err) {
        if (err.type == freighter::TYPE_UNREACHABLE && breaker.wait()) start(latch);
        latch.count_down();
        return err;
    }
    breaker.reset();
    exec_thread = std::thread(&TaskManager::run, this, std::ref(latch));
    return freighter::NIL;
}

freighter::Error driver::TaskManager::startInternal() {
    // Fetch info about the rack.
    auto [rack, rack_err] = client->hardware.retrieveRack(rack_key);
    if (rack_err) return rack_err;
    internal = rack;

    // Fetch task set channel.
    auto [task_set, task_set_err] = client->channels.retrieve(TASK_SET_CHANNEL);
    if (task_set_err) return task_set_err;
    task_set_channel = task_set;

    // Fetch task delete channel.
    auto [task_delete, task_delete_err] = client->channels.retrieve(TASK_DELETE_CHANNEL);
    if (task_delete_err) return task_delete_err;
    task_delete_channel = task_delete;

    // Fetch task command channel.
    auto [task_cmd, task_cmd_err] = client->channels.retrieve(TASK_CMD_CHANNEL);
    if (task_cmd_err) return task_cmd_err;
    task_cmd_channel = task_cmd;

    LOG(ERROR) << "Checkpoint";

    return freighter::NIL;
}


void driver::TaskManager::run(std::latch& latch) {
    LOG(ERROR) << "Running task manager outer";
    auto err = runInternal();
    if (err) {
        // This is the only error type that we retry on.
        if (err.matches(freighter::TYPE_UNREACHABLE) && breaker.wait()) runInternal();
        if (!err.matches(freighter::EOF_)) exit_err = err;
    }
    latch.count_down();
}

freighter::Error driver::TaskManager::stop() {
    auto err = streamer->closeSend();
    exec_thread.join();
    return exit_err;
}

freighter::Error driver::TaskManager::runInternal() {
    LOG(ERROR) << "Running task manager";
    // Open the streamer.
    std::vector stream_channels = {task_set_channel.key, task_delete_channel.key, task_cmd_channel.key};
    auto [s, open_err] = client->telem.openStreamer(StreamerConfig{.channels = stream_channels});
    if (open_err) return open_err;
    streamer = std::make_unique<Streamer>(std::move(s));

    // If we pass here it means we've re-gained network connectivity and can reset the breaker.
    breaker.reset();

    while (true) {
        auto [frame, read_err] = streamer->read();
        LOG(ERROR) << "Read frame";
        if (read_err) return read_err;
        for (size_t i = 0; i < frame.size(); i++) {
            auto& key = (*frame.columns)[i];
            auto& series = (*frame.series)[i];
            if (key == task_set_channel.key) processTaskSet(series);
            else if (key == task_delete_channel.key) processTaskDelete(series);
            else if (key == task_cmd_channel.key) processTaskCmd(series);
        }
    }
    return freighter::NIL;
}

void driver::TaskManager::processTaskSet(const Series& series) {
    auto keys = series.uint64();
    for (auto key: keys) {
        // If a module exists with this key, stop and remove it.
        auto task_iter = tasks.find(key);
        if (task_iter != tasks.end()) {
            task_iter->second->stop();
            tasks.erase(task_iter);
        }
        auto [sy_task, err] = internal.tasks.retrieve(key);
        if (err) {
            std::cerr << err.message() << std::endl;
            continue;
        }
        LOG(ERROR) << "Configuring task: " << sy_task.name << " with key: " << key << ".";
        auto [driver_task, ok] = factory->configureTask(ctx, sy_task);
        if (ok && driver_task != nullptr) {
            tasks[key] = std::move(driver_task);
        }
    }
}

void driver::TaskManager::processTaskCmd(const Series& series) {
    auto commands = series.string();
    for (const auto& cmd_str: commands) {
        LOG(ERROR) << "Processing command: " << cmd_str;
        auto cmd_json = json::parse(cmd_str);
        auto [cmd, err] = task::Command::fromJSON(cmd_json);
        if (err) {
            LOG(ERROR) << "Failed to parse command: " << err;
            continue;
        }
        auto it = tasks.find(cmd.task);
        if (it == tasks.end()) {
            LOG(ERROR) << "Could not find task to execute command: " << cmd.task;
            continue;
        }
        it->second->exec(cmd);
    }
}


void driver::TaskManager::processTaskDelete(const Series& series) {
    auto keys = series.uint64();
    for (auto key: keys) {
        auto it = tasks.find(key);
        if (it != tasks.end()) {
            it->second->stop();
            tasks.erase(it);
        }
    }
}
