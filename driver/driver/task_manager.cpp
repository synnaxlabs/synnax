// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <latch>
#include <utility>
#include <memory>
#include <glog/logging.h>
#include "driver/driver/config/config.h"
#include "driver/driver/driver.h"

driver::TaskManager::TaskManager(
    const RackKey rack_key,
    const std::shared_ptr<Synnax> &client,
    std::unique_ptr<task::Factory> factory,
    breaker::Breaker breaker
) : rack_key(rack_key),
    internal(rack_key, ""),
    ctx(std::make_shared<task::SynnaxContext>(client)),
    factory(std::move(factory)),
    breaker(std::move(breaker)) {
}

const std::string TASK_SET_CHANNEL = "sy_task_set";
const std::string TASK_DELETE_CHANNEL = "sy_task_delete";
const std::string TASK_CMD_CHANNEL = "sy_task_cmd";

freighter::Error driver::TaskManager::start(std::latch &latch) {
    LOG(INFO) << "starting task manager";
    auto err = startGuarded();
    if (err) {
        if (err.matches(freighter::UNREACHABLE) && breaker.wait()) start(latch);
        latch.count_down();
        return err;
    }
    breaker.reset();
    run_thread = std::thread(&TaskManager::run, this, std::ref(latch));
    return freighter::NIL;
}

freighter::Error driver::TaskManager::startGuarded() {
    // Fetch info about the rack.
    auto [rack, rack_err] = ctx->client->hardware.retrieveRack(rack_key);
    if (rack_err) return rack_err;
    internal = rack;

    // Fetch task set channel.
    auto [task_set, task_set_err] = ctx->client->channels.retrieve(TASK_SET_CHANNEL);
    if (task_set_err) return task_set_err;
    task_set_channel = task_set;

    // Fetch task delete channel.
    auto [task_del, task_del_err] = ctx->client->channels.retrieve(TASK_DELETE_CHANNEL);
    if (task_del_err) return task_del_err;
    task_delete_channel = task_del;

    // Fetch task command channel.
    auto [task_cmd, task_cmd_err] = ctx->client->channels.retrieve(TASK_CMD_CHANNEL);
    task_cmd_channel = task_cmd;

    return task_cmd_err;
}


void driver::TaskManager::run(std::latch &latch) {
    const auto err = runGuarded();
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message()))
        return run(latch);
    latch.count_down();
    run_err = err;
}

freighter::Error driver::TaskManager::stop() {
    streamer->closeSend();
    run_thread.join();
    return run_err;
}

freighter::Error driver::TaskManager::runGuarded() {
    const std::vector<ChannelKey> stream_channels = {
        task_set_channel.key, task_delete_channel.key, task_cmd_channel.key
    };
    auto [s, open_err] = ctx->client->telem.openStreamer(StreamerConfig{
        .channels = stream_channels
    });
    if (open_err) return open_err;
    streamer = std::make_unique<Streamer>(std::move(s));

    // If we pass here it means we've re-gained network connectivity and can reset the breaker.
    breaker.reset();

    while (true) {
        auto [frame, read_err] = streamer->read();
        if (read_err) return read_err;
        for (size_t i = 0; i < frame.size(); i++) {
            const auto &key = (*frame.columns)[i];
            const auto &series = (*frame.series)[i];
            if (key == task_set_channel.key) processTaskSet(series);
            else if (key == task_delete_channel.key) processTaskDelete(series);
            else if (key == task_cmd_channel.key) processTaskCmd(series);
        }
    }
    return freighter::NIL;
}

void driver::TaskManager::processTaskSet(const Series &series) {
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
        LOG(ERROR) << "Configuring task: " << sy_task.name << " with key: " << key <<
                ".";
        auto [driver_task, ok] = factory->configureTask(ctx, sy_task);
        if (ok && driver_task != nullptr) {
            tasks[key] = std::move(driver_task);
        }
    }
}

void driver::TaskManager::processTaskCmd(const Series &series) {
    const auto commands = series.string();
    for (const auto &cmd_str: commands) {
        LOG(ERROR) << "Processing command: " << cmd_str;
        auto parser = config::Parser(cmd_str);
        auto cmd = task::Command(parser);
        if (!parser.ok()) {
            LOG(ERROR) << "Failed to parse command: " << parser.error_json().dump();
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


void driver::TaskManager::processTaskDelete(const Series &series) {
    const auto keys = series.uint64();
    for (auto key: keys) {
        const auto it = tasks.find(key);
        if (it != tasks.end()) {
            it->second->stop();
            tasks.erase(it);
        }
    }
}
