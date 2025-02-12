// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/reader.h"
#include "driver/labjack/writer.h"

///////////////////////////////////////////////////////////////////////////////////
//                                    ReaderTask                                 //
///////////////////////////////////////////////////////////////////////////////////
labjack::ReaderTask::ReaderTask(
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task,
    std::shared_ptr<labjack::ReaderSource> labjack_source,
    std::shared_ptr<pipeline::Source> source,
    synnax::WriterConfig writer_config,
    const breaker::Config breaker_config
) : ctx(ctx),
    task(task),
    read_pipe(
        pipeline::Acquisition(
            ctx->client,
            writer_config,
            source,
            breaker_config)
    ),
    source(labjack_source) {
    std::vector<synnax::ChannelKey> channel_keys = labjack_source->get_ai_channel_keys();
    this->tare_mw = std::make_shared<pipeline::TareMiddleware>(channel_keys);
    read_pipe.add_middleware(tare_mw);

    auto parser = config::Parser(task.config);
    auto scale_mw = std::make_shared<pipeline::ScaleMiddleware>(parser);
    read_pipe.add_middleware(scale_mw);
}

void labjack::ReaderTask::exec(task::Command &cmd) {
    if (cmd.type == "start")
        this->start(cmd.key);
    else if (cmd.type == "stop")
        this->stop(cmd.key);
    else if (cmd.type == "tare") {
        this->tare_mw->tare(cmd.args);
        LOG(INFO) << "[labjack.task] tare command received for task " << this->task.name;
    }
}

void labjack::ReaderTask::stop(const std::string &cmd_key) {
    if (!this->running.exchange(false)) return;
    this->read_pipe.stop();
    this->source->stop(cmd_key);
    if (this->source->ok())
        LOG(INFO) << "[labjack.task] successfully stopped task " << this->task.name;
}

void labjack::ReaderTask::stop() { this->stop(""); }


void labjack::ReaderTask::start(const std::string &cmd_key) {
    if (this->running.exchange(true)) return;
    this->source->start(cmd_key);
    this->read_pipe.start();
    if (this->source->ok())
        LOG(INFO) << "[labjack.task] successfully started task " << this->task.name;
}

std::unique_ptr<task::Task> labjack::ReaderTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task,
    std::shared_ptr<labjack::DeviceManager> device_manager
) {
    VLOG(2) << "[labjack.task] configuring task " << task.name;

    auto breaker_config = breaker::default_config(task.name);

    auto parser = config::Parser(task.config);
    ReaderConfig reader_config(parser);

    auto control_subject = synnax::ControlSubject{
            .name = task.name,
            .key =  task.name + "-" +  std::to_string(task.key)
    };

    auto source = std::make_shared<labjack::ReaderSource>(
        ctx,
        task,
        reader_config,
        device_manager
    );

    std::vector<synnax::ChannelKey> channel_keys = source->get_channel_keys();

    auto writer_config = synnax::WriterConfig{
        .channels = channel_keys,
        .start = telem::TimeStamp::now(),
        .subject = control_subject,
        .mode = reader_config.data_saving
                    ? synnax::WriterMode::PersistStream
                    : synnax::WriterMode::StreamOnly,
        .enable_auto_commit = true
    };


    auto p = std::make_unique<labjack::ReaderTask>(
        ctx,
        task,
        source,
        source,
        writer_config,
        breaker_config
    );

    if (!source->ok()) return nullptr;

    ctx->set_state({
        .task = task.key,
        .variant = "success",
        .details = {
            {"running", false},
            {"message", "Successfully configured task"}
        }
    });

    LOG(INFO) << "[labjack.task] successfully configured task " << task.name;
    return p;
}
