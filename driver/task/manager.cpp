// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <memory>
#include <thread>
#include <utility>

#include "glog/logging.h"

#include "x/cpp/json/json.h"
#include "x/cpp/log/log.h"
#include "x/cpp/os/os.h"

#include "driver/task/task.h"

const std::string TASK_SET_CHANNEL = "sy_task_set";
const std::string TASK_DELETE_CHANNEL = "sy_task_delete";
const std::string TASK_CMD_CHANNEL = "sy_task_cmd";

x::errors::Error driver::task::Manager::open_streamer() {
    VLOG(1) << "opening streamer";
    auto [channels, task_set_err] = this->ctx->client->channels.retrieve(
        {TASK_SET_CHANNEL, TASK_DELETE_CHANNEL, TASK_CMD_CHANNEL}
    );
    if (task_set_err) return task_set_err;
    if (channels.size() != 3)
        return x::errors::Error(
            "expected 3 channels, got " + std::to_string(channels.size())
        );
    for (const auto &channel: channels)
        if (channel.name == TASK_SET_CHANNEL)
            this->channels.task_set = channel;
        else if (channel.name == TASK_DELETE_CHANNEL)
            this->channels.task_delete = channel;
        else if (channel.name == TASK_CMD_CHANNEL)
            this->channels.task_cmd = channel;

    if (this->exit_early) return x::errors::NIL;
    std::lock_guard lock{this->mu};
    auto [s, open_err] = this->ctx->client->telem.open_streamer(
        synnax::StreamerConfig{
            .channels = {
                this->channels.task_set.key,
                this->channels.task_delete.key,
                this->channels.task_cmd.key
            }
        }
    );
    if (open_err) return open_err;
    this->streamer = std::make_unique<synnax::Streamer>(std::move(s));
    return x::errors::NIL;
}

x::errors::Error driver::task::Manager::configure_initial_tasks() {
    VLOG(1) << "configuring initial tasks";
    auto [tasks, tasks_err] = this->rack.tasks.list();
    if (tasks_err) return tasks_err;
    VLOG(1) << "retrieved " << tasks.size() << " tasks from cluster";
    for (const auto &task: tasks) {
        VLOG(1) << "configuring task " << task;
        if (task.snapshot) {
            VLOG(1) << "ignoring snapshot task " << task;
            continue;
        }
        auto [driver_task, handled] = this->factory->configure_task(this->ctx, task);
        if (handled && driver_task != nullptr)
            this->tasks[task.key] = std::move(driver_task);
        else if (handled && driver_task == nullptr)
            VLOG(1) << "nullptr returned by factory for " << task;
    }
    VLOG(1) << "configuring initial tasks from factories";
    auto initial_tasks = this->factory->configure_initial_tasks(this->ctx, this->rack);
    for (auto &[sy_task, driver_task]: initial_tasks) {
        if (driver_task == nullptr)
            LOG(ERROR) << "failed to configure task: " << sy_task;
        else
            this->tasks[sy_task.key] = std::move(driver_task);
    }
    VLOG(1) << "configured tasks";
    return x::errors::NIL;
}

void driver::task::Manager::stop() {
    this->exit_early = true;
    std::lock_guard lock{this->mu};
    // Very important that we do NOT set the streamer to a nullptr here, as the run()
    // method still needs access before shutting down.
    if (this->streamer != nullptr) this->streamer->close_send();
}

bool driver::task::Manager::skip_foreign_rack(const synnax::task::Key &task_key) const {
    if (synnax::rack_key_from_task_key(task_key) != this->rack.key) {
        VLOG(1) << "received task for foreign rack: " << task_key << ", skipping";
        return true;
    }
    return false;
}

x::errors::Error driver::task::Manager::run(std::function<void()> on_started) {
    if (this->exit_early) {
        VLOG(1) << "exiting early";
        return x::errors::NIL;
    }
    if (const auto err = this->configure_initial_tasks()) return err;
    if (this->exit_early) {
        VLOG(1) << "exiting early";
        this->stop_all_tasks();
        return x::errors::NIL;
    }
    if (const auto err = this->open_streamer()) return err;
    LOG(INFO) << x::log::GREEN() << "started successfully" << x::log::RESET();
    if (on_started) on_started();
    do {
        // no need to lock the streamer here, as it's safe to call close_send()
        // and read() concurrently.
        auto [frame, read_err] = this->streamer->read();
        if (read_err) break;
        for (size_t i = 0; i < frame.size(); i++) {
            const auto &key = frame.channels->at(i);
            const auto &series = frame.series->at(i);
            if (key == this->channels.task_set.key)
                process_task_set(series);
            else if (key == this->channels.task_delete.key)
                process_task_delete(series);
            else if (key == this->channels.task_cmd.key)
                process_task_cmd(series);
        }
    } while (true);
    this->stop_all_tasks();
    std::lock_guard lock{this->mu};
    const auto c_err = this->streamer->close();
    this->streamer = nullptr;
    return c_err;
}

void driver::task::Manager::process_task_set(const x::telem::Series &series) {
    const auto task_keys = series.values<std::uint64_t>();
    for (const auto task_key: task_keys) {
        if (this->skip_foreign_rack(task_key)) continue;

        auto task_iter = this->tasks.find(task_key);
        if (task_iter != this->tasks.end()) {
            task_iter->second->stop(true);
            this->tasks.erase(task_iter);
        }

        auto [sy_task, err] = this->rack.tasks.retrieve(task_key);
        if (sy_task.snapshot) {
            VLOG(1) << "ignoring snapshot task " << sy_task;
            continue;
        }
        if (err) {
            LOG(WARNING) << "failed to retrieve task: " << err;
            continue;
        }
        LOG(INFO) << "configuring task " << sy_task;
        auto [driver_task, handled] = this->factory->configure_task(this->ctx, sy_task);
        if (handled && driver_task != nullptr)
            this->tasks[task_key] = std::move(driver_task);
        else
            LOG(ERROR) << "failed to configure task: " << sy_task.name;
    }
}

void driver::task::Manager::process_task_cmd(const x::telem::Series &series) {
    const auto commands = series.strings();
    for (const auto &cmd_str: commands) {
        auto parser = x::json::Parser(cmd_str);
        auto cmd = driver::task::Command(parser);
        if (!parser.ok()) {
            LOG(WARNING) << "failed to parse command: " << parser.error_json().dump();
            continue;
        }

        if (this->skip_foreign_rack(cmd.task)) continue;
        auto it = this->tasks.find(cmd.task);
        if (it == this->tasks.end()) {
            LOG(WARNING) << "could not find task to execute command: " << cmd.task;
            continue;
        }
        const std::unique_ptr<Task> &tsk = it->second;
        LOG(INFO) << "processing " << cmd.type << " command for task " << tsk->name()
                  << " (" << cmd.task << ")";
        tsk->exec(cmd);
    }
}

void driver::task::Manager::stop_all_tasks() {
    for (auto &[task_key, task]: this->tasks) {
        VLOG(1) << "stopping task " << task->name();
        task->stop(false);
    }
    this->tasks.clear();
}

void driver::task::Manager::process_task_delete(const x::telem::Series &series) {
    const auto task_keys = series.values<synnax::task::Key>();
    for (const auto task_key: task_keys) {
        if (this->skip_foreign_rack(task_key)) continue;
        const auto it = this->tasks.find(task_key);
        if (it != this->tasks.end()) {
            LOG(INFO) << "stopping task " << it->second->name();
            it->second->stop(false);
            this->tasks.erase(it);
        } else
            LOG(WARNING) << "could not find task for " << task_key << " to delete";
    }
}
