// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <utility>

#include "absl/log/log.h"

#include "driver/task/operation.h"

namespace driver::task {
Operation::Operation(
    Writer write,
    synnax::task::Key task,
    std::string name,
    synnax::task::Command cmd,
    std::string config_hash,
    const x::telem::TimeSpan limit
):
    cmd(std::move(cmd)),
    write(std::move(write)),
    task(task),
    name(std::move(name)),
    config_hash(std::move(config_hash)),
    step_started(x::telem::TimeStamp::now()),
    limit(limit) {
    this->breaker.start();
}

Operation::~Operation() {
    this->breaker.stop();
}

void Operation::set_task(const std::string &name, const std::string &config_hash) {
    std::lock_guard lock{this->mu};
    this->name = name;
    this->config_hash = config_hash;
}

std::pair<std::string, std::string> Operation::task_info() {
    std::lock_guard lock{this->mu};
    return {this->name, this->config_hash};
}

bool Operation::cancel(const synnax::task::Command &stop) {
    {
        std::lock_guard lock{this->mu};
        if (this->finished || this->closed) return false;
        if (!stop.key.empty()) this->stopping = stop;
        LOG(INFO) << "[task.operation] cancelling " << this->cmd << " on task "
                  << this->name;
    }
    this->breaker.stop();
    return true;
}

bool Operation::join(
    const synnax::task::Command &start,
    const std::string &config_hash
) {
    std::lock_guard lock{this->mu};
    if (this->closed || this->finished || !this->breaker.running() ||
        this->cmd.type != synnax::task::START_CMD_TYPE || this->cmd.pinned ||
        this->config_hash.empty() || this->config_hash != config_hash)
        return false;
    this->joined.push_back(start);
    this->unanswered.push_back(start.key);
    return true;
}

std::vector<std::string> Operation::keys() const {
    std::vector<std::string> keys{this->cmd.key};
    for (const auto &start: this->joined)
        keys.push_back(start.key);
    return keys;
}

void Operation::send(
    synnax::task::Status status,
    const std::vector<std::string> &keys
) {
    for (const auto &key: keys) {
        status.details.cmd = key;
        this->write(status);
    }
}

void Operation::begin_step(const synnax::task::Status &status) {
    this->current = status;
    this->step_started = x::telem::TimeStamp::now();
    this->warned = false;
}

void Operation::step(synnax::task::Status status, const x::telem::TimeSpan limit) {
    std::lock_guard write_lock{this->write_mu};
    std::vector<std::string> keys;
    {
        std::lock_guard lock{this->mu};
        if (this->finished) {
            LOG(ERROR) << "[task.operation] dropped step '" << status.message
                       << "' sent after the final status of " << this->cmd;
            return;
        }
        this->begin_step(status);
        this->limit = limit;
        this->unanswered.clear();
        keys = this->keys();
    }
    this->send(std::move(status), keys);
}

void Operation::answer(synnax::task::Status status) {
    std::lock_guard write_lock{this->write_mu};
    std::vector<std::string> keys;
    {
        std::lock_guard lock{this->mu};
        if (this->finished)
            keys = {""};
        else {
            if (status.variant == synnax::status::VARIANT_LOADING)
                this->begin_step(status);
            else
                this->finished = true;
            this->unanswered.clear();
            keys = this->keys();
        }
    }
    this->send(std::move(status), keys);
}

void Operation::check(const x::telem::TimeStamp now) {
    std::lock_guard write_lock{this->write_mu};
    std::vector<std::pair<synnax::task::Status, std::vector<std::string>>> out;
    {
        std::lock_guard lock{this->mu};
        if (this->closed || this->finished) return;
        if (this->current.has_value() && !this->unanswered.empty())
            out.emplace_back(*this->current, std::exchange(this->unanswered, {}));
        if (this->stopping.has_value()) {
            synnax::task::Status status;
            status.owner = synnax::task::resource_id(this->task);
            status.name = this->name;
            status.variant = synnax::status::VARIANT_LOADING;
            status.message = "Stopping task";
            status.description = "Cancelling the start.";
            status.details.config_hash = this->config_hash;
            out.emplace_back(std::move(status), std::vector{this->stopping->key});
            this->stopping.reset();
        }
        if (!this->warned && now - this->step_started > this->limit) {
            this->warned = true;
            if (!this->current.has_value())
                LOG(WARNING) << "[task.operation] " << this->cmd << " on task "
                             << this->name << " has run past its limit of "
                             << this->limit;
            else {
                LOG(WARNING) << "[task.operation] step '" << this->current->message
                             << "' of " << this->cmd << " on task " << this->name
                             << " has run past its limit of " << this->limit;
                auto status = *this->current;
                status.description = "Still working after " + this->limit.to_string() +
                                     ".";
                out.emplace_back(std::move(status), this->keys());
            }
        }
    }
    for (auto &[status, keys]: out) {
        status.time = now;
        this->send(std::move(status), keys);
    }
}

std::vector<synnax::task::Command> Operation::close() {
    std::scoped_lock lock{this->write_mu, this->mu};
    this->closed = true;
    if (this->finished) return {};
    return std::exchange(this->joined, {});
}
}
