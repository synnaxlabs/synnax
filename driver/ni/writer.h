// Copyright 2024 Synnax Labs, Inc.
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
#include <map>
#include <set>

/// external
#include "nlohmann/json.hpp"

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/ni/channels.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"
#include "driver/ni/ni.h"

namespace ni {
struct WriteTaskConfig {
    std::set<synnax::ChannelKey> state_indexes;
    /// @brief the key of the device to retrieve from Synnax Server
    const std::string device_key;
    /// @brief the rate at which the state channel is written to
    const telem::Rate state_rate;
    /// @brief whether data saving is enabled for the task.
    const bool data_saving;

    WriteTaskConfig(WriteTaskConfig&& other) noexcept:
        state_indexes(std::move(other.state_indexes)),
        device_key(other.device_key),
        state_rate(other.state_rate),
        data_saving(other.data_saving),
        channels(std::move(other.channels)),
        cmd_to_state(other.cmd_to_state) {
    }

    WriteTaskConfig(const WriteTaskConfig&) = delete;
    const WriteTaskConfig& operator=(const WriteTaskConfig&) = delete;

    std::unordered_map<synnax::ChannelKey, std::unique_ptr<OutputChan> > channels;
    std::unordered_map<synnax::ChannelKey, synnax::ChannelKey> cmd_to_state;

    // implement move

    explicit WriteTaskConfig(
        std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg,
        std::function<std::pair<std::pair<synnax::ChannelKey, std::unique_ptr<OutputChan> >, bool>(xjson::Parser &)> parse_chan
    ):
        device_key(cfg.required<std::string>("device")),
        state_rate(telem::Rate(cfg.required<float>("state_rate"))),
        data_saving(cfg.optional<bool>("data_saving", false)) {
        cfg.iter("channels", [&](xjson::Parser &ch_cfg) {
            auto [ch, enabled] = parse_chan(ch_cfg);
            if (enabled) channels.insert(std::move(ch));
        });
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size() * 2);
        for (const auto &[_, ch]: channels) {
            keys.push_back(ch->state_ch_key);
            cmd_to_state[ch->cmd_ch_key] = ch->state_ch_key;
        }
        auto [channels_vec, err] = client->channels.retrieve(keys);
        for (const auto &ch: channels_vec)
            if (ch.index != 0)
                state_indexes.insert(ch.key);
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size() + state_indexes.size());
        for (const auto &[_, ch]: channels) keys.push_back(ch->state_ch_key);
        for (const auto &idx: state_indexes) keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .start = telem::TimeStamp::now(),
            .enable_auto_commit = true,
            .mode = synnax::data_saving_writer_mode(this->data_saving)

        };
    }

    [[nodiscard]] synnax::StreamerConfig streamer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &[_, ch]: channels) keys.push_back(ch->cmd_ch_key);
        return synnax::StreamerConfig{.channels = keys};
    }
};

inline WriteTaskConfig digital_write_task_config(
    std::shared_ptr<synnax::Synnax> &client, xjson::Parser &cfg) {
    return WriteTaskConfig(
        client,
        cfg,
        [](xjson::Parser &ch_cfg) {
            auto ch = std::make_unique<DOChan>(ch_cfg);
            return std::make_pair(
                std::move(std::make_pair(ch->cmd_ch_key, std::move(ch))),
                ch->enabled
            );
        }
    );
}

inline WriteTaskConfig analog_write_task_config(std::shared_ptr<synnax::Synnax> &client,
                                                xjson::Parser &cfg) {
    return WriteTaskConfig(
        client,
        cfg,
        [&](xjson::Parser &ch_cfg) {
            auto ch = parse_ao_chan(ch_cfg);
            return std::make_pair(
                std::move(std::make_pair(ch->cmd_ch_key, std::move(ch))),
                ch->enabled
            );
        });
}


template<typename T>
class WriteSinkStateSource : public pipeline::Sink, public pipeline::Source {
    WriteTaskConfig cfg;
    std::set<synnax::ChannelKey> state_indexes;
    std::unique_ptr<T> write_buffer = nullptr;
    loop::Timer state_timer;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;

public:
    const std::shared_ptr<DAQmx> dmx;
    TaskHandle task_handle;

    std::unordered_map<synnax::ChannelKey, telem::SampleValue> state;

    WriteSinkStateSource(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        WriteTaskConfig cfg
    ) : cfg(std::move(cfg)),
        ctx(ctx),
        task(std::move(task)),
        dmx(dmx),
        task_handle(task_handle) {
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override;

    xerrors::Error write(const synnax::Frame &frame) override;

    virtual xerrors::Error write_ni(T *data) const = 0;

    T *format_data(const synnax::Frame &frame);
};


class DigitalWriteSink final : public WriteSinkStateSource<uint8_t> {
public:
    explicit DigitalWriteSink(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        WriteTaskConfig cfg
    ): WriteSinkStateSource(dmx, task_handle, ctx, task, std::move(cfg)) {
    }

    xerrors::Error write_ni(unsigned char *data) const override;
};

class AnalogWriteSink final : public WriteSinkStateSource<double> {
public:
    explicit AnalogWriteSink(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        WriteTaskConfig cfg
    ): WriteSinkStateSource(dmx, task_handle, ctx, task, std::move(cfg)) {
    }

    xerrors::Error write_ni(double *data) const override;
};

class WriteTask final : public task::Task {
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    pipeline::Control cmd_write_pipe;
    pipeline::Acquisition state_write_pipe;

public:
    explicit WriteTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        std::shared_ptr<pipeline::Sink> sink,
        std::shared_ptr<pipeline::Source> source,
        synnax::WriterConfig writer_cfg,
        synnax::StreamerConfig streamer_cfg,
        const breaker::Config& breaker_cfg
    ): ctx(ctx),
       task(std::move(task)),
       cmd_write_pipe(ctx->client, std::move(streamer_cfg), std::move(sink), breaker_cfg),
       state_write_pipe(ctx->client, std::move(writer_cfg), std::move(source), breaker_cfg) {
    }

    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key);
    }

    void stop() override { this->stop(""); }

    void stop(const std::string &cmd_key) {
        this->cmd_write_pipe.stop();
        this->state_write_pipe.stop();
    }

    void start(const std::string &key) {
        this->cmd_write_pipe.start();
        this->state_write_pipe.start();
    }

    std::string name() override { return task.name; }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<DAQmx> &dmx,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);

        auto cfg = task.type == "digital_write"
                       ? digital_write_task_config(ctx->client, parser)
                       : analog_write_task_config(ctx->client, parser);

        auto writer_cfg = cfg.writer_config();
        auto streamer_cfg = cfg.streamer_config();

        TaskHandle task_handle;
        dmx->CreateTask("", &task_handle);

        std::shared_ptr<pipeline::Source> source;
        std::shared_ptr<pipeline::Sink> sink;

        if (task.type == "digital_read") {
            auto source_sink = std::make_shared<DigitalWriteSink>(
                dmx,
                task_handle,
                ctx,
                task,
                std::move(cfg)
            );
            sink = source_sink;
            source = source_sink;
        } else {
            auto source_sink = std::make_shared<AnalogWriteSink>(
                dmx,
                task_handle,
                ctx,
                task,
                std::move(cfg)
            );
            sink = source_sink;
            source = source_sink;
        }

        return std::make_unique<WriteTask>(
            ctx,
            task,
            sink,
            source,
            writer_cfg,
            streamer_cfg,
            breaker::default_config(task.name)
        );
    }
};
}
