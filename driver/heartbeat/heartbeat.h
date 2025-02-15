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
const auto EMISSION_RATE = telem::Rate(1);

/// @brief uint64 heartbeat value that communicates the aliveness of a rack. The
/// first 32 bits are the rack key and the last 32 bits are the version.
typedef std::uint64_t Heartbeat;

/// @brief creates a new heartbeat value from its components.
inline Heartbeat create(const RackKey rack_key, const std::uint32_t version) {
    return static_cast<std::uint64_t>(rack_key) << 32 | version;
}

/// @brief retrieves the rack key from the heartbeat value.
inline Heartbeat rack_key(const std::uint64_t hb) { return hb >> 32; }

/// @brief retrieves the version from the heartbeat value.
inline Heartbeat version(const std::uint64_t hb) { return hb & 0xFFFFFFFF; }

class Source final : public pipeline::Source {
    /// @brief the key of the heartbeat channel.
    const synnax::ChannelKey key;
    /// @brief the key of the rack the heartbeat is for.
    const RackKey rack_key;
    /// @brief the current heartbeat version incremented on every loop iteration.
    std::uint32_t version;
    /// @brief the loop used to control the emission rate of the heartbeat.
    loop::Timer loop;

public:
    Source(const synnax::ChannelKey key, const RackKey rack_key) : key(key),
        rack_key(rack_key),
        version(0),
        loop(loop::Timer(EMISSION_RATE)) {
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        this->loop.wait(breaker);
        const Heartbeat hb = create(this->rack_key, this->version);
        this->version++;
        const auto fr = Frame(key, telem::Series(hb));
        return {Frame(key, telem::Series(hb)), xerrors::NIL};
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
    ) : pipe(pipeline::Acquisition(
        ctx->client,
        writer_config,
        std::move(source),
        breaker_config
    )) {
        pipe.start();
    }

    /// @brief implements task::Task.
    std::string name() override { return "heartbeat"; }

    /// @brief stop the heartbeat process
    void stop() override { pipe.stop(); }

    /// @brief configures the heartbeat task.
    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        auto [ch, err] = ctx->client->channels.retrieve(RACK_HEARTBEAT_CHANNEL);
        if (err) {
            LOG(WARNING) << "[heartbeat] failed to retrieve heartbeat channel: " << err;
            return nullptr;
        }
        auto source = std::make_shared<Source>(
            ch.key,
            synnax::task_key_rack(task.key)
        );
        auto writer_cfg = synnax::WriterConfig{
            .start = telem::TimeStamp::now(),
            .channels = {ch.key},
        };
        auto breaker_config = breaker::default_config(task.name);
        return std::make_unique<Task>(ctx, source, writer_cfg, breaker_config);
    }
};

class Factory final : public task::Factory {
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type == "heartbeat")
            return {Task::configure(ctx, task), true};
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
