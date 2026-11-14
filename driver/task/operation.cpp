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
    std::string name,
    synnax::task::Command cmd,
    std::string config_hash,
    const x::telem::TimeSpan limit
):
    cmd(std::move(cmd)),
    write(std::move(write)),
    name(std::move(name)),
    config_hash(std::move(config_hash)),
    limit(limit) {}

void Operation::set_task(const std::string &name, const std::string &config_hash) {
    std::lock_guard lock{this->mu};
    this->name = name;
    this->config_hash = config_hash;
}

std::pair<std::string, std::string> Operation::task_info() {
    std::lock_guard lock{this->mu};
    return {this->name, this->config_hash};
}

void Operation::begin_step(const synnax::task::Status &status) {
    this->current = status;
    this->step_started = x::telem::TimeStamp::now();
    this->warned = false;
}

void Operation::step(synnax::task::Status status, const x::telem::TimeSpan limit) {
    status.details.cmd = this->cmd.key;
    std::lock_guard write_lock{this->write_mu};
    {
        std::lock_guard lock{this->mu};
        if (this->finished) {
            LOG(ERROR) << "[task.operation] dropped step '" << status.message
                       << "' sent after the final status of command " << this->cmd;
            return;
        }
        this->begin_step(status);
        this->limit = limit;
    }
    this->write(status);
}

void Operation::answer(synnax::task::Status status) {
    status.details.cmd = this->cmd.key;
    std::lock_guard write_lock{this->write_mu};
    {
        std::lock_guard lock{this->mu};
        if (this->finished)
            status.details.cmd.clear();
        else if (status.variant == synnax::status::VARIANT_LOADING)
            this->begin_step(status);
        else
            this->finished = true;
    }
    this->write(status);
}

void Operation::check(const x::telem::TimeStamp now) {
    std::lock_guard write_lock{this->write_mu};
    synnax::task::Status status;
    {
        std::lock_guard lock{this->mu};
        if (this->finished || this->warned || !this->current.has_value()) return;
        if (now - this->step_started <= this->limit) return;
        this->warned = true;
        LOG(WARNING) << "[task.operation] step '" << this->current->message
                     << "' of command " << this->cmd << " on task " << this->name
                     << " has run past its limit of " << this->limit;
        status = *this->current;
    }
    status.time = now;
    status.description = "Still working after " + this->limit.to_string() + ".";
    this->write(status);
}
}
