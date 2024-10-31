// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/reader.h"
#include "driver/labjack/writer.h"
#include "driver/labjack/task.h"

///////////////////////////////////////////////////////////////////////////////////
//                                    ReaderTask                                 //
///////////////////////////////////////////////////////////////////////////////////
labjack::ReaderTask::ReaderTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        std::shared_ptr<labjack::Source> labjack_source,
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
        source(labjack_source){
}

void labjack::ReaderTask::exec(task::Command &cmd) {
    if (cmd.type == "start") {
        this->start(cmd.key);
    } else if (cmd.type == "stop") {
        this->stop(cmd.key);
    } else {
        LOG(ERROR) << "unknown command type: " << cmd.type;
    }
}

void labjack::ReaderTask::stop(const std::string &cmd_key) {
    if(!this->running.exchange(false)) return;
    this->read_pipe.stop();
    this->source->stop(cmd_key);
    if(this->source->ok())
        LOG(INFO) << "[labjack.task] successfully stopped task " << this->task.name;
}

void labjack::ReaderTask::stop() { this->stop("");}


void labjack::ReaderTask::start(const std::string &cmd_key){
    if(this->running.exchange(true)) return;
    this->source->start(cmd_key);
    this->read_pipe.start();
    if(this->source->ok())
        LOG(INFO) << "[labjack.task] successfully started task " << this->task.name;
}

std::unique_ptr<task::Task> labjack::ReaderTask::configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task){
    LOG(INFO) << "[labjack.task] configuring task " << task.name;

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 *SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };

    auto parser = config::Parser(task.config);
    ReaderConfig reader_config(parser);

    auto source = std::make_shared<labjack::Source>(
        ctx,
        task,
        reader_config
    );

    std::vector<synnax::ChannelKey> channel_keys = source->get_channel_keys();

    auto writer_config = synnax::WriterConfig{
        .channels = channel_keys,
        .start = synnax::TimeStamp::now(),
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

    // TODO: change setState to snake case
    ctx->setState({
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


///////////////////////////////////////////////////////////////////////////////////
//                                    WriterTask                                 //
///////////////////////////////////////////////////////////////////////////////////
labjack::WriterTask::WriterTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        std::shared_ptr<pipeline::Sink> sink,
        std::shared_ptr<labjack::WriteSink> labjack_sink,
        std::shared_ptr<pipeline::Source> state_source,
        synnax::WriterConfig writer_config,
        synnax::StreamerConfig streamer_config,
        const breaker::Config breaker_config
) : ctx(ctx),
    task(task),
    cmd_pipe(
        pipeline::Control(
            ctx->client,
            streamer_config,
            sink,
            breaker_config
        )
    ),
    state_pipe(
        pipeline::Acquisition(
            ctx->client,
            writer_config,
            state_source,
            breaker_config
        )
    ),
    sink(labjack_sink){
}

std::unique_ptr <task::Task> labjack::WriterTask::configure(
        const std::shared_ptr <task::Context> &ctx,
        const synnax::Task &task
){

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2
    };

    auto parser = config::Parser(task.config);
    WriterConfig writer_config(parser);

    auto sink = std::make_shared<labjack::WriteSink>(
        ctx,
        task,
        writer_config
    );

    std::vector<synnax::ChannelKey> cmd_keys = sink->get_cmd_channel_keys();
    std::vector<synnax::ChannelKey> state_keys = sink->get_state_channel_keys();

    auto state_writer_config = synnax::WriterConfig{
        .channels = state_keys,
        .start = synnax::TimeStamp::now(),
        .mode = writer_config.data_saving
                ? synnax::WriterMode::PersistStream
                : synnax::WriterMode::StreamOnly,
        .enable_auto_commit = true
    };

    auto cmd_streamer_config = synnax::StreamerConfig{
        .channels = cmd_keys,
    };

    auto state_source = sink->state_source;

    auto p = std::make_unique<labjack::WriterTask>(
        ctx,
        task,
        sink,
        sink,
        state_source,
        state_writer_config,
        cmd_streamer_config,
        breaker_config
    );

    ctx->setState({
                          .task = task.key,
                          .variant = "success",
                          .details = {
                                  {"running", false},
                                  {"message", "Successfully configured task"}
                          }
                  });

    LOG(INFO) << "[labjack.writer] successfully configured task " << task.name;
    return p;
}

void labjack::WriterTask::exec(task::Command &cmd){
    if(cmd.type == "start") this->start(cmd.key);
    else if(cmd.type == "stop") this->stop(cmd.key);
    else LOG(ERROR) << "unknown command type: " << cmd.type;
}

void labjack::WriterTask::start(const std::string &key){
    if(this->running.exchange(true)) return;
    sink->start(key);
    this->cmd_pipe.start();
    this->state_pipe.start();
}

void labjack::WriterTask::stop() { this->stop(""); }

void labjack::WriterTask::stop(const std::string &key){
    if(!this->running.exchange(false)) return;
    this->cmd_pipe.stop();
    this->state_pipe.stop();
    this->sink->stop(key);
}