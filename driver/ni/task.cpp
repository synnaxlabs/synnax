// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source/
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/ni.h"
#include <cassert>
#include <stdio.h>

///////////////////////////////////////////////////////////////////////////////////
//                                    ScannerTask                                //
///////////////////////////////////////////////////////////////////////////////////

ni::ScannerTask::ScannerTask(
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task
) : running(true), scanner(ctx, task), ctx(ctx), task(task) {
    //begin scanning on construction
    LOG(INFO) << "[NI Task] constructing scanner task " << this->task.name;
    thread = std::thread(&ni::ScannerTask::run, this);
}

std::unique_ptr<task::Task> ni::ScannerTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    return std::make_unique<ni::ScannerTask>(ctx, task);
}

void ni::ScannerTask::start() {
    LOG(INFO) << "[NI Task] start function of scanner task " << this->task.name;
    this->running = true;
    // this->thread = std::thread(&ni::ScannerTask::run, this);
}

void ni::ScannerTask::stop() {
    this->running = false;
    this->thread.join();
    LOG(INFO) << "[NI Task] stopped scanner task " << this->task.name;
}


void ni::ScannerTask::exec(task::Command &cmd) {
    if (cmd.type == "scan") {
        scanner.scan();
        scanner.create_devices();
        if (!scanner.ok()) {
            ctx->setState({
                .task = task.key,
                .variant = "error",
                .details = {"message", "failed to scan"}
            });
            LOG(ERROR) << "[NI Task] failed to scan for task " << this->task.name;
        } else {
            auto devices = scanner.get_devices(); // TODO remove and dont send in details
            ctx->setState({
                .task = task.key,
                .variant = "success",
                .details = {
                    {"devices", devices.dump(4)}
                }
            });
            // LOG(INFO) << "[NI Task] successfully scanned for task " << this->task.name;
        }
    } else if (cmd.type == "stop") {
        this->stop();
    } else {
        LOG(ERROR) << "unknown command type: " << cmd.type;
    }
}

void ni::ScannerTask::run() {
    // LOG(INFO) << "[NI Task] Scanner task running " << this->task.name;
    auto scan_cmd = task::Command{task.key, "scan", {}};

    // perform a scan
    while (true) {
        std::this_thread::sleep_for(std::chrono::seconds(5));
        if (this->running) {
            this->exec(scan_cmd);
        } else {
            break;
        }
    }
}


bool ni::ScannerTask::ok() {
    return this->ok_state;
}

ni::ScannerTask::~ScannerTask() {
    LOG(INFO) << "[NI Task] destructing scanner task " << this->task.name;
}

///////////////////////////////////////////////////////////////////////////////////
//                                    ReaderTask                                 //
///////////////////////////////////////////////////////////////////////////////////
ni::ReaderTask::ReaderTask(const std::shared_ptr<task::Context> &ctx,
                           synnax::Task task,
                           std::shared_ptr<pipeline::Source> source,
                           std::shared_ptr<ni::Source> ni_source,
                           synnax::WriterConfig writer_config,
                           const breaker::Config breaker_config
) : ctx(ctx),
    task(task),
    daq_read_pipe(
        pipeline::Acquisition(ctx->client, writer_config, source, breaker_config)),
    source(ni_source) {
}


std::unique_ptr<task::Task> ni::ReaderTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task) {
    LOG(INFO) << "[NI Task] configuring task " << task.name;

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };

    auto parser = config::Parser(task.config);
    auto data_saving = parser.optional<bool>("data_saving", true);

    TaskHandle task_handle;
    ni::NiDAQmxInterface::CreateTask("", &task_handle);

    // determine whether DigitalReadSource or AnalogReadSource is needed
    std::vector<synnax::ChannelKey> channel_keys;
    std::shared_ptr<pipeline::Source> source;
    std::shared_ptr<ni::Source> ni_source;
    if (task.type != "ni_analog_read") {
        ni_source = std::make_shared<ni::DigitalReadSource>(task_handle, ctx, task);
    } else {
        ni_source = std::make_shared<ni::AnalogReadSource>(task_handle, ctx, task);
    }
    source = ni_source;
    ni_source->init();
    channel_keys = ni_source->getChannelKeys();
    // TODO: do a check that the daq reader was constructed successfully

    // construct writer config
    auto writer_config = synnax::WriterConfig{
        .channels = channel_keys,
        .start = synnax::TimeStamp::now(),
        .mode = data_saving 
                    ? synnax::WriterMode::PersistStream 
                    : synnax::WriterMode::StreamOnly,
        .enable_auto_commit = true
    };

    // start and stop to catch any immediate errors
    ni_source->cycle();

    auto p = std::make_unique<ni::ReaderTask>(ctx,
                                              task,
                                              source,
                                              ni_source,
                                              writer_config,
                                              breaker_config);

    if (!ni_source->ok()) {
        LOG(ERROR) << "[NI Task] failed to configure task " << task.name;
        return p;
    }

    // sleep for 10ms: this is here to temporarily fix a race condition in the console after hitting configure
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    ctx->setState({
        .task = task.key,
        .variant = "success",
        .details = {
            {"running", false}
        }
    });

    LOG(INFO) << "[NI Task] successfully configured task " << task.name;

    return p;
}

