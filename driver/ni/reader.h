// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>
#include <vector>
#include <memory>
#include <set>
#include <thread>

/// module
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/loop/loop.h"

/// internal
#include "driver/ni/daqmx/daqmx.h"
#include "driver/ni/ni.h"
#include "driver/ni/channels.h"
#include "driver/queue/ts_queue.h"
#include "driver/pipeline/acquisition.h"
#include "driver/task/task.h"

namespace ni {
struct BaseReadTaskConfig {
    const std::string device_key;
    const telem::Rate sample_rate;
    const telem::Rate stream_rate;
    const std::string timing_source;
    const std::size_t samples_per_channel;
    std::size_t buffer_size = 0;
    std::set<synnax::ChannelKey> indexes;
    const bool data_saving;

    explicit BaseReadTaskConfig(xjson::Parser &cfg):
        device_key(cfg.required<std::string>("device_key")),
        sample_rate(telem::Rate(cfg.required<float>("sample_rate"))),
        stream_rate(telem::Rate(cfg.required<float>("stream_rate"))),
        timing_source(cfg.required<std::string>("timing_source")),
        data_saving(cfg.optional<bool>("data_saving", false)),
        samples_per_channel(std::floor(sample_rate.value / stream_rate.value)) {
    }
};

struct AnalogReadTaskConfig : BaseReadTaskConfig {
    std::vector<std::unique_ptr<AIChan> > channels;

    // Add move constructor to properly move the channels vector
    AnalogReadTaskConfig(AnalogReadTaskConfig &&other) noexcept
        : BaseReadTaskConfig(std::move(other)),
          channels(std::move(other.channels)) {
    }

    // Delete copy constructor to prevent accidental copies
    AnalogReadTaskConfig(const AnalogReadTaskConfig &) = delete;

    AnalogReadTaskConfig &operator=(const AnalogReadTaskConfig &) = delete;

    explicit AnalogReadTaskConfig(
        std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ) : BaseReadTaskConfig(cfg),
        channels(cfg.map<std::unique_ptr<AIChan> >(
            "channels",
            [&](xjson::Parser &ch_cfg) -> std::pair<std::unique_ptr<AIChan>, bool> {
                auto ch = parse_ai_chan(ch_cfg, {});
                return {std::move(ch), ch->enabled};
            }
        )) {
        std::vector<synnax::ChannelKey> channel_keys;
        for (const auto &ch: this->channels) channel_keys.push_back(ch->ch.key);
        auto [channel_vec, err] = client->channels.retrieve(channel_keys);
        if (err) {
            cfg.field_err("", "failed to retrieve channels for task");
            return;
        }
        auto channels = channel_keys_map(channel_vec);
        if (this->device_key != "cross-device") {
            auto [device, err] = client->hardware.retrieve_device(this->device_key);
            if (err) {
                cfg.field_err("", "failed to retrieve device for task");
                return;
            }
        }
        std::vector<std::string> dev_keys;
        for (const auto &ch: this->channels) dev_keys.push_back(ch->dev);
        auto [devices_vec, dev_err] = client->hardware.retrieve_devices(dev_keys);
        if (dev_err) {
            cfg.field_err("", "failed to retrieve devices for task");
            return;
        }
        auto devices = device_keys_map(devices_vec);
        for (const auto &ch: this->channels) {
            ch->bind_remote_info(channels[ch->ch.key], devices[ch->dev].name);
            this->buffer_size = this->samples_per_channel * channels[ch->ch.key].
                                data_type.density();
        }
    }

    xerrors::Error bind(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle handle
    ) const {
        dmx->CfgSampClkTiming(
            handle,
            this->timing_source == "none" ? nullptr : this->timing_source.c_str(),
            this->sample_rate.value,
            DAQmx_Val_Rising,
            DAQmx_Val_ContSamps,
            this->sample_rate.value
        );
        for (const auto &ch: this->channels) {
            if (int32 err = ch->bind_task(dmx, handle))
                return xerrors::Error(
                    "failed to bind channel " + std::to_string(ch->ch.key));
        }
        return xerrors::NIL;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size() + this->indexes.size());
        for (const auto &ch: this->channels) keys.push_back(ch->ch.key);
        for (const auto &idx: this->indexes) keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = synnax::data_saving_writer_mode(this->data_saving)
        };
    }
};

struct DigitalReadTaskConfig : BaseReadTaskConfig {
    std::vector<DIChan> channels;

    explicit DigitalReadTaskConfig(
        std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ): BaseReadTaskConfig(cfg),
       channels(cfg.map<DIChan>(
           "channels",
           [&](xjson::Parser &ch_cfg) -> std::pair<DIChan, bool> {
               auto ch = DIChan(ch_cfg);
               return {ch, ch.enabled};
           }
       )) {
        std::vector<synnax::ChannelKey> channel_keys;
        for (const auto &ch: this->channels) channel_keys.push_back(ch.ch.key);
        auto [channel_vec, err] = client->channels.retrieve(channel_keys);
        if (err) {
            cfg.field_err("", "failed to retrieve channels for task");
            return;
        }
        auto channels = channel_keys_map(channel_vec);

        const auto [device, d_err] = client->hardware.retrieve_device(this->device_key);
        if (d_err) {
            cfg.field_err("", "failed to retrieve device for task");
            return;
        }
        for (auto &ch: this->channels) {
            ch.bind_remote_info(channels[ch.ch.key], device.location);
            this->buffer_size = this->samples_per_channel * channels[ch.ch.key].
                                data_type.density();
        }
    }

