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
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"

namespace rack::state {
const std::string RACK_STATE_CHANNEL = "sy_rack_state";
const std::string INTEGRATION_NAME = "heartbeat";
const std::string TASK_NAME = "Heartbeat";
const std::string TASK_TYPE = INTEGRATION_NAME;
const auto EMISSION_RATE = telem::HZ * 1;

struct State {
    synnax::RackKey key;
    std::string variant;
    std::string message;

    [[nodiscard]] json to_json() const {
        return json{
            {"key", this->key},
            {"variant", this->variant},
            {"message", this->message},
        };
    }
};

class Source final : public pipeline::Source {
    /// @brief the key of the heartbeat channel.
    const synnax::ChannelKey key;
    /// @brief the key of the rack the heartbeat is for.
    const synnax::RackKey rack_key;
    /// @brief the current heartbeat version incremented on every loop iteration.
    std::uint32_t version;
    /// @brief the loop used to control the emission rate of the heartbeat.
    loop::Timer loop;

public:
    Source(const synnax::ChannelKey key, const synnax::RackKey rack_key):
        key(key), rack_key(rack_key), version(0), loop(loop::Timer(EMISSION_RATE)) {}

    xerrors::Error read(breaker::Breaker &breaker, synnax::Frame &fr) override {
        fr.clear();
        this->loop.wait(breaker);
        const State state{
            .key = this->rack_key,
            .variant = "success",
            .message = "Driver is running"
        };
        fr.emplace(key, telem::Series(state.to_json()));
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
        pipe(pipeline::Acquisition(
            ctx->client,
            writer_config,
            std::move(source),
            breaker_config
        )) {
        pipe.start();
    }

    /// @brief implements task::Task.
    std::string name() override { return TASK_NAME; }

    /// @brief stop the heartbeat process
    void stop(bool will_reconfigure) override { pipe.stop(); }

    /// @brief configures the heartbeat task.
    static std::unique_ptr<task::Task>
    configure(const std::shared_ptr<task::Context> &ctx, const synnax::Task &task) {
        auto [ch, err] = ctx->client->channels.retrieve(RACK_STATE_CHANNEL);
        if (err) {
            LOG(WARNING) << "[heartbeat] failed to retrieve heartbeat channel: " << err;
            return nullptr;
        }
        auto source = std::make_shared<Source>(ch.key, synnax::task_key_rack(task.key));
        auto writer_cfg = synnax::WriterConfig{
            .channels = {ch.key},
            .start = telem::TimeStamp::now(),
        };
        auto breaker_config = breaker::Config{
            .name = "heartbeat",
            .base_interval = 1 * telem::SECOND,
            .max_retries = breaker::RETRY_INFINITELY,
            .scale = 1.05,
            .max_interval = 5 * telem::SECOND,
        };
        return std::make_unique<Task>(ctx, source, writer_cfg, breaker_config);
    }
};

class Factory final : public task::Factory {
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
        std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>> tasks;
        auto [existing, err] = rack.tasks.retrieve_by_type(TASK_TYPE);
        if (err.matches(xerrors::NOT_FOUND)) {
            auto sy_task = synnax::Task(rack.key, TASK_NAME, TASK_TYPE, "", true);
            err = rack.tasks.create(sy_task);
            if (err) LOG(ERROR) << "failed to create heartbeat task: " << err;
            auto [task, ok] = configure_task(ctx, sy_task);
            if (ok && task != nullptr) tasks.emplace_back(sy_task, std::move(task));
        } else if (err)
            LOG(ERROR) << "failed to retrieve heartbeat task: " << err;
        return tasks;
    }
};
}
