// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <cassert>
#include <stdio.h>

#include "driver/ni/ni.h"
#include "driver/pipeline/middleware.h"


///////////////////////////////////////////////////////////////////////////////////
//                                    ScannerTask                                //
///////////////////////////////////////////////////////////////////////////////////
ni::ScannerTask::ScannerTask(
    const std::shared_ptr<task::Context> &ctx,
    synnax::Task task
) : scanner(ctx, task), ctx(ctx), task(task) {
    this->breaker = breaker::Breaker(breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    });

    if (!scanner.ok()) {
        ctx->set_state({
            .task = task.key,
            .variant = "error",
            .details = {"message", "failed to initialize scanner"}
        });
        return;
    }
    this->breaker.start();
    thread = std::make_shared<std::thread>(&ni::ScannerTask::run, this);
    this->scanner.set_scan_thread(thread);
}

std::unique_ptr<task::Task> ni::ScannerTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    return std::make_unique<ni::ScannerTask>(ctx, task);
}


void ni::ScannerTask::stop() {
    this->breaker.stop();
}


void ni::ScannerTask::exec(task::Command &cmd) {
    if (cmd.type == "scan") {
        scanner.scan();
        scanner.create_devices();
    } else if (cmd.type == "stop") {
        this->stop();
    } else {
        LOG(ERROR) << "unknown command type: " << cmd.type;
    }
}

void ni::ScannerTask::run() {
    auto scan_cmd = task::Command{task.key, "scan", {}};
    // perform a scan
    while (this->breaker.running()) {
        this->breaker.waitFor(this->scan_rate.period().chrono());
        this->exec(scan_cmd);
    }
}


bool ni::ScannerTask::ok() {
    return this->ok_state;
}

ni::ScannerTask::~ScannerTask() {
    if (this->thread->joinable() && this->thread->get_id() != std::this_thread::get_id())
        this->thread->detach();
}

///////////////////////////////////////////////////////////////////////////////////
//                                    ReaderTask                                 //
///////////////////////////////////////////////////////////////////////////////////
ni::ReaderTask::ReaderTask(
        const std::shared_ptr<task::Context> &ctx,
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

    this->ok_state = ni_source->ok();

    // middleware chain
    std::vector<synnax::ChannelKey> channel_keys = ni_source->get_channel_keys();
    this->tare_mw = std::make_shared<pipeline::TareMiddleware>(channel_keys);
    daq_read_pipe.add_middleware(tare_mw);

}


std::unique_ptr<task::Task> ni::ReaderTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task) {
    LOG(INFO) << "[ni.task] configuring task " << task.name;

    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };

    auto parser = config::Parser(task.config);
    auto data_saving = parser.optional<bool>("data_saving", true);
//    LOG(INFO) << "Task config: " << parser.get_json().dump(4);

    TaskHandle task_handle;
    ni::NiDAQmxInterface::CreateTask("", &task_handle);

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
    channel_keys = ni_source->get_channel_keys();

    auto writer_config = synnax::WriterConfig{
        .channels = channel_keys,
        .start = synnax::TimeStamp::now(),
        .mode = data_saving
                    ? synnax::WriterMode::PersistStream
                    : synnax::WriterMode::StreamOnly,
        .enable_auto_commit = true
    };

    // start and stop to catch any immediate errors
    if (ni_source->ok()) ni_source->cycle();

    auto p = std::make_unique<ni::ReaderTask>(
            ctx,
            task,
            source,
            ni_source,
            writer_config,
            breaker_config
        );

    if (!ni_source->ok()) {
        LOG(ERROR) << "[ni.task] failed to configure task " << task.name;
        return p;
    }

    ctx->set_state({
        .task = task.key,
        .variant = "success",
        .details = {
            {"running", false},
            {"message", "Successfully configured task"}
        }
    });

    LOG(INFO) << "[ni.task] successfully configured task " << task.name;

    return p;
}

void ni::ReaderTask::exec(task::Command &cmd) {
    if (cmd.type == "start") {
        this->start(cmd.key);
        LOG(INFO) << "[ni.reader] started task " << this->task.name;
    } else if (cmd.type == "stop") {
        this->stop(cmd.key);
        LOG(INFO) << "[ni.reader] stopped task " << this->task.name;
    } else if (cmd.type == "tare"){
        if(this->ok()){
            this->tare_mw->tare(cmd.args);
            LOG(INFO) << "[ni.reader] tared channels for " << this->task.name;
        }
    }
}

