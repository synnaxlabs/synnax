// Copyright 2025 Synnax Labs, Inc.
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

#include "x/cpp/xjson/xjson.h"
#include "x/cpp/xlog/xlog.h"
#include "x/cpp/xos/xos.h"

#include "driver/task/task.h"

const std::string TASK_SET_CHANNEL = "sy_task_set";
const std::string TASK_DELETE_CHANNEL = "sy_task_delete";
const std::string TASK_CMD_CHANNEL = "sy_task_cmd";

xerrors::Error task::Manager::open_streamer() {
    VLOG(1) << "opening streamer";
    auto [channels, task_set_err] = this->ctx->client->channels.retrieve(
        {TASK_SET_CHANNEL, TASK_DELETE_CHANNEL, TASK_CMD_CHANNEL}
    );
    if (task_set_err) return task_set_err;
    if (channels.size() != 3)
        return xerrors::Error(
            "expected 3 channels, got " + std::to_string(channels.size())
        );
    for (const auto &channel: channels)
        if (channel.name == TASK_SET_CHANNEL)
            this->channels.task_set = channel;
        else if (channel.name == TASK_DELETE_CHANNEL)
            this->channels.task_delete = channel;
        else if (channel.name == TASK_CMD_CHANNEL)
            this->channels.task_cmd = channel;

    if (this->exit_early) return xerrors::NIL;
    std::lock_guard<std::mutex> lock{this->mu};
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
    return xerrors::NIL;
}

xerrors::Error task::Manager::configure_initial_tasks() {
    VLOG(1) << "configuring initial tasks";
    auto [tasks, tasks_err] = this->rack.tasks.list();
    if (tasks_err) return tasks_err;
    VLOG(1) << "retrieved " << tasks.size() << " tasks from cluster";
    size_t queued = 0;
    {
        std::lock_guard<std::mutex> lock(this->mu);
        for (const auto &task: tasks) {
            if (task.snapshot) {
                VLOG(1) << "ignoring snapshot task " << task;
                continue;
            }
            VLOG(1) << "queuing configure for task " << task;
            this->entries[task.key] = std::make_shared<Entry>();
            this->op_queue.push_back(Op{Op::Type::CONFIGURE, task.key, task, {}});
            queued++;
        }
    }
    if (queued > 0) this->cv.notify_all();
    VLOG(1) << "configuring initial tasks from factories";
    auto initial_tasks = this->factory->configure_initial_tasks(this->ctx, this->rack);
    {
        std::lock_guard<std::mutex> lock(this->mu);
        for (auto &[sy_task, driver_task]: initial_tasks) {
            if (driver_task == nullptr)
                LOG(WARNING) << "unexpected nullptr returned by factory for "
                                "initial task"
                             << sy_task;
            else {
                auto &entry = this->entries[sy_task.key];
                if (!entry) entry = std::make_shared<Entry>();
                entry->task = std::move(driver_task);
            }
        }
    }
    VLOG(1) << "queued " << queued << " initial tasks";
    return xerrors::NIL;
}

void task::Manager::stop() {
    this->exit_early = true;
    std::lock_guard<std::mutex> lock{this->mu};
    // Very important that we do NOT set the streamer to a nullptr here, as the run()
    // method still needs access before shutting down.
    if (this->streamer != nullptr) this->streamer->close_send();
}

bool task::Manager::skip_foreign_rack(const synnax::TaskKey &task_key) const {
    if (synnax::rack_key_from_task_key(task_key) != this->rack.key) {
        VLOG(1) << "received task for foreign rack: " << task_key << ", skipping";
        return true;
    }
    return false;
}

