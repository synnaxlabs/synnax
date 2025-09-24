// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// internal
#include "driver/pipeline/acquisition.h"
#include "driver/task/task.h"

/// module
#include "client/cpp/synnax.h"
#include "driver/task/common/factory.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/status/status.h"

namespace rack::status {
const std::string INTEGRATION_NAME = "rack_status";
const std::string LEGACY_HEARTBEAT_TYPE = "heartbeat";
const std::string TASK_NAME = "Rack State";
const std::string TASK_TYPE = TASK_NAME;
const auto EMISSION_RATE = telem::HZ * 1;

class Source final : public pipeline::Source {
    /// @brief the key of the heartbeat channel.
    const synnax::ChannelKey key;
    /// @brief the key of the rack the heartbeat is for.
    const synnax::RackKey rack_key;
    /// @brief the loop used to control the emission rate of the heartbeat.
    loop::Timer loop;

public:
    Source(const synnax::ChannelKey key, const synnax::RackKey rack_key):
        key(key), rack_key(rack_key), loop(loop::Timer(EMISSION_RATE)) {}

    xerrors::Error read(breaker::Breaker &breaker, synnax::Frame &fr) override {
        fr.clear();
        this->loop.wait(breaker);
        const synnax::RackStatus status{
            .variant = ::status::variant::SUCCESS,
            .message = "Driver is running",
            .details = synnax::RackStatusDetails{
                .rack = this->rack_key,
            }
        };
        VLOG(1) << "[rack_state] emitting state for rack " << this->rack_key;
        fr.emplace(key, telem::Series(status.to_json()));
        return xerrors::NIL;
    }
};

/// @brief a task that periodically
/// to indicate that the driver is still alive.
class Task final : public task::Task {
    pipeline::Acquisition pipe;

public:
    Task(
        const std::shared_ptr<task::Context> &ctx,
        std::shared_ptr<pipeline::Source> source,
        const synnax::WriterConfig &writer_config,
        const breaker::Config &breaker_config
    ):
        pipe(
            pipeline::Acquisition(
                ctx->client,
                writer_config,
                std::move(source),
                breaker_config
            )
        ) {
        pipe.start();
    }

    /// @brief implements task::Task.
    std::string name() const override { return TASK_NAME; }

    /// @brief stop the heartbeat process
    void stop(bool will_reconfigure) override { pipe.stop(); }

    /// @brief configures the heartbeat task.
    static std::unique_ptr<task::Task>
    configure(const std::shared_ptr<task::Context> &ctx, const synnax::Task &task) {
        auto [ch, err] = ctx->client->channels.retrieve(
            synnax::RACK_STATUS_CHANNEL_NAME
        );
        if (err) {
            LOG(WARNING) << "[rack_state] failed to retrieve rack state channel: "
                         << err;
            return nullptr;
        }
        auto source = std::make_shared<Source>(
            ch.key,
            synnax::rack_key_from_task_key(task.key)
        );
        auto writer_cfg = synnax::WriterConfig{
            .channels = {ch.key},
            .start = telem::TimeStamp::now(),
        };
        auto breaker_config = breaker::Config{
            .name = TASK_NAME,
            .base_interval = 1 * telem::SECOND,
            .max_retries = breaker::RETRY_INFINITELY,
            .scale = 1.05,
            .max_interval = 5 * telem::SECOND,
        };
        return std::make_unique<Task>(ctx, source, writer_cfg, breaker_config);
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
