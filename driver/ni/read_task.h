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

/// module
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xjson/xjson.h"
#include "x/cpp/loop/loop.h"

/// internal
#include "driver/ni/channel/channels.h"
#include "driver/pipeline/acquisition.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/task/task.h"
#include "driver/task/common/read_task.h"
#include "driver/ni/ni.h"
#include "driver/task/common/sample_clock.h"

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
       timing_source(cfg.optional<std::string>("timing_source", "")),
       samples_per_chan(sample_rate / stream_rate),
       software_timed(this->timing_source.empty() && task_type == "ni_digital_read"),
       channels(cfg.map<std::unique_ptr<channel::Input>>(
           "channels",
           [&](xjson::Parser &ch_cfg) -> std::pair<std::unique_ptr<channel::Input>,
       bool> {
               auto ch = channel::parse_input(ch_cfg);
               return {std::move(ch), ch->enabled};
           })) {
        if (this->channels.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        if (this->sample_rate < this->stream_rate) {
            cfg.field_err("sample_rate",
                          "sample rate must be greater than or equal to stream rate");
            return;
        }
        std::vector<synnax::ChannelKey> channel_keys;
        for (const auto &ch: this->channels) channel_keys.push_back(ch->synnax_key);
        auto [channel_vec, err] = client->channels.retrieve(channel_keys);
        if (err) {
            cfg.field_err("channels",
                          "failed to retrieve channels for task: " + err.message());
            return;
        }
        auto remote_channels = map_channel_Keys(channel_vec);
        std::unordered_map<std::string, synnax::Device> devices;
        if (this->device_key != "cross-device") {
            auto [device, err] = client->hardware.retrieve_device(this->device_key);
            if (err) {
                cfg.field_err("device",
                              "failed to retrieve device for task: " + err.message());
                return;
            }
            devices[device.key] = device;
        } else {
            std::vector<std::string> dev_keys;
            for (const auto &ch: this->channels) dev_keys.push_back(ch->dev_key);
            auto [devices_vec, dev_err] = client->hardware.retrieve_devices(dev_keys);
            if (dev_err) {
                cfg.field_err("device",
                              "failed to retrieve devices for task: " + dev_err.
                              message());
                return;
            }
            devices = map_device_keys(devices_vec);
        }
        for (auto &ch: this->channels) {
            const auto &remote_ch = remote_channels.at(ch->synnax_key);
            auto dev = this->device_key == "cross-device"
                           ? devices.at(ch->dev_key)
                           : devices.at(this->device_key);
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

    std::vector<synnax::Channel> sy_channels() const {
        std::vector<synnax::Channel> chs;
        chs.reserve(this->channels.size());
        for (const auto &ch: this->channels) chs.push_back(ch->ch);
        return chs;
    }

    [[nodiscard]]

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle handle
    ) const {
        for (const auto &ch: this->channels)
            if (auto err = ch->apply(dmx, handle)) return err;
        if (this->software_timed) return xerrors::NIL;
        return dmx->CfgSampClkTiming(
            handle,
            this->timing_source.empty() ? nullptr : this->timing_source.c_str(),
            this->sample_rate.hz(),
            DAQmx_Val_Rising,
            DAQmx_Val_ContSamps,
            this->samples_per_chan
        );
    }

    [[nodiscard]] synnax::WriterConfig writer() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(this->channels.size() + this->indexes.size());
        for (const auto &ch: this->channels) keys.push_back(ch->ch.key);
        for (const auto &idx: this->indexes) keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = synnax::data_saving_writer_mode(this->data_saving),
            .enable_auto_commit = true
        };
    }

    [[nodiscard]] std::unique_ptr<common::SampleClock> sample_clock() const {
        if (this->software_timed)
            return std::make_unique<common::SoftwareTimedSampleClock>(this->stream_rate);
        return std::make_unique<common::HardwareTimedSampleClock>(this->sample_rate);
    }
};

/// @brief an internal source that we pass to the acquisition pipeline that manages
/// the lifecycle of this task.
template<typename T>
class ReadTaskSource final : public common::Source {
public:
    /// @brief constructs a source bound to the provided parent read task.
    explicit ReadTaskSource(ReadTaskConfig cfg,
                            std::unique_ptr<hardware::Reader<T>> hw_reader):
        cfg(std::move(cfg)),
        buffer(this->cfg.samples_per_chan * this->cfg.channels.size()),
        hw_reader(std::move(hw_reader)),
        sample_clock(this->cfg.sample_clock()) {
    }

private:
    /// @brief the raw synnax task configuration.
    /// @brief the parsed configuration for the task.
    const ReadTaskConfig cfg;
    /// @brief the buffer used to read data from the hardware. This vector is
    /// pre-allocated and reused.
    std::vector<T> buffer;
    /// @brief interface used to read data from the hardware.
    std::unique_ptr<hardware::Reader<T>> hw_reader;
    /// @brief the timestamp at which the hardware task was started. We use this to
    /// interpolate the correct timestamps of recorded samples.
    std::unique_ptr<common::SampleClock> sample_clock;
    /// @brief the error accumulated from the latest read. Primarily used to determine
    /// whether we've just recovered from an error state.
    xerrors::Error curr_read_err = xerrors::NIL;

    std::vector<synnax::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    xerrors::Error start() override {
        this->sample_clock->reset();
        return this->hw_reader->start();
    }

    xerrors::Error stop() override {
        return this->hw_reader->stop();
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return this->cfg.writer();
    }

    std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
        auto start = this->sample_clock->wait(breaker);
        const auto [n, err] = this->hw_reader->read(
            this->cfg.samples_per_chan,
            buffer
        );
        auto prev_read_err = this->curr_read_err;
        this->curr_read_err = translate_error(err);
        if (this->curr_read_err) return {Frame(), this->curr_read_err};
        // If we just recovered from an error, we need to reset the sample clock so
        // we can start timing samples again from a steady state.
        if (prev_read_err) this->sample_clock->reset();
        auto end = this->sample_clock->end(n);
        synnax::Frame f(this->cfg.channels.size() + this->cfg.indexes.size());
        size_t i = 0;
        for (const auto &ch: this->cfg.channels)
            f.emplace(
                ch->synnax_key,
                telem::Series::cast(ch->ch.data_type, buffer.data() + i++ * n, n)
            );
        common::generate_index_data(f, this->cfg.indexes, start, end, n);
        return std::make_pair(std::move(f), xerrors::NIL);
    }
};
}
