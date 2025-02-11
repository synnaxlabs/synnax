// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>
#include <memory>
#include <thread>
#include "glog/logging.h"
#include "x/cpp/config/config.h"
#include "task.h"
#include "x/cpp/xos/xos.h"

task::Manager::Manager(
    const RackKey &rack_key,
    std::string cluster_key,
    task::PersistRemoteInfo persist_rack_info,
    const std::shared_ptr<Synnax> &client,
    std::unique_ptr<task::Factory> factory,
    const breaker::Config &breaker
) : rack_key(rack_key),
    cluster_key(std::move(cluster_key)),
    ctx(std::make_shared<task::SynnaxContext>(client)),
    factory(std::move(factory)),
    persist_remote_info(std::move(persist_rack_info)),
    breaker(breaker) {
}

task::Manager::~Manager() {
    if (this->run_thread.joinable()) this->run_thread.join();
}

const std::string TASK_SET_CHANNEL = "sy_task_set";
const std::string TASK_DELETE_CHANNEL = "sy_task_delete";
const std::string TASK_CMD_CHANNEL = "sy_task_cmd";

xerrors::Error task::Manager::start() {
    if (this->breaker.running()) return xerrors::NIL;
    VLOG(1) << "[driver] starting up";
    this->breaker.start();
    this->run_thread = std::thread(&Manager::run, this);
    return xerrors::NIL;
}

xerrors::Error task::Manager::start_guarded() {
    // Fetch info about the rack.
    if (const auto rack_err = this->resolve_remote_info()) return rack_err;
    if (auto err = this->persist_remote_info(
        this->rack.key,
        this->ctx->client->auth->cluster_info.cluster_key
    ))
        LOG(WARNING) << "[driver] failed to persist rack info: " << err;

    // Fetch task set channel.
    auto [task_set, task_set_err] = this->ctx->client->channels.retrieve(
        TASK_SET_CHANNEL);
    if (task_set_err) return task_set_err;
    this->channels.task_set = task_set;

    // Fetch task delete channel.
    auto [task_del, task_del_err] = this->ctx->client->channels.retrieve(
        TASK_DELETE_CHANNEL);
    if (task_del_err) return task_del_err;
    this->channels.task_delete = task_del;

    // Fetch task command channel.
    auto [task_cmd, task_cmd_err] = this->ctx->client->channels.retrieve(
        TASK_CMD_CHANNEL);
    this->channels.task_cmd = task_cmd;

    // Retrieve all tasks that are already configured and start them.
    VLOG(1) << "[driver] pulling and configuring existing tasks from Synnax";
    auto [tasks, tasks_err] = this->rack.tasks.list();
    if (tasks_err) return tasks_err;
    for (const auto &task: tasks) {
        auto [driver_task, ok] = this->factory->configure_task(this->ctx, task);
        if (ok && driver_task != nullptr)
            this->tasks[task.key] = std::move(driver_task);
    }

    VLOG(1) << "[driver] configuring initial tasks from factory";
    auto initial_tasks =
            this->factory->configure_initial_tasks(this->ctx, this->rack);
    for (auto &[sy_task, task]: initial_tasks)
        this->tasks[sy_task.key] = std::move(task);

    return task_cmd_err;
}

xerrors::Error task::Manager::resolve_remote_info() {
    std::pair<synnax::Rack, xerrors::Error> res;
    if (const auto err = this->ctx->client->auth->authenticate()) return err;
    if (this->cluster_key != this->ctx->client->auth->cluster_info.cluster_key && this->rack_key != 0) {
        LOG(WARNING) << "[driver] detected a change in cluster key. Resetting rack key";
        this->rack_key = 0;
        this->cluster_key = this->ctx->client->auth->cluster_info.cluster_key;
    }
    if (this->rack_key != 0) {
        // if the rack key is non-zero, it means that persisted state or
        // configuration believes there's an existing rack in the cluster, and
        // we should use it as our task manager's rack.
        if (breaker.num_retries() == 0)
            LOG(INFO) << "[driver] existing rack key found in configuration: " << this->
                    rack_key;
        res = this->ctx->client->hardware.retrieve_rack(this->rack_key);
        // If we tried to retrieve the rack and it doesn't exist, then we assume
        // that:
        //     1. Someone deleted the rack.
        //     2. The cluster identity has changed.
        //
        // In either case, set the rack key to zero and call the instantiate_rack
        // recursively to create a enw rack.
        if (res.second.matches(xerrors::NOT_FOUND)) {
            this->rack_key = 0;
            return this->resolve_remote_info();
        }
    } else {
        /// If the rack key is zero, we should create a new rack to use.
        if (breaker.num_retries() == 0)
            LOG(INFO) <<
                    "[driver] no existing rack key found in configuration. Creating a new rack";
        const auto [host_name, ok] = xos::get_hostname();
        res = this->ctx->client->hardware.create_rack(host_name);
    }
    const xerrors::Error err = res.second;
    // If we can't reach the cluster, keep trying according to the breaker retry logic.
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err.message()))
        return this->resolve_remote_info();

    LOG(INFO) << "[driver] using rack " << res.first.key << " - " << res.first.name;
    this->rack = res.first;
    this->rack_key = res.first.key;
    return err;
}

