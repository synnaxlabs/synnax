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

#include "absl/log/log.h"

#include "x/cpp/json/json.h"
#include "x/cpp/log/log.h"

#include "driver/task/task.h"

namespace driver::task {

x::errors::Error Manager::open_streamer() {
    VLOG(1) << "opening streamer";

    auto node_key = synnax::rack::rack_key_node(this->rack.key);
    const auto control_ch_name = "sy_node_" + std::to_string(node_key) + "_control";

    std::vector chan_names{
        synnax::task::SET_CHANNEL,
        synnax::task::DELETE_CHANNEL,
        synnax::task::CMD_CHANNEL,
        control_ch_name,
    };

    auto [channels, retrieve_err] = this->ctx->client->channels.retrieve(chan_names);
    if (retrieve_err) return retrieve_err;
    if (channels.size() != chan_names.size())
        return x::errors::Error(
            "expected " + std::to_string(chan_names.size()) + " channels, got " +
            std::to_string(channels.size())
        );
    for (const auto &channel: channels)
        if (channel.name == synnax::task::SET_CHANNEL)
            this->channels.task_set = channel;
        else if (channel.name == synnax::task::DELETE_CHANNEL)
            this->channels.task_delete = channel;
        else if (channel.name == synnax::task::CMD_CHANNEL)
            this->channels.task_cmd = channel;
        else if (channel.name == control_ch_name)
            this->channels.control_state = channel;

    if (this->exit_early) return x::errors::NIL;
    std::lock_guard<std::mutex> lock{this->mu};
    auto [s, open_err] = this->ctx->client->telem.open_streamer(
        synnax::framer::StreamerConfig{
            .channels = {
                this->channels.task_set.key,
                this->channels.task_delete.key,
                this->channels.task_cmd.key,
                this->channels.control_state.key
            }
        }
    );
    if (open_err) return open_err;
    this->streamer = std::make_unique<synnax::framer::Streamer>(std::move(s));
    return x::errors::NIL;
}

std::shared_ptr<Manager::Entry> Manager::entry_for(const synnax::task::Key &key) {
    auto &entry = this->entries[key];
    if (!entry) entry = std::make_shared<Entry>();
    return entry;
}

void Manager::remove(const synnax::task::Key &key) {
    const auto it = this->entries.find(key);
    if (it == this->entries.end()) return;
    if (it->second->relevant(this->rack.key)) return;
    this->entries.erase(it);
}

x::errors::Error Manager::configure_initial_tasks() {
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
            const auto entry = this->entry_for(task.key);
            {
                std::lock_guard entry_lock{entry->mu};
                entry->row = task;
            }
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
                const auto entry = this->entry_for(sy_task.key);
                entry->instance = std::move(driver_task);
                std::lock_guard entry_lock{entry->mu};
                entry->row = sy_task;
                entry->deployed_hash = sy_task.config_hash;
            }
        }
    }
    VLOG(1) << "queued " << queued << " initial tasks";
    return x::errors::NIL;
}

void Manager::stop() {
    this->exit_early = true;
    std::lock_guard<std::mutex> lock{this->mu};
    // Very important that we do NOT set the streamer to a nullptr here, as the run()
    // method still needs access before shutting down.
    if (this->streamer != nullptr) this->streamer->close_send();
}

x::errors::Error Manager::run(std::function<void()> on_started) {
    if (this->exit_early) {
        VLOG(1) << "exiting early";
        return x::errors::NIL;
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
        return x::errors::NIL;
    }
    if (const auto err = this->open_streamer()) {
        this->stop_workers();
        return err;
    }
    LOG(INFO) << x::log::GREEN() << "started successfully" << x::log::RESET();
    if (on_started) on_started();
    do {
        auto [frame, read_err] = this->streamer->read();
        if (read_err) break;
        for (size_t i = 0; i < frame.size(); i++) {
            const auto &key = frame.channels->at(i);
            const auto &series = frame.series->at(i);
            if (key == this->channels.task_set.key)
                this->process_task_set(series);
            else if (key == this->channels.task_delete.key)
                this->process_task_delete(series);
            else if (key == this->channels.task_cmd.key)
                this->process_task_cmd(series);
            else if (key == this->channels.control_state.key)
                this->control_states_->apply(series);
        }
    } while (true);
    this->stop_all_tasks();
    this->stop_workers();
    std::lock_guard<std::mutex> lock{this->mu};
    const auto c_err = this->streamer->close();
    this->streamer = nullptr;
    return c_err;
}

