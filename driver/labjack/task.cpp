// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/labjack/reader.h"
#include "driver/labjack/task.h"

///////////////////////////////////////////////////////////////////////////////////
//                                    ReaderTask                                 //
///////////////////////////////////////////////////////////////////////////////////
labjack::ReaderTask::ReaderTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
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
                    breaker_config)){
}

void labjack::ReaderTask::exec(task::Command &cmd) {
    if (cmd.type == "start") {
        LOG(INFO) << "[labjack.task] started reader task " << this->task.name;
        this->start(cmd.key);
    } else if (cmd.type == "stop") {
        LOG(INFO) << "[labjack.task] stopped reader task " << this->task.name;
        this->stop(cmd.key);
    } else {
        LOG(ERROR) << "unknown command type: " << cmd.type;
    }
}


void labjack::ReaderTask::stop(const std::string &cmd_key) {
    if(!this->running.exchange(false)) return;
    this->read_pipe.stop();
//    this->source->stop(cmd_key);
    LOG(INFO) << "[labjack.task[ successfully stopped task " << this->task.name;
}

void labjack::ReaderTask::stop() { this->stop("");}


void labjack::ReaderTask::start(const std::string &cmd_key){
    if(this->running.exchange(true)) return; // TODO: add a ok check for the task and source?
//    this->source->start(cmd_key);
    this->read_pipe.start();
    LOG(INFO) << "[labjack.task] successfully started task " << this->task.name;
}

std::unique_ptr<task::Task> labjack::ReaderTask::configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task){
    LOG(INFO) << "[labjack.task] configuring task " << task.name;

    // TODO: consolidate all the palces we use the breaker config to one place
    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 *SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };

    auto parser = config::Parser(task.config);
    ReaderConfig reader_config(parser);//


    LOG(INFO) << "Constructing task \n\n";

    auto source = std::make_shared<labjack::Source>(
        1, // TODO: changel handle
        ctx,
        task,
        reader_config
    );

    std::vector<synnax::ChannelKey> channel_keys = source->get_channel_keys();

    LOG(INFO) << "Creating writer config \n";
    //print channel keys
    for (auto &key : channel_keys) {
        LOG(INFO) << "Channel key: " << key;
    }

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