xerrors::Error task::Manager::run(std::function<void()> on_started) {
    if (this->exit_early) {
        VLOG(1) << "exiting early";
        return xerrors::NIL;
    }
    this->start_workers();
    if (const auto err = this->configure_initial_tasks()) {
        this->stop_workers();
        return err;
    }
    if (this->exit_early) {
        VLOG(1) << "exiting early";
        this->stop_workers();
        this->stop_all_tasks();
        return xerrors::NIL;
    }
    if (const auto err = this->open_streamer()) {
        this->stop_workers();
        return err;
    }
    LOG(INFO) << xlog::GREEN() << "started successfully" << xlog::RESET();
    if (on_started) on_started();
    do {
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
    this->stop_workers();
    std::lock_guard<std::mutex> lock{this->mu};
    const auto c_err = this->streamer->close();
    this->streamer = nullptr;
    return c_err;
}

void task::Manager::process_task_set(const telem::Series &series) {
    const auto task_keys = series.values<std::uint64_t>();
    for (const auto task_key: task_keys) {
        if (this->skip_foreign_rack(task_key)) continue;
        auto [tsk, err] = this->rack.tasks.retrieve(task_key);
        if (err) {
            LOG(WARNING) << "failed to retrieve task: " << err;
            continue;
        }
        if (tsk.snapshot) {
            VLOG(1) << "ignoring snapshot task " << tsk;
            continue;
        }
        VLOG(1) << "queuing configure for task " << tsk;
        std::lock_guard<std::mutex> lock(this->mu);
        if (!this->entries[task_key])
            this->entries[task_key] = std::make_shared<Entry>();
        this->op_queue.push_back(Op{Op::Type::CONFIGURE, tsk.key, tsk, {}});
        this->cv.notify_one();
    }
}

void task::Manager::process_task_cmd(const telem::Series &series) {
    const auto commands = series.strings();
    for (const auto &cmd_str: commands) {
        auto parser = xjson::Parser(cmd_str);
        auto cmd = task::Command(parser);
        if (!parser.ok()) {
            LOG(WARNING) << "failed to parse command: " << parser.error_json().dump();
            continue;
        }
        if (this->skip_foreign_rack(cmd.task)) continue;
        VLOG(1) << "queuing " << cmd.type << " command for task " << cmd.task;
        std::lock_guard<std::mutex> lock(this->mu);
        if (!this->entries[cmd.task])
            this->entries[cmd.task] = std::make_shared<Entry>();
        this->op_queue.push_back(Op{Op::Type::COMMAND, cmd.task, {}, cmd});
        this->cv.notify_one();
    }
}

void task::Manager::stop_all_tasks() {
    {
        std::lock_guard<std::mutex> lock(this->mu);
        this->op_queue.clear();
        for (auto &[key, entry]: this->entries) {
            if (!entry) continue;
            this->op_queue.push_back(Op{Op::Type::STOP, key, {}, {}});
        }
    }
    this->cv.notify_all();
    const auto deadline = telem::TimeStamp::now() + this->shutdown_timeout;
    while (telem::TimeStamp::now() < deadline) {
        {
            std::lock_guard<std::mutex> lock(this->mu);
            if (this->op_queue.empty()) {
                bool any_processing = false;
                for (auto &[key, entry]: this->entries) {
                    if (entry && entry->processing) {
                        any_processing = true;
                        break;
                    }
                }
                if (!any_processing) break;
            }
        }
        std::this_thread::sleep_for((50 * telem::MILLISECOND).chrono());
    }
    this->entries.clear();
}

void task::Manager::process_task_delete(const telem::Series &series) {
    const auto task_keys = series.values<synnax::TaskKey>();
    for (const auto task_key: task_keys) {
        if (this->skip_foreign_rack(task_key)) continue;
        std::lock_guard<std::mutex> lock(this->mu);
        if (!this->entries[task_key])
            this->entries[task_key] = std::make_shared<Entry>();
        this->op_queue.push_back(Op{Op::Type::DELETE, task_key, {}, {}});
        this->cv.notify_one();
    }
}

void task::Manager::start_workers() {
    this->breaker.start();
    for (size_t i = 0; i < this->worker_count; i++) {
        auto done = std::make_shared<std::atomic<bool>>(false);
        this->workers.push_back(
            {std::thread([this, done] {
                 this->worker_loop();
                 *done = true;
             }),
             done}
        );
    }
    this->monitor_thread = std::thread([this] { this->monitor_loop(); });
}

void task::Manager::stop_workers() {
    this->breaker.stop();
    this->cv.notify_all();
    const auto deadline = telem::TimeStamp::now() + this->shutdown_timeout;
    for (auto &w: this->workers) {
        if (!w.thread.joinable()) continue;
        while (!w.done->load() && telem::TimeStamp::now() < deadline)
            std::this_thread::sleep_for((50 * telem::MILLISECOND).chrono());
        if (w.done->load())
            w.thread.join();
        else {
            LOG(WARNING) << "worker thread did not finish in time, detaching";
            w.thread.detach();
        }
    }
    this->workers.clear();
    if (this->monitor_thread.joinable()) this->monitor_thread.join();
}

void task::Manager::worker_loop() {
    while (this->breaker.running()) {
        std::unique_lock<std::mutex> lock(this->mu);
        this->cv.wait(lock, [this] {
            return !this->breaker.running() || !this->op_queue.empty();
        });
        if (!this->breaker.running()) break;
        for (auto it = this->op_queue.begin(); it != this->op_queue.end(); ++it) {
            auto entry = this->entries[it->task_key];
            if (!entry->processing.exchange(true)) {
                Op op = std::move(*it);
                this->op_queue.erase(it);
                entry->op_started = telem::TimeStamp::now();
                lock.unlock();
                this->execute_op(op, entry);
                entry->op_started = telem::TimeStamp(0);
                entry->processing = false;
                this->cv.notify_all();
                break;
            }
        }
    }
}

void task::Manager::monitor_loop() {
    while (this->breaker.running()) {
        this->breaker.wait_for(this->poll_interval);
        if (!this->breaker.running()) break;
        std::lock_guard<std::mutex> lock(this->mu);
        for (auto &[key, entry]: this->entries) {
            if (!entry->processing) continue;
            auto started = entry->op_started.load();
            if (started.nanoseconds() == 0) continue;
            if (telem::TimeStamp::now() - started > this->op_timeout) {
                LOG(ERROR) << "task " << key << " operation timed out";
                synnax::TaskStatus status;
                status.key = synnax::task_ontology_id(key).string();
                status.variant = status::variant::ERR;
                status.message = "operation timed out";
                status.details.task = key;
                this->ctx->set_status(status);
            }
        }
    }
}

void task::Manager::execute_op(
    const Op &op,
    const std::shared_ptr<Entry> &entry
) const {
    switch (op.type) {
        case Op::Type::CONFIGURE: {
            if (entry->task != nullptr) entry->task->stop(true);
            LOG(INFO) << "configuring task " << op.task;
            auto [driver_task, handled] = this->factory->configure_task(
                this->ctx,
                op.task
            );
            if (!handled)
                LOG(WARNING) << "failed to find integration to handle task" << op.task;
            if (driver_task != nullptr)
                entry->task = std::move(driver_task);
            else
                VLOG(1) << "failed to configure task: " << op.task;
            break;
        }
        case Op::Type::COMMAND: {
            if (entry->task == nullptr) {
                LOG(WARNING) << "no task for command " << op.task_key;
                return;
            }
            auto cmd = op.cmd;
            LOG(INFO) << "executing command " << cmd << " on task "
                      << entry->task->name();
            entry->task->exec(cmd);
            break;
        }
        case Op::Type::STOP: {
            if (entry->task == nullptr) return;
            LOG(INFO) << "stopping task " << entry->task->name();
            if (entry->task != nullptr) entry->task->stop(false);
            break;
        }
        case Op::Type::DELETE: {
            if (entry->task == nullptr) return;
            LOG(INFO) << "deleting task " << entry->task->name();
            entry->task->stop(false);
            entry->task = nullptr;
            break;
        }
    }
}
