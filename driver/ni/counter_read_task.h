// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <set>
#include <sstream>
#include <string>
#include <vector>

#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xjson/xjson.h"

#include "driver/ni/channel/channels.h"
#include "driver/ni/daqmx/nidaqmx.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/ni/ni.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/sample_clock.h"

namespace ni {
/// @brief the configuration for a counter read task.
struct CounterReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief the device key that will be used for the channels in the task.
    const std::string device_key;
    /// @brief sets the timing source for the task. If not provided, the task will
    /// use the sample clock.
    const std::string timing_source;
    /// @brief the number of samples per channel to acquire on each call to read.
    const std::size_t samples_per_chan;
    /// @brief the indexes of the channels in the task.
    std::set<synnax::ChannelKey> indexes;
    /// @brief the configurations for each channel in the task.
    std::vector<std::unique_ptr<channel::Input>> channels;
    /// @brief the amount of sample skew needed to trigger a warning that Synnax
    /// cannot keep up with the amount of clock skew.
    size_t skew_warn_on_count;

    /// @brief Move constructor to allow transfer of ownership
    CounterReadTaskConfig(CounterReadTaskConfig &&other) noexcept:
        common::BaseReadTaskConfig(std::move(other)),
        device_key(other.device_key),
        timing_source(other.timing_source),
        samples_per_chan(other.samples_per_chan),
        indexes(std::move(other.indexes)),
        channels(std::move(other.channels)),
        skew_warn_on_count(other.skew_warn_on_count) {}

    /// @brief delete copy constructor and copy assignment to prevent accidental
    /// copies.
    CounterReadTaskConfig(const CounterReadTaskConfig &) = delete;

    const CounterReadTaskConfig &operator=(const CounterReadTaskConfig &) = delete;

    explicit CounterReadTaskConfig(
        std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg,
        common::TimingConfig timing_cfg = common::TimingConfig()
    ):
        BaseReadTaskConfig(cfg, timing_cfg),
        device_key(cfg.required<std::string>("device")),
        timing_source(cfg.optional<std::string>("timing_source", "")),
        samples_per_chan(sample_rate / stream_rate),
        channels(cfg.map<std::unique_ptr<channel::Input>>(
            "channels",
            [](xjson::Parser &ch_cfg)
                -> std::pair<std::unique_ptr<channel::Input>, bool> {
                auto ch = channel::parse_input(ch_cfg);
                if (ch == nullptr) return {nullptr, false};
                return {std::move(ch), ch->enabled};
            }
        )),
        skew_warn_on_count(cfg.optional<std::size_t>(
            "skew_warn_on_count",
            this->sample_rate.hz() // Default to 1 second behind
        )) {
        if (this->channels.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        if (this->sample_rate < this->stream_rate) {
            cfg.field_err(
                "sample_rate",
                "sample rate must be greater than or equal to stream rate"
            );
            return;
        }
        std::vector<synnax::ChannelKey> channel_keys;
        for (const auto &ch: this->channels)
            channel_keys.push_back(ch->synnax_key);
        auto [channel_vec, err] = client->channels.retrieve(channel_keys);
        if (err) {
            cfg.field_err(
                "channels",
                "failed to retrieve channels for task: " + err.message()
            );
            return;
        }
        auto remote_channels = map_channel_Keys(channel_vec);
        auto [device, dev_err] = client->hardware.retrieve_device(this->device_key);
        if (dev_err) {
            cfg.field_err(
                "device",
                "failed to retrieve device for task: " + dev_err.message()
            );
            return;
        }
        for (auto &ch: this->channels) {
            const auto &remote_ch = remote_channels.at(ch->synnax_key);
            ch->bind_remote_info(remote_ch, device.location);
            if (ch->ch.index != 0) this->indexes.insert(ch->ch.index);
        }
    }

    static std::pair<CounterReadTaskConfig, xerrors::Error> parse(
        std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task,
        const common::TimingConfig timing_cfg
    ) {
        auto parser = xjson::Parser(task.config);
        return {CounterReadTaskConfig(client, parser, timing_cfg), parser.error()};
    }

    [[nodiscard]] std::vector<synnax::Channel> sy_channels() const {
        std::vector<synnax::Channel> chs;
        chs.reserve(this->channels.size());
        for (const auto &ch: this->channels)
            chs.push_back(ch->ch);
        return chs;
    }

    [[nodiscard]]

    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        const TaskHandle handle
    ) const {
        for (const auto &ch: this->channels)
            if (auto err = ch->apply(dmx, handle)) return err;

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
        for (const auto &ch: this->channels)
            keys.push_back(ch->ch.key);
        for (const auto &idx: this->indexes)
            keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = synnax::data_saving_writer_mode(this->data_saving),
            .enable_auto_commit = true,
        };
    }

    [[nodiscard]] std::unique_ptr<common::SampleClock> sample_clock() const {
        return std::make_unique<common::HardwareTimedSampleClock>(
            common::HardwareTimedSampleClockConfig::create_simple(
                sample_rate,
                stream_rate,
                this->timing.correct_skew
            )
        );
    }
};