void Manager::process_task_set(const x::telem::Series &series) {
    for (const auto &meta_str: series.strings()) {
        auto parser = x::json::Parser(meta_str);
        const auto task_key = parser.field<synnax::task::Key>("key");
        const auto rack_key = parser.field<synnax::rack::Key>("rack", 0);
        const auto name = parser.field<std::string>("name", "");
        const auto type = parser.field<std::string>("type", "");
        const auto config_hash = parser.field<std::string>("config_hash", "");
        const auto snapshot = parser.field<bool>("snapshot", false);
        if (!parser.ok()) {
            LOG(WARNING) << "failed to parse task metadata: "
                         << parser.error_json().dump();
            continue;
        }
        if (snapshot) continue;
        std::lock_guard<std::mutex> lock(this->mu);
        // An untracked task becomes ours only when its row lands on this rack.
        if (this->entries.find(task_key) == this->entries.end() &&
            rack_key != this->rack.key)
            continue;
        const auto entry = this->entry_for(task_key);
        {
            std::lock_guard entry_lock{entry->mu};
            entry->row.key = task_key;
            entry->row.rack = rack_key;
            entry->row.name = name;
            entry->row.type = type;
            entry->row.config_hash = config_hash;
        }
        this->remove(task_key);
    }
}

void Manager::process_task_cmd(const x::telem::Series &series) {
    const auto commands = series.strings();
    for (const auto &cmd_str: commands) {
        auto parser = x::json::Parser(cmd_str);
        auto cmd = synnax::task::Command::parse(parser);
        if (!parser.ok()) {
            LOG(WARNING) << "failed to parse command: " << parser.error_json().dump();
            continue;
        }
        if (cmd.type == synnax::task::START_CMD_TYPE) {
            this->process_start(cmd);
            continue;
        }
        // Non-start commands go to whichever driver holds the live instance, even
        // when the stored row has moved to another rack.
        std::lock_guard<std::mutex> lock(this->mu);
        if (this->entries.find(cmd.task) == this->entries.end()) continue;
        VLOG(1) << "queuing " << cmd.type << " command for task " << cmd.task;
        this->op_queue.push_back(Op{Op::Type::COMMAND, cmd.task, {}, cmd});
        this->cv.notify_one();
    }
}

void Manager::process_start(const synnax::task::Command &cmd) {
    {
        std::lock_guard<std::mutex> lock(this->mu);
        if (const auto it = this->entries.find(cmd.task); it != this->entries.end()) {
            const auto entry = it->second;
            bool ours;
            std::string deployed, wanted;
            {
                std::lock_guard entry_lock{entry->mu};
                ours = entry->row.rack == this->rack.key;
                deployed = entry->deployed_hash;
                // A command carrying a hash states what the sender wants running
                // without waiting for a set event to land.
                wanted = cmd.config_hash.empty() ? entry->row.config_hash
                                                 : cmd.config_hash;
            }
            if (!ours) {
                // The start deploys the task on its new rack and doubles as the
                // teardown signal for the instance held here.
                VLOG(1) << "releasing task moved to another rack " << cmd.task;
                this->op_queue.push_back(Op{Op::Type::RELEASE, cmd.task, {}, {}});
                this->cv.notify_one();
                return;
            }
            if (!deployed.empty() && deployed == wanted) {
                VLOG(1) << "queuing start for live task " << cmd.task;
                this->op_queue.push_back(Op{Op::Type::COMMAND, cmd.task, {}, cmd});
                this->cv.notify_one();
                return;
            }
        }
    }
    // A deploy needs the config, which set events omit.
    auto [tsk, err] = this->rack.tasks.retrieve(cmd.task);
    if (err) {
        LOG(WARNING) << "failed to retrieve task for start: " << err;
        return;
    }
    if (tsk.snapshot) {
        VLOG(1) << "ignoring start for snapshot task " << tsk;
        return;
    }
    std::lock_guard<std::mutex> lock(this->mu);
    if (tsk.rack != this->rack.key) {
        if (this->entries.find(tsk.key) == this->entries.end()) {
            VLOG(1) << "ignoring start for task " << tsk;
            return;
        }
        this->op_queue.push_back(Op{Op::Type::RELEASE, tsk.key, {}, {}});
        this->cv.notify_one();
        return;
    }
    VLOG(1) << "queuing deploy and start for task " << tsk;
    const auto entry = this->entry_for(tsk.key);
    {
        std::lock_guard entry_lock{entry->mu};
        entry->row = tsk;
    }
    this->op_queue.push_back(Op{Op::Type::CONFIGURE, tsk.key, tsk, cmd});
    this->cv.notify_one();
}

