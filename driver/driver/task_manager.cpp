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

/// Internal.
#include "driver/driver/driver.h"

using json = nlohmann::json;

driver::TaskManager::TaskManager(
        synnax::RackKey rack_key,
        const std::shared_ptr<synnax::Synnax> &client,
        std::unique_ptr<TaskFactory> factory,
        breaker::Breaker breaker
) :
        rack_key(rack_key),
        client(client),
        factory(std::move(factory)),
        exit_err(freighter::NIL),
        breaker(std::move(breaker)),
        internal(rack_key, "") {
}

const std::string TASK_SET_CHANNEL = "sy_task_set";
const std::string TASK_DELETE_CHANNEL = "sy_task_delete";
const std::string TASK_CMD_CHANNEL = "sy_task_cmd";
const std::string TASK_STATE_CHANNEL = "sy_task_state";

freighter::Error driver::TaskManager::start(std::latch &latch) {
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
    auto [rack, rack_err] = client->devices.retrieveRack(rack_key.value);
    if (rack_err) return rack_err;
    internal = rack;

    // Fetch module set channel.
    auto [task_set, task_set_err] = client->channels.retrieve(TASK_SET_CHANNEL);
    if (task_set_err) return task_set_err;
    task_set_channel = task_set;

    // Fetch module delete channel.
    auto [task_delete, task_delete_err] = client->channels.retrieve(TASK_DELETE_CHANNEL);
    if (task_delete_err) return task_delete_err;
    task_delete_channel = task_delete;

    // Fetch module comms channel.
    auto [task_state, task_state_err] = client->channels.retrieve(TASK_STATE_CHANNEL);
    if (task_state_err) return task_state_err;
    task_state_channel = task_state;
    return freighter::NIL;
}


void driver::TaskManager::run(std::latch &latch) {
    auto err = runInternal();
    if (err) {
        // This is the only error type that we retry on.
        if (err == freighter::TYPE_UNREACHABLE && breaker.wait()) runInternal();
        exit_err = err;
    }
    latch.count_down();
}

freighter::Error driver::TaskManager::stop() {
    streamer->closeSend();
    exec_thread.join();
    return exit_err;
}

freighter::Error driver::TaskManager::runInternal() {
    // Open the streamer.
    std::vector<synnax::ChannelKey> stream_channels = {task_set_channel.key, task_delete_channel.key};
    auto [s, open_err] = client->telem.openStreamer(synnax::StreamerConfig{.channels = stream_channels});
    if (open_err) return open_err;
    streamer = std::make_unique<synnax::Streamer>(std::move(s));

    // Open the writer.
    std::vector<synnax::ChannelKey> write_channels = {task_state_channel.key};
    auto [writer, writer_err] = client->telem.openWriter(synnax::WriterConfig{.channels = write_channels});
    if (writer_err) return writer_err;

    // If we pass here it means we've re-gained network connectivity and can reset the breaker.
    breaker.reset();

    while (true) {
        auto [frame, read_err] = streamer->read();
        if (read_err) return read_err;
        for (size_t i = 0; i < frame.size(); i++) {
            auto &key = (*frame.columns)[i];
            auto &series = (*frame.series)[i];
            if (key == task_set_channel.key) processTaskSet(series, writer);
            else if (key == task_delete_channel.key) processModuleDelete(series);
        }
    }
    return freighter::NIL;
}

void driver::TaskManager::processTaskSet(const synnax::Series &series, synnax::Writer &comms) {
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
        json config_err;
        bool valid_config = true;
        auto driver_task = factory->createTask(client, sy_task, valid_config, config_err);
        if (!valid_config) {
            json config_err_pld;
            config_err_pld["type"] = "config_error";
            config_err_pld["error"] = config_err;
            config_err_pld["module"] = sy_task.key.value;
            auto fr = synnax::Frame(1);
            fr.add(task_state_channel.key, synnax::Series(std::vector<std::string>{to_string(config_err_pld)}, synnax::JSON));
            comms.write(std::move(fr));
            continue;
        }
        tasks[key] = std::move(driver_task);
    }
}

void driver::TaskManager::processModuleDelete(const synnax::Series &series) {
    auto keys = series.uint64();
    for (auto key: keys) {
        auto it = tasks.find(key);
        if (it != tasks.end()) {
            it->second->stop();
            tasks.erase(it);
        }
    }
}