void ni::ReaderTask::exec(task::Command &cmd) {
    if (cmd.type == "start") {
        LOG(INFO) << "[NI Task] started reader task " << this->task.name;
        this->start();
    } else if (cmd.type == "stop") {
        LOG(INFO) << "[NI Task] stopped reader task " << this->task.name;
        this->stop();
    } else {
        LOG(ERROR) << "unknown command type: " << cmd.type;
    }
}

void ni::ReaderTask::stop() {
    if (!this->running.exchange(false)) {
        LOG(INFO) << "[NI Task] did not stop " << this->task.name << " running: " <<
                this->running << " ok: "
                << this->ok();
        return;
    }
    this->daq_read_pipe.stop();
    this->source->stop();
    LOG(INFO) << "[NI Task] successfully stopped task " << this->task.name;
}

void ni::ReaderTask::start() {
    if (this->running.exchange(true) || !this->ok() || !this->source->ok()) {
        LOG(INFO) << "[NI Task] did not start " << this->task.name <<
                " as it is not running or in error state";
        return;
    }
    this->source->start();
    this->daq_read_pipe.start();
    LOG(INFO) << "[NI Task] successfully started task " << this->task.name;
}

bool ni::ReaderTask::ok() {
    return this->ok_state;
}

///////////////////////////////////////////////////////////////////////////////////
//                                    WriterTask                                 //
///////////////////////////////////////////////////////////////////////////////////
ni::WriterTask::WriterTask(const std::shared_ptr<task::Context> &ctx,
                           synnax::Task task,
                           std::shared_ptr<pipeline::Sink> sink,
                           std::shared_ptr<ni::DigitalWriteSink> ni_sink,
                           std::shared_ptr<pipeline::Source> state_source,
                           synnax::WriterConfig writer_config,
                           synnax::StreamerConfig streamer_config,
                           const breaker::Config breaker_config
) : ctx(ctx),
    task(task),
    cmd_write_pipe(pipeline::Control(ctx->client, streamer_config, std::move(sink),
                                     breaker_config)),
    state_write_pipe(
        pipeline::Acquisition(ctx->client, writer_config, state_source,
                              breaker_config)),
    sink(ni_sink) {
}


std::unique_ptr<task::Task> ni::WriterTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task) {
    // create a breaker config TODO: use the task to generate the other parameters?
    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };

    auto parser = config::Parser(task.config);
    auto data_saving = parser.optional<bool>("data_saving", true);

    TaskHandle task_handle;
    ni::NiDAQmxInterface::CreateTask("", &task_handle);

    LOG(INFO) << "[ni.writer] configuring task " << task.name;
    auto daq_writer = std::make_shared<ni::DigitalWriteSink>(task_handle, ctx, task);

    // construct writer config
    std::vector<synnax::ChannelKey> cmd_keys = daq_writer->get_cmd_channel_keys();
    std::vector<synnax::ChannelKey> state_keys = daq_writer->get_state_channel_keys();

    // create a writer config to write state channels
    auto writer_config = synnax::WriterConfig{
        .channels = state_keys,
        .start = synnax::TimeStamp::now(),
        .mode = data_saving 
                    ? synnax::WriterMode::PersistStream 
                    : synnax::WriterMode::StreamOnly,
        .enable_auto_commit = true,
    };

    // create a streamer config to stream incoming cmds
    auto streamer_config = synnax::StreamerConfig{
        .channels = cmd_keys,
        .start = synnax::TimeStamp::now(),
    };


    auto state_writer = daq_writer->writer_state_source;

    daq_writer->cycle();
    LOG(INFO) << "[ni.writer] constructed writer for " << task.name;
    auto p = std::make_unique<ni::WriterTask>(ctx,
                                              task,
                                              daq_writer,
                                              daq_writer,
                                              state_writer,
                                              writer_config,
                                              streamer_config,
                                              breaker_config);

    if (!daq_writer->ok()) {
        LOG(ERROR) << "[ni.writer] failed to construct writer for " << task.name;
        return p;
    }

    // sleep for 10ms: this is here to temporarily fix a race condition in the console after hitting configure
    std::this_thread::sleep_for(std::chrono::milliseconds(10));

    ctx->setState({
        .task = task.key,
        .variant = "success",
        .details = {
            {"running", false}
        }
    });

    LOG(INFO) << "[ni.writer] successfully configured task " << task.name;
    return p;
}

void ni::WriterTask::exec(task::Command &cmd) {
    if (cmd.type == "start") this->start();
    else if (cmd.type == "stop") this->stop();
    else LOG(ERROR) << "unknown command type: " << cmd.type;
}


void ni::WriterTask::start() {
    if (this->running.exchange(true) || !this->ok() || !this->sink->ok()) return;

    sink->start();
    this->cmd_write_pipe.start();
    this->state_write_pipe.start();
}


void ni::WriterTask::stop() {
    if (!this->running.exchange(false)) {
        LOG(INFO) << "[NI Task] did not stop " << this->task.name << " running: " <<
                this->running << " ok: " << this->ok();
        return;
    }
    this->state_write_pipe.stop();
    this->cmd_write_pipe.stop();
    sink->stop();
}


bool ni::WriterTask::ok() {
    return this->ok_state;
}
