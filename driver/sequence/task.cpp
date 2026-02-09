// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/loop/loop.h"
#include "x/cpp/status/status.h"
#include "x/cpp/thread/thread.h"

#include "driver/sequence/sequence.h"

namespace driver::sequence {
Task::Task(
    const std::shared_ptr<task::Context> &ctx,
    synnax::task::Task task,
    TaskConfig cfg,
    std::unique_ptr<Sequence> seq,
    const x::breaker::Config &breaker_config
):
    cfg(std::move(cfg)),
    task(std::move(task)),
    breaker(breaker_config),
    ctx(ctx),
    seq(std::move(seq)),
    status(
        synnax::task::Status{
            .key = task.status_key(),
            .variant = x::status::variant::SUCCESS,
            .details = synnax::task::StatusDetails{
                .task = task.key,
                .running = false,
            }
        }
    ) {}

void Task::run() {
    x::thread::set_name(this->task.name.c_str());
    if (const auto err = this->seq->begin(); err) {
        if (const auto end_err = this->seq->end())
            LOG(ERROR) << "[sequence] failed to end after failed start:" << end_err;
        this->status.variant = x::status::variant::ERR;
        this->status.details.running = false;
        this->status.message = err.message();
        return ctx->set_status(status);
    }
    this->status.variant = x::status::variant::SUCCESS;
    this->status.details.running = true;
    this->status.message = "Sequence started";
    this->ctx->set_status(this->status);
    x::loop::Timer timer(this->cfg.rate);
    while (this->breaker.running()) {
        if (const auto next_err = this->seq->next()) {
            this->status.variant = x::status::variant::ERR;
            this->status.message = next_err.message();
            break;
        }
        auto [elapsed, ok] = timer.wait(this->breaker);
        if (!ok) {
            this->status.variant = x::status::variant::WARNING;
            this->status.message = "Sequence script is executing too slowly for the "
                                   "configured loop rate. Last execution took " +
                                   elapsed.to_string();
            this->ctx->set_status(this->status);
        }
    }
    if (const auto end_err = this->seq->end()) {
        this->status.variant = x::status::variant::ERR;
        this->status.message = end_err.message();
    }
    this->status.details.running = false;
    if (this->status.variant == x::status::variant::ERR)
        return this->ctx->set_status(this->status);
    this->status.variant = x::status::variant::SUCCESS;
    this->status.message = "Sequence stopped";
}

void Task::stop(bool will_reconfigure) {
    this->stop("", will_reconfigure);
}

void Task::exec(task::Command &cmd) {
    if (cmd.type == "start") return this->start(cmd.key);
    if (cmd.type == "stop") return this->stop(cmd.key, false);
}

void Task::start(const std::string &key) {
    if (this->breaker.running()) return;
    this->breaker.reset();
    this->breaker.start();
    this->status.key = key;
    this->thread = std::thread([this] { this->run(); });
}

void Task::stop(const std::string &key, bool will_reconfigure) {
    if (!this->breaker.running()) return;
    this->breaker.stop();
    this->breaker.reset();
    if (this->thread.joinable()) this->thread.join();
    this->status.key = key;
    this->ctx->set_status(this->status);
}

std::unique_ptr<task::Task> Task::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::task::Task &task
) {
    synnax::task::Status cfg_status;
    cfg_status.details.task = task.key;

    auto parser = x::json::Parser(task.config);
    TaskConfig cfg(parser);
    if (!parser.ok()) {
        LOG(ERROR) << "[sequence] failed to parse task configuration: "
                   << parser.error();
        cfg_status.variant = x::status::variant::ERR;
        cfg_status.details.data = parser.error_json();
        ctx->set_status(cfg_status);
        return nullptr;
    }

    auto json_plugin = std::make_shared<plugins::JSON>(cfg.globals);
    auto time_plugin = std::make_shared<plugins::Time>();
    std::vector<std::shared_ptr<plugins::Plugin>> plugins_list{
        json_plugin,
        time_plugin
    };

    if (!cfg.read.empty()) {
        auto [read_channels, r_err] = ctx->client->channels.retrieve(cfg.read);
        if (r_err) {
            LOG(ERROR) << "[sequence] failed to retrieve read channels: " << r_err;
            cfg_status.variant = x::status::variant::ERR;
            cfg_status.details.running = false;
            cfg_status.message = r_err.message();
            return nullptr;
        }
        auto ch_receive_plugin = std::make_shared<plugins::ChannelReceive>(
            ctx->client,
            read_channels
        );
        plugins_list.push_back(ch_receive_plugin);
    }

    if (!cfg.write.empty()) {
        auto [write_channels, w_err] = ctx->client->channels.retrieve(cfg.write);
        if (w_err) {
            LOG(ERROR) << "[sequence] failed to retrieve write channels: " << w_err;
            cfg_status.variant = x::status::variant::ERR;
            cfg_status.details.running = false;
            cfg_status.message = w_err.message();
            return nullptr;
        }
        for (const auto &ch: write_channels)
            if (!ch.is_virtual &&
                std::find(cfg.write.begin(), cfg.write.end(), ch.index) ==
                    cfg.write.end())
                cfg.write.push_back(ch.index);

        const synnax::framer::WriterConfig writer_cfg {
            .channels = cfg.write, .start = x::telem::TimeStamp::now(),
            .authorities = {cfg.authority},
            .subject = x::control::Subject{
                .name = task.name,
                .key = std::to_string(task.key),
            }
        };
        auto sink = std::make_shared<plugins::SynnaxFrameSink>(ctx->client, writer_cfg);
        auto ch_write_plugin = std::make_shared<plugins::ChannelWrite>(
            sink,
            write_channels
        );
        plugins_list.push_back(ch_write_plugin);
    }

    auto breaker_config = x::breaker::default_config("sequence (" + task.name + ")");
    auto seq = std::make_unique<Sequence>(
        std::make_shared<plugins::MultiPlugin>(plugins_list),
        cfg.script
    );
    if (const auto compile_err = seq->compile(); compile_err) {
        cfg_status.variant = x::status::variant::ERR;
        cfg_status.details.running = false;
        cfg_status.message = compile_err.message();
        ctx->set_status(cfg_status);
        return nullptr;
    }

    cfg_status.variant = x::status::variant::SUCCESS;
    cfg_status.details.running = false;
    cfg_status.message = "Sequence configured successfully";
    ctx->set_status(cfg_status);
    return std::make_unique<Task>(ctx, task, cfg, std::move(seq), breaker_config);
}
}