void Manager::stop_all_tasks() {
    {
        std::lock_guard<std::mutex> lock(this->mu);
        this->op_queue.clear();
        for (auto &[key, entry]: this->entries) {
            if (!entry) continue;
            this->op_queue.push_back(Op{Op::Type::SHUTDOWN, key, {}, {}});
        }
    }
    this->cv.notify_all();
    const auto deadline = x::telem::TimeStamp::now() + this->shutdown_timeout;
    while (x::telem::TimeStamp::now() < deadline) {
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
        std::this_thread::sleep_for((50 * x::telem::MILLISECOND).chrono());
    }
    this->entries.clear();
}

void Manager::process_task_delete(const x::telem::Series &series) {
    for (const auto &task_key: series.uuids()) {
        std::lock_guard<std::mutex> lock(this->mu);
        if (this->entries.find(task_key) == this->entries.end()) continue;
        this->op_queue.push_back(Op{Op::Type::REMOVE, task_key, {}, {}});
        this->cv.notify_one();
    }
}

void Manager::start_workers() {
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

void Manager::stop_workers() {
    this->breaker.stop();
    this->cv.notify_all();
    const auto deadline = x::telem::TimeStamp::now() + this->shutdown_timeout;
    for (auto &w: this->workers) {
        if (!w.thread.joinable()) continue;
        while (!w.done->load() && x::telem::TimeStamp::now() < deadline)
            std::this_thread::sleep_for((50 * x::telem::MILLISECOND).chrono());
        if (w.done->load())
            w.thread.join();
        else {
            // NOTE: C++ has no safe way to forcibly terminate threads. Detaching here
            // means the thread may access freed Manager resources (use-after-free).
            // This is acceptable because:
            // 1. Manager destruction only occurs during process shutdown
            // 2. The OS will terminate all threads when the process exits
            // 3. The alternative (waiting forever) would hang shutdown
            // If this becomes a mid-process operation, we'd need to either:
            // - Ensure all tasks respect stop() within the timeout
            // - Move tasks to child processes that can be safely killed
            LOG(WARNING) << "worker thread did not finish in time, detaching";
            w.thread.detach();
        }
    }
    this->workers.clear();
    if (this->monitor_thread.joinable()) this->monitor_thread.join();
}

void Manager::worker_loop() {
    while (this->breaker.running()) {
        std::unique_lock<std::mutex> lock(this->mu);
        // Wait until there's work we can actually process - not just ops in the queue,
        // but ops whose entries aren't currently being processed by another worker.
        this->cv.wait(lock, [this] {
            if (!this->breaker.running()) return true;
            for (const auto &op: this->op_queue) {
                const auto it = this->entries.find(op.task_key);
                if (it != this->entries.end() && !it->second->processing.load())
                    return true;
            }
            return false;
        });
        if (!this->breaker.running()) break;

        // Find and claim the first available op. An op whose entry is gone was
        // queued before the task was deleted or released.
        std::shared_ptr<Entry> entry;
        Op op;
        for (auto it = this->op_queue.begin(); it != this->op_queue.end();) {
            const auto e = this->entries.find(it->task_key);
            if (e == this->entries.end()) {
                it = this->op_queue.erase(it);
                continue;
            }
            if (e->second->processing.exchange(true)) {
                ++it;
                continue;
            }
            entry = e->second;
            op = std::move(*it);
            this->op_queue.erase(it);
            break;
        }
        if (!entry) continue;

        {
            std::lock_guard entry_lock{entry->mu};
            entry->op_cmd = op.cmd.key;
            entry->op_config_hash = op.task.config_hash.empty() ? entry->deployed_hash
                                                                : op.task.config_hash;
        }
        entry->timed_out = false;
        entry->op_started = x::telem::TimeStamp::now();
        lock.unlock();
        this->execute_op(op, entry);
        entry->op_started = x::telem::TimeStamp(0);
        entry->processing = false;
        this->cv.notify_all();
    }
}

void Manager::monitor_loop() {
    while (this->breaker.running()) {
        this->breaker.wait_for(this->poll_interval);
        if (!this->breaker.running()) break;
        std::lock_guard<std::mutex> lock(this->mu);
        for (auto &[key, entry]: this->entries) {
            if (!entry->processing) continue;
            auto started = entry->op_started.load();
            if (started.nanoseconds() == 0) continue;
            if (x::telem::TimeStamp::now() - started > this->op_timeout) {
                if (entry->timed_out.exchange(true)) continue;
                LOG(ERROR) << "task " << key << " operation timed out";
                synnax::task::Status status;
                status.key = synnax::task::ontology_id(key).string();
                status.variant = synnax::status::VARIANT_ERROR;
                status.message = "operation timed out";
                status.details.task = key;
                {
                    std::lock_guard entry_lock{entry->mu};
                    status.name = entry->row.name;
                    status.details.config_hash = entry->op_config_hash;
                    // running is left false: the operation is stuck, and the
                    // instance belongs to the claiming worker while processing.
                    status.details.cmd = entry->op_cmd;
                }
                this->ctx->set_status(status);
            }
        }
    }
}

void Manager::execute_op(const Op &op, const std::shared_ptr<Entry> &entry) {
    switch (op.type) {
        case Op::Type::CONFIGURE: {
            if (entry->instance != nullptr) {
                entry->instance->stop(true);
                entry->instance = nullptr;
            }
            LOG(INFO) << "configuring task " << op.task;
            auto [driver_task, handled] = this->factory->configure_task(
                this->ctx,
                op.task,
                op.cmd.key
            );
            {
                std::lock_guard entry_lock{entry->mu};
                entry->deployed_hash = driver_task != nullptr ? op.task.config_hash
                                                              : "";
            }
            if (driver_task != nullptr) {
                entry->instance = std::move(driver_task);
                if (!op.cmd.type.empty()) {
                    auto cmd = op.cmd;
                    LOG(INFO) << "executing command " << cmd << " on task "
                              << entry->instance->name();
                    entry->instance->exec(cmd);
                }
                break;
            }
            if (!handled) {
                LOG(WARNING) << "failed to find integration to handle task" << op.task;
                if (!op.cmd.key.empty()) {
                    synnax::task::Status status;
                    status.key = synnax::task::status_key(op.task);
                    status.name = op.task.name;
                    status.variant = synnax::status::VARIANT_ERROR;
                    status.message = "no driver integration handles task type '" +
                                     op.task.type + "'";
                    status.details.task = op.task_key;
                    status.details.running = false;
                    status.details.cmd = op.cmd.key;
                    status.details.config_hash = op.task.config_hash;
                    this->ctx->set_status(status);
                }
            }
            VLOG(1) << "failed to configure task: " << op.task;
            break;
        }
        case Op::Type::COMMAND: {
            if (entry->instance == nullptr) {
                LOG(WARNING) << "no task for command " << op.task_key;
                return;
            }
            auto cmd = op.cmd;
            LOG(INFO) << "executing command " << cmd << " on task "
                      << entry->instance->name();
            entry->instance->exec(cmd);
            break;
        }
        case Op::Type::SHUTDOWN: {
            if (entry->instance == nullptr) return;
            LOG(INFO) << "shutting down task " << entry->instance->name();
            entry->instance->stop(false);
            break;
        }
        case Op::Type::REMOVE: {
            if (entry->instance != nullptr) {
                LOG(INFO) << "deleting task " << entry->instance->name();
                entry->instance->stop(false);
                entry->instance = nullptr;
            }
            this->clear_deploy(entry);
            // The row is gone from the cluster, so the entry goes unconditionally.
            std::lock_guard lock{this->mu};
            this->entries.erase(op.task_key);
            break;
        }
        case Op::Type::RELEASE: {
            if (entry->instance != nullptr) {
                LOG(INFO) << "releasing task moved to another rack: "
                          << entry->instance->name();
                entry->instance->stop(true);
                entry->instance = nullptr;
            }
            // The row may have moved back to this rack while the release was queued,
            // so the entry can survive and must not keep a stale hash.
            this->clear_deploy(entry);
            std::lock_guard lock{this->mu};
            this->remove(op.task_key);
            break;
        }
    }
}

void Manager::clear_deploy(const std::shared_ptr<Entry> &entry) {
    std::lock_guard lock{entry->mu};
    entry->deployed_hash.clear();
}
}