void task::Manager::run() {
    const auto err = run_guarded();
    if (err.matches(freighter::UNREACHABLE) && this->breaker.wait(err)) return run();
    this->run_err = err;
    for (auto &[key, task]: this->tasks) task->stop();
    VLOG(1) << "[driver] run thread exiting";
}

xerrors::Error task::Manager::stop() {
    if (!this->breaker.running()) return xerrors::NIL;
    if (!this->run_thread.joinable()) return xerrors::NIL;
    this->streamer->close_send();
    this->breaker.stop();
    this->run_thread.join();
    for (auto &[key, task]: this->tasks) {
        LOG(INFO) << "[driver] stopping task " << task->name();
        task->stop();
        LOG(INFO) << "[driver] task " << task->name() << " stopped";
    }
    return run_err;
}

bool task::Manager::skip_foreign_rack(const TaskKey &task_key) const {
    if (task_key !=  this->rack_key) {
        LOG(WARNING) << "[driver] received task for foreign rack: " << task_key;
        return true;
    }
    return false;
}

xerrors::Error task::Manager::run_guarded() {
    if (const auto err = this->start_guarded()) return err;
    auto [s, open_err] = this->ctx->client->telem.open_streamer(StreamerConfig{
        .channels = this->channels.stream_keys()
    });

    if (open_err) return open_err;
    this->streamer = std::make_unique<Streamer>(std::move(s));


    LOG(INFO) << "[driver] operational";
    // If we pass here it means we've re-gained network connectivity and can reset the breaker.
    this->breaker.reset();

    while (this->breaker.running()) {
        auto [frame, read_err] = this->streamer->read();
        if (read_err) break;
        for (size_t i = 0; i < frame.size(); i++) {
            const auto &key = (*frame.channels)[i];
            const auto &series = (*frame.series)[i];
            if (key == this->channels.task_set.key) process_task_set(series);
            else if (key == this->channels.task_delete.key) process_task_delete(series);
            else if (key == this->channels.task_cmd.key) process_task_cmd(series);
        }
    }
    return this->streamer->close();
}

void task::Manager::process_task_set(const telem::Series &series) {
    const auto task_keys = series.values<std::uint64_t>();
    for (const auto task_key: task_keys) {
        // If a module exists with this key, stop and remove it.
        auto task_iter = this->tasks.find(task_key);
        if (task_iter != this->tasks.end()) {
            task_iter->second->stop();
            this->tasks.erase(task_iter);
        }
        if (this->skip_foreign_rack(task_key)) continue;
        auto [sy_task, err] = this->rack.tasks.retrieve(task_key);
        if (err) {
            std::cerr << err.message() << std::endl;
            continue;
        }
        LOG(INFO) << "[driver] configuring task " << sy_task.name << " with key: " <<
                task_key << ".";
        auto [driver_task, ok] = this->factory->configure_task(this->ctx, sy_task);
        if (ok && driver_task != nullptr) this->tasks[task_key] = std::move(driver_task);
        else
            LOG(ERROR) << "[driver] failed to configure task: " << sy_task.name;
    }
}

void task::Manager::process_task_cmd(const telem::Series &series) {
    const auto commands = series.strings();
    for (const auto &cmd_str: commands) {
        auto parser = config::Parser(cmd_str);
        auto cmd = task::Command(parser);
        if (!parser.ok()) {
            LOG(WARNING) << "[driver] failed to parse command: " << parser.error_json().
                    dump();
            continue;
        }
        LOG(INFO) << "[driver] processing command " << cmd.type << " for task " << cmd.
                task;
        if (this->skip_foreign_rack(cmd.task)) continue;
        auto it = this->tasks.find(cmd.task);
        if (it == this->tasks.end()) {
            LOG(WARNING) << "[driver] could not find task to execute command: " << cmd.
                    task;
            continue;
        }
        it->second->exec(cmd);
    }
}


void task::Manager::process_task_delete(const telem::Series &series) {
    const auto task_keys = series.values<std::uint64_t>();
    for (const auto task_key: task_keys) {
        if (this->skip_foreign_rack(task_key)) continue;
        const auto it = this->tasks.find(task_key);
        if (it != this->tasks.end()) {
            it->second->stop();
            this->tasks.erase(it);
        }
    }
}
