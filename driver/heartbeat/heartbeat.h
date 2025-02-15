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

namespace heartbeat {
const std::string RACK_HEARTBEAT_CHANNEL = "sy_rack_heartbeat";
const std::string INTEGRATION_NAME = "heartbeat";

class HeartbeatSource final : public pipeline::Source {
    synnax::ChannelKey key;
    RackKey rack_key;
    std::uint32_t version;
    loop::Timer timer;

public:
    HeartbeatSource(
        const synnax::ChannelKey key,
        const RackKey rack_key
    ) : key(key),
        rack_key(rack_key),
        version(0),
        timer(loop::Timer(telem::Rate(1))) {
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        this->timer.wait(breaker);
        const auto heartbeat = static_cast<std::uint64_t>(this->rack_key) << 32 | this->
                               version;
        this->version++;
        return {Frame(key, telem::Series(heartbeat)), xerrors::NIL};
    }
};

/// @brief a task that periodically
/// to indicate that the driver is still alive.
class Heartbeat final : public task::Task {
    pipeline::Acquisition pipe;

public:
    Heartbeat(
        const std::shared_ptr<task::Context> &ctx,
        std::shared_ptr<pipeline::Source> source,
        const synnax::WriterConfig &writer_config,
        const breaker::Config &breaker_config
    ) : pipe(pipeline::Acquisition(
        ctx->client,
        writer_config,
        std::move(source),
        breaker_config
    )) {
        pipe.start();
    }

    std::string name() override { return "heartbeat"; }

    /// @brief starts the heartbeat process
    /// @param done a flag that is set to true when the heartbeat process exits.
    xerrors::Error start(std::atomic<bool> &done);

    /// @brief stop the heartbeat process
    void stop() override { pipe.stop(); }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        auto [ch, err] = ctx->client->channels.retrieve(RACK_HEARTBEAT_CHANNEL);
        if (err.matches(xerrors::NOT_FOUND)) return nullptr;
        auto source = std::make_shared<HeartbeatSource>(
            ch.key,
            synnax::task_key_rack(task.key)
        );
        auto writer_cfg = synnax::WriterConfig{
            .channels = {ch.key},
            .start = telem::TimeStamp::now(),
        };
        auto breaker_config = breaker::default_config(task.name);
        return std::make_unique<Heartbeat>(ctx, source, writer_cfg, breaker_config);
    }
};

class Factory final : public task::Factory {
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type == "heartbeat")
            return {Heartbeat::configure(ctx, task), true};
        return {nullptr, false};
    }

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override {
        std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;
        auto [existing, err] = rack.tasks.retrieveByType("heartbeat");
        if (err.matches(xerrors::NOT_FOUND)) {
            auto sy_task = synnax::Task(
                rack.key,
                "heartbeat",
                "heartbeat",
                "",
                true
            );
            err = rack.tasks.create(sy_task);
            if (err)
                LOG(ERROR) << "failed to create heartbeat task: " << err;
            auto [task, ok] = configure_task(ctx, sy_task);
            if (ok && task != nullptr) tasks.emplace_back(sy_task, std::move(task));
        } else if (err)
            LOG(ERROR) << "failed to retrieve heartbeat task: " << err;
        return tasks;
    }
};
}