/// @brief an internal source that we pass to the acquisition pipeline that manages
/// reading data from the NI DAQmx library and writing it to Synnax.
template<typename T>
struct ReadTaskSource final : common::Source {
    CounterReadTaskConfig cfg;
    std::unique_ptr<hardware::daqmx::CounterReader> hw;
    synnax::WriterConfig writer_cfg;
    breaker::Breaker breaker;
    size_t samples_skewed = 0;

    ReadTaskSource(
        CounterReadTaskConfig &&cfg,
        std::unique_ptr<hardware::daqmx::CounterReader> &&hw
    ):
        cfg(std::move(cfg)),
        hw(std::move(hw)),
        writer_cfg(this->cfg.writer()),
        breaker(breaker::default_config(cfg.task_name)) {}

    /// @brief implements the common::Source interface and reads data from NI
    /// hardware and writes it into a Synnax frame.
    common::SourceReadResult read(breaker::Breaker &breaker, synnax::Frame &frame) override {
        const auto samples = this->cfg.samples_per_chan;
        common::SourceReadResult result;
        const int n_channels = this->cfg.channels.size();
        int32_t n_samples_read = 0;
        // Allocate buffers for each channel
        std::vector<std::vector<T>> channel_buffers(n_channels);
        std::vector<T *> buffer_ptrs(n_channels);
        for (int i = 0; i < n_channels; i++) {
            channel_buffers[i].resize(samples);
            buffer_ptrs[i] = channel_buffers[i].data();
        }

        // Interleaved read buffer
        std::vector<T> interleaved_buffer(samples * n_channels);

        result.error = this->hw->read(
            interleaved_buffer.data(),
            samples,
            &n_samples_read
        );

        if (result.error) {
            result.error = translate_error(result.error);
            return result;
        }

        // Deinterleave data
        for (int32_t i = 0; i < n_samples_read; i++)
            for (int j = 0; j < n_channels; j++)
                buffer_ptrs[j][i] = interleaved_buffer[i * n_channels + j];

        // Create frame with channel data
        auto indexes = std::set(this->cfg.indexes);
        for (size_t i = 0; i < this->cfg.channels.size(); i++) {
            auto &ch = this->cfg.channels[i];
            if (indexes.contains(ch->ch.key)) continue;
            frame.append(
                ch->ch.key,
                synnax::Series(
                    channel_buffers[i].data(),
                    n_samples_read,
                    ch->ch.data_type
                )
            );
        }

        // Generate timestamps for index channels
        auto now = synnax::TimeStamp::now();
        auto index_series = synnax::Series(
            std::vector<std::uint64_t>(
                n_samples_read,
                (now - synnax::TimeSpan(n_samples_read * this->cfg.sample_rate.period()))
                    .value
            ),
            synnax::TIMESTAMP
        );
        for (auto &idx: this->cfg.indexes)
            frame.append(idx, index_series);

        result.acquired = n_samples_read;
        return result;
    }

    [[nodiscard]] synnax::WriterConfig writer() const override { return this->writer_cfg; }

    [[nodiscard]] std::vector<synnax::ChannelKey> indexes() const override {
        return std::vector(this->cfg.indexes.begin(), this->cfg.indexes.end());
    }

    [[nodiscard]] std::unique_ptr<common::SampleClock> sample_clock() const override {
        return this->cfg.sample_clock();
    }
};
}
