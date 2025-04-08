// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/pipeline/acquisition.h"
#include "driver/task/task.h"
#include "x/cpp/loop/loop.h"

namespace meminfo {
std::uint32_t getUsage();


class MemInfoSource final : public pipeline::Source {
    synnax::ChannelKey key;
    loop::Timer timer;

public:
    explicit MemInfoSource(const synnax::ChannelKey &key):
        key(key), timer(telem::HZ * 1) {}

    xerrors::Error read(breaker::Breaker &breaker, synnax::Frame &data) override {
        timer.wait(breaker);
        if (data.empty()) data.emplace(this->key, telem::Series(telem::UINT32_T, 1));
        auto &s = data.series->at(0);
        s.set(0, getUsage());
        return xerrors::NIL;
    }
};

class MemInfo final : public task::Task {
    pipeline::Acquisition pipe;

public:
    MemInfo(
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

    std::string name() override { return "meminfo"; }

    static std::unique_ptr<task::Task>
    configure(const std::shared_ptr<task::Context> &ctx, const synnax::Task &task) {
        auto ch_name = "sy_rack" +
                       std::to_string(
                           synnax::rack_key_node(synnax::task_key_rack(task.key))
                       ) +
                       "_meminfo";
        auto [ch, err] = ctx->client->channels.retrieve(ch_name);
        if (err.matches(xerrors::NOT_FOUND)) {
            ch = synnax::Channel(ch_name, telem::UINT32_T, true);
            auto new_err = ctx->client->channels.create(ch);
        }
        auto source = std::make_shared<MemInfoSource>(ch.key);
        auto writer_cfg = synnax::WriterConfig{
            .channels = {ch.key},
            .start = telem::TimeStamp::now()
        };
        return std::make_unique<MemInfo>(
            ctx,
            source,
            writer_cfg,
            breaker::default_config(task.name)
        );
    }

    void stop(bool will_reconfigure) override { pipe.stop(); }
};

class Factory final : public task::Factory {
    std::pair<std::unique_ptr<task::Task>, bool> configure_task(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) override {
        if (task.type == "meminfo") return {MemInfo::configure(ctx, task), true};
        return {nullptr, false};
    }

    std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>>
    configure_initial_tasks(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Rack &rack
    ) override {
        std::vector<std::pair<synnax::Task, std::unique_ptr<task::Task>>> tasks;
        auto [existing, err] = rack.tasks.retrieve_by_type("meminfo");
        if (err.matches(xerrors::NOT_FOUND)) {
            auto sy_task = synnax::Task(rack.key, "meminfo", "meminfo", "", true);
            err = rack.tasks.create(sy_task);
            if (err) {
                LOG(ERROR) << "[meminfo] failed to retrieve meminfo task: " << err;
                return {};
            }
            auto [task, ok] = configure_task(ctx, sy_task);
            if (ok && task != nullptr) tasks.emplace_back(sy_task, std::move(task));
        } else if (err) {
            LOG(ERROR) << "[meminfo] failed to retrieve existing tasks: " << err;
            return {};
        }
        return tasks;
    }
};
}
