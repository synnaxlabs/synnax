// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <utility>

#include "absl/log/log.h"

#include "client/cpp/status/status.h"
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"

#include "driver/common/factory.h"
#include "driver/pipeline/base.h"
#include "driver/task/task.h"

namespace driver::rack::status {
const std::string INTEGRATION_NAME = "rack_status";
const std::string LEGACY_HEARTBEAT_TYPE = "heartbeat";
const std::string TASK_NAME = "Rack Status";
const std::string TASK_TYPE = "rack_status";
const auto EMISSION_RATE = x::telem::HERTZ * 1;
/// @brief a gap between good heartbeats after which the Core may have marked the tasks
/// of the rack as stopped.
const auto LATE_GAP = EMISSION_RATE.period() * 3;

class Source final : public driver::pipeline::Base {
    /// @brief the key of the rack the heartbeat is for.
    const synnax::rack::Key rack_key;
    const synnax::task::Task task;
    /// @brief the loop used to control the emission rate of the heartbeat.
    x::loop::Timer loop;
    std::shared_ptr<task::Context> ctx;

public:
    Source(
        const synnax::rack::Rack &rack,
        synnax::task::Task task,
        const std::shared_ptr<task::Context> &ctx
    ):
        Base(
            x::breaker::Config{
                .name = TASK_NAME,
                .base_interval = 1 * x::telem::SECOND,
                .max_retries = x::breaker::RETRY_INFINITELY,
                .scale = 1.05f,
                .max_interval = 5 * x::telem::SECOND,
            },
            TASK_NAME
        ),
        rack_key(rack.key),
        task(std::move(task)),
        loop(x::loop::Timer(EMISSION_RATE)),
        ctx(ctx) {}

    void run() override {
        synnax::task::Status stat{
            .key = synnax::task::status_key(this->task),
            .name = this->task.name,
            .variant = ::synnax::status::VARIANT_SUCCESS,
            .message = "Started",
            .time = x::telem::TimeStamp::now(),
            .details = synnax::task::StatusDetails{
                .task = this->task.key,
                .running = true,
                .config_hash = this->task.config_hash,
            }
        };
        this->ctx->set_status(stat);
        bool resend_pending = false;
        auto last_beat = x::telem::TimeStamp::now();
        while (breaker.running()) {
            this->loop.wait(breaker);
            synnax::rack::Status status{
                .key = synnax::rack::ontology_id(this->rack_key).string(),
                .name = this->ctx->rack_name(),
                .variant = ::synnax::status::VARIANT_SUCCESS,
                .message = "Driver is running",
                .time = x::telem::TimeStamp::now(),
                .details = synnax::rack::StatusDetails{.rack = this->rack_key}
            };
            if (const auto err = this->ctx->client->statuses
                                     .set<synnax::rack::StatusDetails>(status);
                err) {
                LOG(ERROR) << "[rack_status] error updating status: " << err;
                // Task status writes also fail while the Core is out of reach.
                resend_pending = true;
                continue;
            }
            VLOG(1) << "[rack_status] successfully set status for rack "
                    << this->rack_key;
            // A sleep or a slow write leaves a gap without a failed write.
            const auto now = x::telem::TimeStamp::now();
            if (now - last_beat > LATE_GAP) resend_pending = true;
            last_beat = now;
            if (!resend_pending) continue;
            if (const auto resend_err = this->ctx->resend_statuses()) {
                LOG(ERROR) << "[rack_status] failed to send task statuses again: "
                           << resend_err;
                continue;
            }
            resend_pending = false;
        }
        stat.message = "Stopped";
        stat.time = x::telem::TimeStamp::now();
        stat.details.running = false;
        this->ctx->set_status(stat);
    };

    void stopped_with_err(const x::errors::Error &err) override {
        synnax::task::Status stat{
            .key = synnax::task::status_key(this->task),
            .name = this->task.name,
            .variant = ::synnax::status::VARIANT_ERROR,
            .message = "Failed to report Driver status",
            .description = err.data,
            .time = x::telem::TimeStamp::now(),
            .details = synnax::task::StatusDetails{
                .task = this->task.key,
                .running = false,
                .config_hash = this->task.config_hash,
            }
        };
        this->ctx->set_status(stat);
    }
};

/// @brief a task that periodically
/// to indicate that the driver is still alive.
class Task final : public driver::task::Task {
    Source pipe;

public:
    Task(
        const synnax::rack::Rack &rack,
        const synnax::task::Task &task,
        const std::shared_ptr<task::Context> &ctx
    ):
        pipe(rack, task, ctx) {
        this->pipe.start();
    }

    /// @brief implements driver::task::Task.
    std::string name() const override { return TASK_NAME; }

    /// @brief stop the heartbeat process
    void stop(bool will_reconfigure) override { this->pipe.stop(); }

    /// @brief configures the heartbeat task.
    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::task::Task &task
    ) {
        auto rack_key = task.rack;
        auto [rack, rack_err] = ctx->client->racks.retrieve(rack_key);
        if (rack_err) {
            synnax::task::Status stat{
                .key = synnax::task::status_key(task),
                .name = TASK_NAME,
                .variant = ::synnax::status::VARIANT_ERROR,
                .message = "Failed to retrieve rack for status task",
                .description = rack_err.data,
                .details = synnax::task::StatusDetails{
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
        const synnax::task::Task &task,
        const std::string &cmd_key
    ) override {
        if (task.type == TASK_TYPE) return {Task::configure(ctx, task), true};
        return {nullptr, false};
    }

    std::vector<std::pair<synnax::task::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::rack::Rack &rack
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