    xerrors::Error bind(const std::shared_ptr<DAQmx> &dmx, TaskHandle task_handle) {
        if (this->timing_source != "none") {
            dmx->CfgSampClkTiming(
                task_handle,
                this->timing_source.c_str(),
                this->sample_rate.value,
                DAQmx_Val_Rising,
                DAQmx_Val_ContSamps,
                this->sample_rate.value
            );
        }
        this->buffer_size = this->channels.size() * this->samples_per_channel;
        return xerrors::NIL;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size() + this->indexes.size());
        for (const auto &ch: this->channels) keys.push_back(ch.ch.key);
        for (const auto &idx: this->indexes) keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = synnax::data_saving_writer_mode(this->data_saving)
        };
    }
};

class Source : public pipeline::Source {
public:
    explicit Source(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ): sample_thread_breaker(breaker::default_config(task.name)),
       task_handle(task_handle),
       dmx(dmx),
       ctx(ctx),
       task(task) {
    }

    void stopped_with_err(const xerrors::Error &err) override;

    virtual void acquire_data() = 0;

    /// @brief shared resources between daq sampling thread and acquisition thread
    struct DataPacket {
        std::vector<double> analog_data;
        std::vector<std::uint8_t> digital_data;
        telem::TimeStamp t0; // initial timestamp
        telem::TimeStamp tf; // final timestamp
        int32 samples_read_per_channel;
    };

    std::thread sample_thread;
    breaker::Breaker sample_thread_breaker;

    TaskHandle task_handle;
    TSQueue<DataPacket> queue;
    std::shared_ptr<DAQmx> dmx;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
};

class AnalogReadSource final : public Source {
    AnalogReadTaskConfig cfg;

public:
    explicit AnalogReadSource(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        AnalogReadTaskConfig cfg
    ) : Source(dmx, task_handle, ctx, task),
        cfg(std::move(cfg)) {
    }

    std::pair<synnax::Frame, xerrors::Error>
    read(breaker::Breaker &breaker) override;

    void acquire_data() override;
};

class DigitalReadSource final : public Source {
    DigitalReadTaskConfig cfg;
    loop::Timer timer;
    loop::Timer sample_timer;

public:
    explicit DigitalReadSource(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        DigitalReadTaskConfig cfg
    ) : Source(dmx, task_handle, ctx, task), cfg(std::move(cfg)) {
    }

    std::pair<synnax::Frame, xerrors::Error>
    read(breaker::Breaker &breaker) override;

    void acquire_data() override;
};

class ReadTask final : public task::Task {
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    pipeline::Acquisition daq_read_pipe;
    std::shared_ptr<ni::Source> source;
    std::shared_ptr<pipeline::TareMiddleware> tare_mw;

public:
    explicit ReadTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        const std::shared_ptr<pipeline::Source> &source,
        const synnax::WriterConfig &writer_config,
        const breaker::Config &breaker_config
    ): ctx(ctx),
       task(std::move(task)),
       daq_read_pipe(ctx->client, writer_config, source, breaker_config),
       tare_mw(std::make_shared<pipeline::TareMiddleware>(writer_config.channels)) {
        this->daq_read_pipe.add_middleware(this->tare_mw);
    }

    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key);
        else if (cmd.type == "tare") this->tare_mw->tare(cmd.args);
    }

    void stop() override { this->stop(""); }

    void stop(const std::string &cmd_key) {
        this->daq_read_pipe.stop();
    }

    void start(const std::string &cmd_key) {
        this->daq_read_pipe.start();
    }

    std::string name() override { return task.name; }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<DAQmx> &dmx,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        if (parser.error()) return nullptr;

        std::shared_ptr<pipeline::Source> source;
        synnax::WriterConfig writer_config;

        if (task.type == "ni_analog_read") {
            auto cfg = AnalogReadTaskConfig(ctx->client, parser);
            if (parser.error()) return nullptr;
            TaskHandle task_handle;
            dmx->CreateTask("", &task_handle);
            if (const auto err = cfg.bind(dmx, task_handle))
                return nullptr;
            writer_config = cfg.writer_config();
            source = std::make_shared<AnalogReadSource>(
                dmx, task_handle, ctx, task, std::move(cfg));
        } else {
            auto cfg = DigitalReadTaskConfig(ctx->client, parser);
            if (parser.error()) return nullptr;
            TaskHandle task_handle;
            dmx->CreateTask("", &task_handle);
            if (const auto err = cfg.bind(dmx, task_handle))
                return nullptr;
            writer_config = cfg.writer_config();
            source = std::make_shared<DigitalReadSource>(
                dmx, task_handle, ctx, task, std::move(cfg));
        }
        return std::make_unique<ReadTask>(
            ctx, task, source, writer_config, breaker::default_config(task.name)
        );
    }
};
}
