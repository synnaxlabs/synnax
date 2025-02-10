// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "x/cpp/loop/loop.h";
#include "driver/sequence/sequence.h"

sequence::Task::Task(
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task,
    TaskConfig cfg,
    std::unique_ptr<sequence::Sequence> seq,
    const breaker::Config &breaker_config
): cfg(std::move(cfg)),
   task(std::move(task)),
   breaker(breaker_config),
   ctx(ctx),
   seq(std::move(seq)),
   state({
       .task = task.key,
       .variant = "success",
       .details = {
           {"running", false},
           {"message", ""}
       }
   }) {
}

void sequence::Task::run() {
    if (const auto err = this->seq->start(); err) {
        this->state.variant = "error";
        this->state.details["running"] = false;
        this->state.details["message"] = err.message();
        return ctx->set_state(state);
    }
    this->state.variant = "success";
    this->state.details["running"] = true;
    this->state.details["message"] = "Sequence started";
    this->ctx->set_state(this->state);
    loop::Timer timer(this->cfg.rate);
    while (this->breaker.running()) {
        if (const auto next_err = this->seq->next()) {
            this->state.variant = "error";
            this->state.details["message"] = next_err.message();
            break;
        }
        timer.wait(this->breaker);
    }
    if (const auto end_err = this->seq->end()) {
        this->state.variant = "error";
        this->state.details["message"] = end_err.message();
    }
    this->state.details["running"] = false;
    if (this->state.variant == "error")
        return this->ctx->set_state(this->state);
    this->state.variant = "success";
    this->state.details["message"] = "Sequence stopped";
}

void sequence::Task::stop() { this->stop(""); }

void sequence::Task::exec(task::Command &cmd) {
    if (cmd.type == "start") return this->start(cmd.key);
    if (cmd.type == "stop") return this->stop(cmd.key);
}

void sequence::Task::start(const std::string &key) {
    if (this->breaker.running()) return;
    this->breaker.reset();
    this->breaker.start();
    this->state.key = key;
    this->thread = std::thread([this] { this->run(); });
}

void sequence::Task::stop(const std::string &key) {
    if (!this->breaker.running()) return;
    this->breaker.stop();
    this->breaker.reset();
    if (this->thread.joinable()) this->thread.join();
    this->state.key = key;
    this->ctx->set_state(this->state);
}

std::unique_ptr<task::Task> sequence::Task::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    task::State cfg_state{.task = task.key};

    LOG(INFO) << "[sequence] configuring task " << task.name;
    auto parser = config::Parser(task.config);
    TaskConfig cfg(parser);
    if (!parser.ok()) {
        LOG(ERROR) << "[sequence] failed to parse task configuration: " << parser.
                error();
        cfg_state.variant = "error";
        cfg_state.details = parser.error_json();
        ctx->set_state(cfg_state);
        return nullptr;
    }

    auto json_plugin = std::make_shared<plugins::JSON>(cfg.globals);
    auto time_plugin = std::make_shared<plugins::Time>();
    std::vector<std::shared_ptr<plugins::Plugin> > plugins_list{
        json_plugin,
        time_plugin
    };

    if (!cfg.read.empty()) {
        auto [read_channels, r_err] = ctx->client->channels.retrieve(cfg.read);
        if (r_err) {
            LOG(ERROR) << "[sequence] failed to retrieve read channels: " << r_err;
            cfg_state.variant = "error";
            cfg_state.details = {
                {"running", false},
                {"message", r_err.message()},
            };
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
            cfg_state.variant = "error";
            cfg_state.details = {
                {"running", false},
                {"message", w_err.message()},
            };
            return nullptr;
        }
        for (const auto &ch: write_channels) 
            if (!ch.is_virtual && std::find(cfg.write.begin(), cfg.write.end(), ch.index) == cfg.write.end())
                cfg.write.push_back(ch.index);

        const synnax::WriterConfig writer_cfg{
            .channels = cfg.write,
            .start = telem::TimeStamp::now(),
            .authorities = {200},
            .subject = synnax::ControlSubject{
                .name = task.name,
                .key = std::to_string(task.key),
            }
        };
        auto sink = std::make_shared<plugins::SynnaxFrameSink>(ctx->client, writer_cfg);
        auto ch_write_plugin = std::make_shared<
            plugins::ChannelWrite>(sink, write_channels);
        plugins_list.push_back(ch_write_plugin);
    }

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * telem::SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };


    auto seq = std::make_unique<sequence::Sequence>(
        std::make_shared<plugins::MultiPlugin>(plugins_list),
        cfg.script
    );
    if (const auto compile_err = seq->compile(); compile_err) {
        cfg_state.variant = "error";
        cfg_state.details = {
            {"running", false},
            {"message", compile_err.message()}
        };
        ctx->set_state(cfg_state);
        return nullptr;
    }

    cfg_state.variant = "success";
    cfg_state.details = {
        {"running", false},
        {"message", "Sequence configured successfully"}
    };
    ctx->set_state(cfg_state);
    return std::make_unique<Task>(ctx, task, cfg, std::move(seq), breaker_config);
}
