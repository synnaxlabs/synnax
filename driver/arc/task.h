// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <vector>

#include "client/cpp/arc/arc.h"
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/json/json.h"

#include "arc/cpp/module/module.h"
#include "arc/cpp/runtime/errors/errors.h"
#include "arc/cpp/runtime/loop/loop.h"
#include "arc/cpp/runtime/runtime.h"
#include "arc/cpp/runtime/state/state.h"
#include "driver/arc/arc.h"
#include "driver/common/common.h"
#include "driver/common/status.h"
#include "driver/errors/errors.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

namespace driver::arc {
/// @brief configuration for an arc runtime task.
struct TaskConfig : common::BaseTaskConfig {
    std::string arc_key;
    ::arc::module::Module module;
    ::arc::runtime::loop::Config loop;

    TaskConfig(TaskConfig &&other) noexcept:
        BaseTaskConfig(std::move(other)),
        arc_key(std::move(other.arc_key)),
        module(std::move(other.module)),
        loop(std::move(other.loop)) {}

    TaskConfig(const TaskConfig &) = delete;
    const TaskConfig &operator=(const TaskConfig &) = delete;

    explicit TaskConfig(x::json::Parser &parser):
        BaseTaskConfig(parser),
        arc_key(parser.field<std::string>("arc_key")),
        loop(parser) {}

    static std::pair<TaskConfig, x::errors::Error>
    parse(const std::shared_ptr<synnax::Synnax> &client, x::json::Parser &parser) {
        auto cfg = TaskConfig(parser);
        if (!parser.ok()) return {std::move(cfg), parser.error()};
        auto [arc_data, arc_err] = client->arcs.retrieve_by_key(
            cfg.arc_key,
            synnax::arc::RetrieveOptions{.compile = true}
        );
        if (arc_err) return {std::move(cfg), arc_err};
        cfg.module = ::arc::module::Module(arc_data.module);
        return {std::move(cfg), x::errors::NIL};
    }
};

/// @brief arc runtime task that manages both read and write pipelines.
class Task final : public task::Task {
    std::shared_ptr<::arc::runtime::Runtime> runtime;
    std::unique_ptr<pipeline::Acquisition> acquisition;
    std::unique_ptr<pipeline::Control> control;
    common::StatusHandler state;

    /// @brief source that reads output data from the arc runtime.
    class Source final : public pipeline::Source {
        Task &task;

    public:
        explicit Source(Task &task): task(task) {}

        x::errors::Error read(
            x::breaker::Breaker &breaker,
            x::telem::Frame &fr,
            pipeline::Authorities &authorities
        ) override {
            ::arc::runtime::Output out;
            if (!this->task.runtime->read(out)) return errors::NOMINAL_SHUTDOWN_ERROR;
            fr = std::move(out.frame);
            for (auto &c: out.authority_changes) {
                if (c.channel_key.has_value())
                    authorities.keys.push_back(*c.channel_key);
                authorities.authorities.push_back(c.authority);
            }
            return x::errors::NIL;
        }

        void stopped_with_err(const x::errors::Error &err) override {
            this->task.state.error(err);
            this->task.stop("", true);
        }
    };

    /// @brief sink that receives input data from Synnax.
    class Sink final : public pipeline::Sink {
        Task &task;

    public:
        explicit Sink(Task &task): task(task) {}

        x::errors::Error write(x::telem::Frame &frame) override {
            if (frame.empty()) return x::errors::NIL;
            return this->task.runtime->write(std::move(frame));
        }
    };