void ni::ReaderTask::stop() { this->stop(""); }

void ni::ReaderTask::stop(const std::string &cmd_key) {
    if (!this->running.exchange(false)) { // TODO: if running false, return silenelty as task is already stopped
        LOG(INFO) << "[ni.task] did not stop " << this->task.name << " running: " <<
                this->running << " ok: "
                << this->ok();
        return;
    }
    this->daq_read_pipe.stop();
    this->source->stop(cmd_key);
    LOG(INFO) << "[ni.task] successfully stopped task " << this->task.name;
}

void ni::ReaderTask::start(const std::string &cmd_key) {
    if (this->running.exchange(true) || !this->ok() || !this->source->ok()) {
        LOG(INFO) << "[ni.task] did not start " << this->task.name <<
                " as it is not running or in error state";
        return;
    }
    this->source->start(cmd_key);
    this->daq_read_pipe.start();
    LOG(INFO) << "[ni.task] successfully started task " << this->task.name;
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
                           synnax::WriterConfig state_writer_config,
                           synnax::StreamerConfig cmd_streamer_config,
                           const breaker::Config breaker_config
) : ctx(ctx),
    task(task),
    cmd_write_pipe(
        pipeline::Control(
            ctx->client,
            cmd_streamer_config,
            std::move(sink),
            breaker_config
        )
    ),
    state_write_pipe(
        pipeline::Acquisition(
            ctx->client,
            state_writer_config,
            state_source,
            breaker_config
        )
    ),
    sink(ni_sink) {
}


std::unique_ptr<task::Task> ni::WriterTask::configure(
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task) {
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

    std::vector<synnax::ChannelKey> cmd_keys = daq_writer->get_cmd_channel_keys();
    std::vector<synnax::ChannelKey> state_keys = daq_writer->get_state_channel_keys();

    auto state_writer_config = synnax::WriterConfig{
        .channels = state_keys,
        .start = synnax::TimeStamp::now(),
        .mode = data_saving
                    ? synnax::WriterMode::PersistStream
                    : synnax::WriterMode::StreamOnly,
        .enable_auto_commit = true,
    };

    auto cmd_streamer_config = synnax::StreamerConfig{
        .channels = cmd_keys,
    };

    if (daq_writer->ok()) daq_writer->cycle();

    auto state_writer = daq_writer->writer_state_source;

    VLOG(1) << "[ni.writer] constructed writer for " << task.name;
    auto p = std::make_unique<ni::WriterTask>(
            ctx,
            task,
            daq_writer,
            daq_writer,
            state_writer,
            state_writer_config,
            cmd_streamer_config,
            breaker_config
        );

    if (!daq_writer->ok()) {
        LOG(ERROR) << "[ni.writer] failed to construct writer for " << task.name;
        return p;
    }

    ctx->set_state({
        .task = task.key,
        .variant = "success",
        .details = {
            {"running", false},
            {"message", "Successfully configured task"}
        }
    });

    LOG(INFO) << "[ni.writer] successfully configured task " << task.name;
    return p;
}

void ni::WriterTask::exec(task::Command &cmd) {
    if (cmd.type == "start") this->start(cmd.key);
    else if (cmd.type == "stop") this->stop(cmd.key);
}


void ni::WriterTask::start(const std::string &key) {
    if (this->running.exchange(true) || !this->ok() || !this->sink->ok()) return;
    sink->start(key);
    this->cmd_write_pipe.start();
    this->state_write_pipe.start();
}


void ni::WriterTask::stop() { this->stop(""); }

void ni::WriterTask::stop(const std::string &cmd_key) {
    if (!this->running.exchange(false)) {
        LOG(INFO) << "[ni.task] did not stop " << this->task.name << " running: " <<
                this->running << " ok: " << this->ok();
        return;
    }
    this->state_write_pipe.stop();
    this->cmd_write_pipe.stop();
    sink->stop(cmd_key);
}


bool ni::WriterTask::ok() {
    return this->ok_state;
}
