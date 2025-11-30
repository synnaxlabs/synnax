// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/status/status.h"

#include "driver/pipeline/acquisition.h"
#include "driver/task/common/factory.h"
#include "driver/task/task.h"

namespace rack::status {
const std::string INTEGRATION_NAME = "rack_status";
const std::string LEGACY_HEARTBEAT_TYPE = "heartbeat";
const std::string TASK_NAME = "Rack Status";
const std::string TASK_TYPE = TASK_NAME;
const auto EMISSION_RATE = telem::HERTZ * 1;

class Source final : public pipeline::Base {
    /// @brief the key of the rack the heartbeat is for.
    const synnax::Rack rack;
    const synnax::Task task;
    /// @brief the loop used to control the emission rate of the heartbeat.
    loop::Timer loop;
    std::shared_ptr<synnax::Synnax> client;

public:
    Source(
        const synnax::Rack &rack,
        const synnax::Task &task,
        const std::shared_ptr<synnax::Synnax> &client
    ):
        Base(
            breaker::Config{
                .name = TASK_NAME,
                .base_interval = 1 * telem::SECOND,
                .max_retries = breaker::RETRY_INFINITELY,
                .scale = 1.05f,
                .max_interval = 5 * telem::SECOND,
            }
        ),
        rack(rack),
        task(task),
        loop(loop::Timer(EMISSION_RATE)),
        client(client) {}

    void run() override {
        synnax::TaskStatus stat{
            .key = this->task.status_key(),
            .name = this->task.name,
            .variant = ::status::variant::SUCCESS,
            .message = "Started",
            .time = telem::TimeStamp::now(),
            .details = synnax::TaskStatusDetails{
                .task = this->task.key,
            }
        };
        if (const auto err = this->client->statuses.set(stat); err)
            LOG(ERROR) << "[rack_status] failed to update task status" << err;
        while (breaker.running()) {
            this->loop.wait(breaker);
            synnax::RackStatus status{
                .key = synnax::rack_ontology_id(this->rack.key).string(),
                .name = this->rack.name,
                .variant = ::status::variant::SUCCESS,
                .message = "Driver is running",
                .time = telem::TimeStamp::now(),
                .details = synnax::RackStatusDetails{.rack = this->rack.key}
            };
            if (const auto err = this->client->statuses.set<synnax::RackStatusDetails>(
                    status
                );
                err)
                LOG(ERROR) << "[rack_status] error updating status: " << err;
            else
                VLOG(1) << "[rack_status] successfully set status" << this->rack.key;
        }
    };
};

/// @brief a task that periodically
/// to indicate that the driver is still alive.
class Task final : public task::Task {
    Source pipe;

public:
    Task(
        const synnax::Rack &rack,
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ):
        pipe(rack, task, ctx->client) {
        this->pipe.start();
    }

    /// @brief implements task::Task.
    std::string name() const override { return TASK_NAME; }

    /// @brief stop the heartbeat process
    void stop(bool will_reconfigure) override { this->pipe.stop(); }

    /// @brief configures the heartbeat task.
    static std::unique_ptr<task::Task>
    configure(const std::shared_ptr<task::Context> &ctx, const synnax::Task &task) {
        auto rack_key = synnax::rack_key_from_task_key(task.key);
        auto [rack, rack_err] = ctx->client->racks.retrieve(rack_key);
        if (rack_err) {
            synnax::TaskStatus stat{
                .key = task.status_key(),
                .name = TASK_NAME,
                .variant = ::status::variant::ERR,
                .message = "Failed to retrieve rack for status task",
                .description = rack_err.message(),
                .details = synnax::TaskStatusDetails{
                    .task = task.key,
                }
            };
            ctx->set_status(stat);
            return nullptr;
        }
        return std::make_unique<Task>(rack, task, ctx);
    }
};

struct Factory final : task::Factory {
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type == TASK_TYPE) return {Task::configure(ctx, task), true};
        return {nullptr, false};
    }

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override {
        common::delete_legacy_task_by_type(
            rack,
            LEGACY_HEARTBEAT_TYPE,
            INTEGRATION_NAME
        );
        return common::configure_initial_factory_tasks(
            this,
            ctx,
            rack,
            TASK_NAME,
            TASK_TYPE,
            INTEGRATION_NAME
        );
    }

    std::string name() override { return INTEGRATION_NAME; }
};
}
