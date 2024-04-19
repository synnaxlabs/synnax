// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver//task//task.h"
#include "driver/pipeline/acquisition.h"

namespace meminfo {
std::uint32_t getUsage();


class MemInfoSource final : public pipeline::Source {
    synnax::ChannelKey key;
public:

    MemInfoSource(
        const synnax::ChannelKey &key
    ): key(key) {
    }

    std::pair<Frame, freighter::Error> read() override {
        std::this_thread::sleep_for(std::chrono::seconds(1));
        auto fr = Frame(1);
        fr.add(key, Series(getUsage(), synnax::UINT32));
        return {std::move(fr), freighter::NIL};
    }
};

class MemInfo final : public task::Task {
    pipeline::Acquisition pipe;

public:
    MemInfo(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        auto ch_name = "sy_rack" + std::to_string(rackKeyNode(taskKeyRack(task.key))) + "_meminfo";
        auto [ch, err] = ctx->client->channels.retrieve(ch_name);
        if (err.matches(synnax::NOT_FOUND)) {
            ch = synnax::Channel(
                ch_name,
                synnax::UINT32,
                true
            );
            auto new_err = ctx->client->channels.create(ch);
        }
        auto source = std::make_unique<MemInfoSource>(ch.key);

        auto writer_cfg = synnax::WriterConfig{
            .channels = {ch.key},
            .start = TimeStamp::now(),
        };

        auto breaker_config = breaker::Config{
            .name = task.name,
            .base_interval = 1 * SECOND,
            .max_retries = 20,
            .scale = 1.2
        };

        pipe = pipeline::Acquisition(
            ctx,
            writer_cfg,
            std::move(source),
            breaker_config
        );
        pipe.start();
    }

    void exec(task::Command &cmd) override {
    };

    void stop() override {
        pipe.stop();
    };
};

class Factory final : public task::Factory {
    std::pair<std::unique_ptr<task::Task>, bool> configureTask(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        // if (task.type == "meminfo")
        //     return {std::make_unique<MemInfo>(ctx, task), true};
        return {nullptr, false};
    }

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > >
    configureInitialTasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override {
        std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task> > > tasks;
        auto [existing, err] = rack.tasks.list();
        if (err) {
            LOG(ERROR) << "[Meminfo] Failed to list existing tasks: " << err;
            return {};
        }
        bool hasMeminfo = false;
        for (const auto &t: existing) {
            if (t.type == "meminfo") {
                LOG(INFO) << "[Meminfo] found existing meminfo task with key: " << t.key
                        << "skipping creation." << std::endl;
                hasMeminfo = true;
            }
        }

        if (!hasMeminfo) {
            auto sy_task = synnax::Task(
                rack.key,
                "meminfo",
                "meminfo",
                ""
            );
            auto err = rack.tasks.create(sy_task);
            LOG(INFO) << "[Meminfo] created meminfo task with key: " << sy_task.key;
            if (err) {
                LOG(ERROR) << "[Meminfo] Failed to create meminfo task: " << err;
                return {};
            }
            auto [task, ok] = configureTask(ctx, sy_task);
            if (ok) tasks.push_back({sy_task, std::move(task)});
        }
        return tasks;
    }
};
}
