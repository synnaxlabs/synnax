// Copyright 2025 Synnax Labs, Inc.
//
// Use of this is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>
#include <vector>
#include <set>
#include <thread>

/// module
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/loop/loop.h"

/// internal
#include "driver/ni/channel/channels.h"
#include "driver/pipeline/acquisition.h"
#include "driver/ni/hardware.h"
#include "driver/task/task.h"
#include "driver/ni/ni.h"

namespace ni {
/// @brief the configuration for a read task.
struct ReadTaskConfig {
    /// @brief whether data saving is enabled for the task.
    const bool data_saving;
    /// @brief the device key that will be used for the channels in the task. Analog
    /// read tasks can specify multiple devices. In this case, the device key field
    /// is empty and automatically set to "cross-device".
    const std::string device_key;
    /// @brief sets the sample rate for the task.
    const telem::Rate sample_rate;
    /// @brief sets the stream rate for the task.
    const telem::Rate stream_rate;
    /// @brief sets the timing source for the task. If not provided, the task will
    /// use software timing on digital tasks and the sample clock on analog tasks.
    const std::string timing_source;
    /// @brief the number of samples per channel to connect on each call to read.
    const std::size_t samples_per_chan;
    /// @brief whether the task should be software timed.
    const bool software_timed;
    /// @brief the indexes of the channels in the task.
    std::set<synnax::ChannelKey> indexes;
    /// @brief the configurations for each channel in the task.
    std::vector<std::unique_ptr<channel::Input>> channels;

    /// @brief Move constructor to allow transfer of ownership
    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        data_saving(other.data_saving),
        device_key(other.device_key),
        sample_rate(other.sample_rate),
        stream_rate(other.stream_rate),
        timing_source(other.timing_source),
        samples_per_chan(other.samples_per_chan),
        software_timed(other.software_timed),
        indexes(std::move(other.indexes)),
        channels(std::move(other.channels)) {
    }

    /// @brief delete copy constructor and copy assignment to prevent accidental copies.
    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg,
        const std::string &task_type
    ): data_saving(cfg.optional<bool>("data_saving", false)),
       device_key(cfg.optional<std::string>("device", "cross-device")),
       sample_rate(telem::Rate(cfg.required<float>("sample_rate"))),
       stream_rate(telem::Rate(cfg.required<float>("stream_rate"))),
       timing_source(cfg.optional<std::string>("timing_source", "none")),
       samples_per_chan(
           static_cast<size_t>(std::floor(sample_rate.value / stream_rate.value))),
       software_timed(this->timing_source == "none" && task_type == "ni_digital_read"),
       channels(cfg.map<std::unique_ptr<channel::Input>>(
           "channels",
           [&](xjson::Parser &ch_cfg) -> std::pair<std::unique_ptr<channel::Input>,
       bool> {
               auto ch = channel::parse_input(ch_cfg, {});
               return {std::move(ch), ch->enabled};
           })) {
        if (this->channels.empty()) {
            cfg.field_err("channels", "task must have at least one channel");
            return;
        }
        std::vector<synnax::ChannelKey> channel_keys;
        for (const auto &ch: this->channels) channel_keys.push_back(ch->synnax_key);
        auto [channel_vec, err] = client->channels.retrieve(channel_keys);
        if (err) {
            cfg.field_err("", "failed to retrieve channels for task");
            return;
        }
        auto remote_channels = channel_keys_map(channel_vec);
        if (this->device_key != "cross-device") {
            auto [device, err] = client->hardware.retrieve_device(this->device_key);
            if (err) {
                cfg.field_err("", "failed to retrieve device for task");
                return;
            }
        }
        std::vector<std::string> dev_keys;
        for (const auto &ch: this->channels) dev_keys.push_back(ch->dev_key);
        auto [devices_vec, dev_err] = client->hardware.retrieve_devices(dev_keys);
        if (dev_err) {
            cfg.field_err("", "failed to retrieve devices for task");
            return;
        }
        auto devices = device_keys_map(devices_vec);
        for (auto &ch: this->channels) {
            const auto &remote_ch = remote_channels.at(ch->synnax_key);
            auto dev = devices[ch->dev_key];
            ch->bind_remote_info(remote_ch, dev.location);
            if (ch->ch.index != 0) this->indexes.insert(ch->ch.index);
        }
    }

    static std::pair<ReadTaskConfig, xerrors::Error> parse(
        std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {ReadTaskConfig(client, parser, task.type), parser.error()};
    }