    Task(
        const synnax::task::Task &task_meta,
        const std::shared_ptr<task::Context> &ctx
    ):
        state(ctx, task_meta) {}

public:
    static std::pair<std::unique_ptr<Task>, x::errors::Error> create(
        const synnax::task::Task &task_meta,
        const std::shared_ptr<task::Context> &ctx,
        const TaskConfig &cfg,
        std::shared_ptr<pipeline::WriterFactory> writer_factory = nullptr,
        std::shared_ptr<pipeline::StreamerFactory> streamer_factory = nullptr
    ) {
        auto task = std::unique_ptr<Task>(new Task(task_meta, ctx));

        const ::arc::runtime::Config runtime_cfg{
            .mod = cfg.module,
            .breaker = x::breaker::default_config("arc_runtime"),
            .retrieve_channels = [client = ctx->client](
                                     const std::vector<::arc::types::ChannelKey> &keys
                                 )
                -> std::pair<
                    std::vector<::arc::runtime::state::ChannelDigest>,
                    x::errors::Error> {
                auto [channels, err] = client->channels.retrieve(keys);
                if (err) return {{}, err};
                std::vector<::arc::runtime::state::ChannelDigest> digests;
                for (const auto &ch: channels)
                    digests.push_back({ch.key, ch.data_type, ch.index});
                return {digests, x::errors::NIL};
            },
            .loop = cfg.loop,
        };

        auto [rt, err] = ::arc::runtime::load(
            runtime_cfg,
            [task_ptr = task.get()](const x::errors::Error &err) {
                if (err.matches(::arc::runtime::errors::WARNING))
                    task_ptr->state.send_warning(err);
                else {
                    task_ptr->state.send_error(err);
                    task_ptr->runtime->close_outputs();
                }
            }
        );
        if (err) return {nullptr, err};

        task->runtime = std::move(rt);

        auto source = std::make_unique<Source>(*task);
        auto sink = std::make_unique<Sink>(*task);
        if (!writer_factory)
            writer_factory = std::make_shared<pipeline::SynnaxWriterFactory>(
                ctx->client
            );
        if (!streamer_factory)
            streamer_factory = std::make_shared<pipeline::SynnaxStreamerFactory>(
                ctx->client
            );
        auto initial_authorities = ::arc::runtime::build_authorities(
            cfg.module.authorities,
            task->runtime->write_channels
        );
        task->acquisition = std::make_unique<pipeline::Acquisition>(
            writer_factory,
            synnax::framer::WriterConfig{
                .channels = task->runtime->write_channels,
                .start = x::telem::TimeStamp::now(),
                .authorities = std::move(initial_authorities),
                .subject =
                    x::telem::ControlSubject{
                        .name = task_meta.name,
                        .key = std::to_string(task_meta.key),
                    },
                .mode = common::data_saving_writer_mode(cfg.data_saving),
            },
            std::move(source),
            x::breaker::default_config("arc_acquisition"),
            "arc_acquisition"
        );
        task->control = std::make_unique<pipeline::Control>(
            streamer_factory,
            synnax::framer::StreamerConfig{.channels = task->runtime->read_channels},
            std::move(sink),
            x::breaker::default_config("arc_control"),
            "arc_control"
        );

        return {std::move(task), x::errors::NIL};
    }

    bool start(const std::string &cmd_key) {
        const auto runtime_started = this->runtime->start();
        const auto acq_started = this->acquisition->start();
        const auto control_started = this->control->start();
        this->state.send_start(cmd_key);
        return acq_started && control_started && runtime_started;
    }

    bool stop(const std::string &cmd_key, const bool propagate_state) {
        const auto control_stopped = this->control->stop();
        this->runtime->close_outputs();
        const auto acq_stopped = this->acquisition->stop();
        const auto runtime_stopped = this->runtime->stop();
        if (propagate_state) this->state.send_stop(cmd_key);
        return control_stopped && acq_stopped && runtime_stopped;
    }

    void exec(task::Command &cmd) override {
        if (cmd.type == "start")
            this->start(cmd.key);
        else if (cmd.type == "stop")
            this->stop(false);
        else
            LOG(WARNING) << "[arc] unknown command type: " << cmd.type;
    }

    void stop(const bool will_reconfigure) override {
        this->stop("", !will_reconfigure);
    }

    [[nodiscard]] std::string name() const override { return "Arc Runtime Task"; }
};
}
