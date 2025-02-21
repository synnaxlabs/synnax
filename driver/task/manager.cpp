// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// std
#include <utility>
#include <memory>
#include <thread>

/// external
#include "glog/logging.h"

/// internal
#include "driver/task/task.h"

/// module
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xlog/xlog.h"
#include "x/cpp/xos/xos.h"

const std::string TASK_SET_CHANNEL = "sy_task_set";
const std::string TASK_DELETE_CHANNEL = "sy_task_delete";
const std::string TASK_CMD_CHANNEL = "sy_task_cmd";

xerrors::Error task::Manager::open_streamer() {
    auto [channels, task_set_err] = this->ctx->client->channels.retrieve({
        TASK_SET_CHANNEL,
        TASK_DELETE_CHANNEL,
        TASK_CMD_CHANNEL
    });
    if (task_set_err) return task_set_err;
    if (channels.size() != 3)
        return xerrors::Error(
            "expected 3 channels, got " + std::to_string(channels.size()));
    for (const auto &channel: channels)
        if (channel.name == TASK_SET_CHANNEL) this->channels.task_set = channel;
        else if (channel.name == TASK_DELETE_CHANNEL)
            this->channels.task_delete = channel;
        else if (channel.name == TASK_CMD_CHANNEL) this->channels.task_cmd = channel;

    std::lock_guard lock{this->mu};
    auto [s, open_err] = this->ctx->client->telem.open_streamer(StreamerConfig{
        .channels = {
            this->channels.task_set.key,
            this->channels.task_delete.key,
            this->channels.task_cmd.key
        }
    });
    if (open_err) return open_err;
    this->streamer = std::make_unique<Streamer>(std::move(s));
    return xerrors::NIL;
}

xerrors::Error task::Manager::configure_initial_tasks() {
    auto [tasks, tasks_err] = this->rack.tasks.list();
    if (tasks_err)return tasks_err;
    for (const auto &task: tasks) {
        auto [driver_task, ok] = this->factory->configure_task(this->ctx, task);
        if (ok && driver_task != nullptr)
            this->tasks[task.key] = std::move(driver_task);
    }
    auto initial_tasks =
            this->factory->configure_initial_tasks(this->ctx, this->rack);
    for (auto &[sy_task, task]: initial_tasks)
        this->tasks[sy_task.key] = std::move(task);
    return xerrors::NIL;
}

void task::Manager::stop() {
    std::lock_guard lock{this->mu};
    // Very important that we do NOT set the streamer to a nullptr here, as the run()
    // method still needs access before shutting down.
    if (this->streamer != nullptr) this->streamer->close_send();
}

bool task::Manager::skip_foreign_rack(const TaskKey &task_key) const {
    if (synnax::task_key_rack(task_key) != this->rack.key) {
        LOG(INFO) << "[driver] received task for foreign rack: " << task_key << ", skipping";
        return true;
    }
    return false;
}

xerrors::Error task::Manager::run(std::promise<void>* started_promise) {
    if (const auto err = this->configure_initial_tasks()) return err;
    if (const auto err = this->open_streamer()) return err;
    LOG(INFO) << xlog::GREEN << "[driver] started successfully" << xlog::RESET;
    if (started_promise != nullptr) started_promise->set_value();
    do {
        // no need to lock the streamer here, as it's safe to call close_send()
        // and read() concurrently.
        auto [frame, read_err] = this->streamer->read();
        if (read_err) break;
        for (size_t i = 0; i < frame.size(); i++) {
            const auto &key = frame.channels->at(i);
            const auto &series = frame.series->at(i);
            if (key == this->channels.task_set.key) process_task_set(series);
            else if (key == this->channels.task_delete.key) process_task_delete(series);
            else if (key == this->channels.task_cmd.key) process_task_cmd(series);
        }
    } while (true);
    this->stop_all_tasks();
    std::lock_guard lock{this->mu};
    const auto c_err = this->streamer->close();
    this->streamer = nullptr;
    return c_err;
}

void task::Manager::process_task_set(const telem::Series &series) {
    const auto task_keys = series.values<std::uint64_t>();
    for (const auto task_key: task_keys) {
        if (this->skip_foreign_rack(task_key)) continue;

        auto task_iter = this->tasks.find(task_key);
        if (task_iter != this->tasks.end()) {
            task_iter->second->stop();
            this->tasks.erase(task_iter);
        }

        auto [sy_task, err] = this->rack.tasks.retrieve(task_key);
        if (err) {
            LOG(WARNING) << "[driver] failed to retrieve task: " << err;
            continue;
        }
        LOG(INFO) << "[driver] configuring task " << sy_task.name << " (" << task_key << ")";
        auto [driver_task, ok] = this->factory->configure_task(this->ctx, sy_task);
        if (ok && driver_task != nullptr)
            this->tasks[task_key] = std::move(driver_task);
        else
            LOG(ERROR) << "[driver] failed to configure task: " << sy_task.name;
    }
}

void task::Manager::process_task_cmd(const telem::Series &series) {
    const auto commands = series.strings();
    for (const auto &cmd_str: commands) {
        auto parser = xjson::Parser(cmd_str);
        auto cmd = task::Command(parser);
        if (!parser.ok()) {
            LOG(WARNING) << "[driver] failed to parse command: " << parser.error_json().
                    dump();
            continue;
        }

        if (this->skip_foreign_rack(cmd.task)) continue;
        auto it = this->tasks.find(cmd.task);
        if (it == this->tasks.end()) {
            LOG(WARNING) << "[driver] could not find task to execute command: " << cmd.
                    task;
            continue;
        }
        const std::unique_ptr<Task> &tsk = it->second;
        LOG(INFO) << "[driver] processing " << cmd.type << " command for task " << tsk->name() << " (" << cmd.task << ")";
        tsk->exec(cmd);
    }
}

void task::Manager::stop_all_tasks() {
    for (auto &[task_key, task]: this->tasks) task->stop();
    this->tasks.clear();
}

void task::Manager::process_task_delete(const telem::Series &series) {
    const auto task_keys = series.values<synnax::TaskKey>();
    for (const auto task_key: task_keys) {
        if (this->skip_foreign_rack(task_key)) continue;
        const auto it = this->tasks.find(task_key);
        if (it != this->tasks.end()) {
            it->second->stop();
            this->tasks.erase(it);
        }
    }
}