    xerrors::Error apply(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle handle
    ) const {
        for (const auto &ch: this->channels)
            if (auto err = ch->apply(dmx, handle)) return err;
        if (this->software_timed) return xerrors::NIL;
        return dmx->CfgSampClkTiming(
            handle,
            this->timing_source == "none" ? nullptr : this->timing_source.c_str(),
            this->sample_rate.value,
            DAQmx_Val_Rising,
            DAQmx_Val_ContSamps,
            this->samples_per_chan
        );
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

/// @brief a read task that can pull from both analog and digital channels.
template<typename T>
class ReadTask final : public task::Task {
    /// @brief the raw synnax task configuration.
    /// @brief the parsed configuration for the task.
    const ReadTaskConfig cfg;
    /// @brief the task context used to communicate state changes back to Synnax.
    /// @brief tare middleware used for taring values.
    std::shared_ptr<pipeline::TareMiddleware> tare_mw;
    /// @brief the pipeline used to read data from the hardware and pipe it to Synnax.
    pipeline::Acquisition pipe;
    /// @brief interface used to read data from the hardware.
    std::unique_ptr<HardwareReader<T>> hw;
    /// @brief the timestamp at which the hardware task was started. We use this to
    /// interpolate the correct timestamps of recorded samples.
    telem::TimeStamp hw_start_time = telem::TimeStamp(0);
    /// @brief handles communicating the task state back to the cluster.
    ni::TaskStateHandler state;

    /// @brief an internal source that we pass to the acquisition pipeline that manages
    /// the lifecycle of this task.
    class Source final : public pipeline::Source {
    public:
        explicit Source(ReadTask &task):
            p(task),
            timer(task.cfg.stream_rate),
            buffer(task.cfg.samples_per_chan * task.cfg.channels.size()) {
        }

    private:
        /// @brief the parent read task.
        ReadTask &p;
        /// @brief a separate thread to acquire samples in.
        loop::Timer timer;
        /// @brief automatically infer the data type from the template parameter. This
        /// will either be UINT8_T or FLOAT64_T. We use this to appropriately cast
        /// the data read from the hardware.
        const telem::DataType data_type = telem::DataType::infer<T>();
        /// @brief the buffer used to read data from the hardware. This vector is
        /// pre-allocated and reused.
        std::vector<T> buffer;

        void stopped_with_err(const xerrors::Error &err) override {
            this->p.state.error(this->p.hw->stop());
            this->p.state.send_stop("");
        }

        std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
            if (this->p.cfg.software_timed) this->timer.wait(breaker);
            auto start = this->p.hw_start_time;
            const size_t count = this->p.cfg.samples_per_chan;
            const auto [n, err] = this->p.hw->read(count, buffer);
            if (err) return {Frame(), err};
            auto end = start + (n - 1) * this->p.cfg.sample_rate.period();
            this->p.hw_start_time = end + this->p.cfg.sample_rate.period();

            auto f = synnax::Frame(this->p.cfg.channels.size());
            size_t data_index = 0;
            for (const auto &ch: this->p.cfg.channels) {
                auto s = telem::Series(ch->ch.data_type, count);
                const size_t start_idx = data_index * count;
                if (s.data_type == this->data_type)
                    s.write(buffer.data() + start_idx, count);
                else
                    for (int i = 0; i < count; ++i)
                        s.write(s.data_type.cast(buffer.at(start_idx + i)));
                f.emplace(ch->synnax_key, std::move(s));
                data_index++;
            }
            if (!this->p.cfg.indexes.empty()) {
                const auto index_data = telem::Series::linspace(start, end, count);
                for (const auto &idx: this->p.cfg.indexes)
                    f.emplace(idx, std::move(index_data.deep_copy()));
            }
            return std::make_pair(std::move(f), xerrors::NIL);
        }
    };

public:
    explicit ReadTask(
        synnax::Task task,
        const std::shared_ptr<task::Context> &ctx,
        ReadTaskConfig cfg,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<HardwareReader<T>> hw,
        const std::shared_ptr<pipeline::WriterFactory> &factory
    ): cfg(std::move(cfg)),
       tare_mw(std::make_shared<pipeline::TareMiddleware>(
           this->cfg.writer_config().channels
       )),
       pipe(
           factory,
           this->cfg.writer_config(),
           std::make_shared<Source>(*this),
           breaker_cfg
       ),
       hw(std::move(hw)),
       state(ctx, task) {
        this->pipe.add_middleware(this->tare_mw);
    }

    explicit ReadTask(
        synnax::Task task,
        const std::shared_ptr<task::Context> &ctx,
        ReadTaskConfig cfg,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<HardwareReader<T>> hw
    ): ReadTask(
        std::move(task),
        ctx,
        std::move(cfg),
        breaker_cfg,
        std::move(hw),
        std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client)
    ) {
    }

    /// @brief executes the given command on the task.
    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key);
        else if (cmd.type == "tare") this->tare_mw->tare(cmd.args);
    }

    /// @brief stops the task.
    void stop() override { this->stop(""); }

    /// @brief stops the task, using the given command key as reference for
    /// communicating success state.
    void stop(const std::string &cmd_key) {
        this->pipe.stop();
        this->state.error(this->hw->stop());
        this->state.send_stop(cmd_key);
    }

    /// @brief starts the task, using the given command key as a reference for
    /// communicating task state.
    void start(const std::string &cmd_key) {
        if (!this->state.error(this->hw->start())) this->pipe.start();
        this->state.send_start(cmd_key);
        this->pipe.start();
    }

    /// @brief implements task::Task.
    std::string name() override { return this->state.task.name; }
};
}
