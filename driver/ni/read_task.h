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
#include <set>
#include <sstream>
#include <string>
#include <vector>

/// module
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/ni/channel/channels.h"
#include "driver/ni/daqmx/nidaqmx.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/ni/ni.h"
#include "driver/task/common/read_task.h"
#include "driver/task/common/sample_clock.h"

namespace ni {
/// @brief the configuration for a read task.
struct ReadTaskConfig : common::BaseReadTaskConfig {
    /// @brief the device key that will be used for the channels in the task. Analog
    /// read tasks can specify multiple devices. In this case, the device key field
    /// is empty and automatically set to "cross-device".
    const std::string device_key;
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
    /// @brief the amount of sample skew needed to trigger a warning that the Synnax
    /// cannot keep up with the amount of clock skew.
    size_t skew_warn_on_count;
    /// @brief the device resource name(s) used for DAQmx API calls. For single device
    /// tasks this will have one entry, for cross-device tasks it will have multiple.
    /// Stored as pairs of (location, model) e.g., ("E103Mod1", "NI 9229").
    std::vector<std::pair<std::string, std::string>> device_resource_names;

    /// @brief Move constructor to allow transfer of ownership
    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        common::BaseReadTaskConfig(std::move(other)),
        device_key(other.device_key),
        timing_source(other.timing_source),
        samples_per_chan(other.samples_per_chan),
        software_timed(other.software_timed),
        indexes(std::move(other.indexes)),
        channels(std::move(other.channels)),
        skew_warn_on_count(other.skew_warn_on_count),
        device_resource_names(std::move(other.device_resource_names)) {}

    /// @brief delete copy constructor and copy assignment to prevent accidental
    /// copies.
    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg,
        const std::string &task_type,
        common::TimingConfig timing_cfg = common::TimingConfig()
    ):
        BaseReadTaskConfig(cfg, timing_cfg),
        device_key(cfg.optional<std::string>("device", "cross-device")),
        timing_source(cfg.optional<std::string>("timing_source", "")),
        samples_per_chan(sample_rate / stream_rate),
        software_timed(this->timing_source.empty() && task_type == "ni_digital_read"),
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
        std::unordered_map<std::string, synnax::Device> devices;
        if (this->device_key != "cross-device") {
            auto [device, err] = client->hardware.retrieve_device(this->device_key);
            if (err) {
                cfg.field_err(
                    "device",
                    "failed to retrieve device for task: " + err.message()
                );
                return;
            }
            devices[device.key] = device;
            // Store the DAQmx device location and model
            this->device_resource_names.push_back({device.location, device.model});
            VLOG(1) << "[ni.read_task] using device for validation: " << device.location
                    << " (" << device.model << ")";
        } else {
            std::vector<std::string> dev_keys;
            for (const auto &ch: this->channels)
                dev_keys.push_back(ch->dev_key);
            auto [devices_vec, dev_err] = client->hardware.retrieve_devices(dev_keys);
            if (dev_err) {
                cfg.field_err(
                    "device",
                    "failed to retrieve devices for task: " + dev_err.message()
                );
                return;
            }
            devices = map_device_keys(devices_vec);
            // Store DAQmx device locations and models for all devices
            for (const auto &dev: devices_vec) {
                this->device_resource_names.push_back({dev.location, dev.model});
                VLOG(1) << "[ni.read_task] using device for validation: "
                        << dev.location << " (" << dev.model << ")";
            }
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
        const synnax::Task &task,
        const common::TimingConfig timing_cfg
    ) {
        auto parser = xjson::Parser(task.config);
        return {ReadTaskConfig(client, parser, task.type, timing_cfg), parser.error()};
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
        if (this->software_timed) return xerrors::NIL;

        // Validate sample rate against device minimum(s)
        VLOG(1) << "[ni.read_task] validating sample rate for "
                << this->device_resource_names.size() << " device(s)";
        for (const auto &[location, model]: this->device_resource_names) {
            float64 min_rate = 0.0;
            auto err = dmx->GetDeviceAttributeDouble(
                location.c_str(),
                DAQmx_Dev_AI_MinRate,
                &min_rate
            );
            if (err) {
                LOG(WARNING) << "[ni.read_task] failed to query min rate for device "
                             << location << ": " << err.message();
                continue;
            }
            VLOG(1) << "[ni.read_task] device " << location << " (" << model
                    << ") min_rate: " << min_rate
                    << " Hz, configured: " << this->sample_rate.hz() << " Hz";
            if (this->sample_rate.hz() < min_rate) {
                std::ostringstream msg;
                msg << "configured sample rate (" << this->sample_rate.hz()
                    << " Hz) is below device minimum (" << min_rate << " Hz) for "
                    << location << " (" << model << ")";
                return xerrors::Error("ni.sample_rate_too_low", msg.str());
            }
        }

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
        if (this->software_timed)
            return std::make_unique<common::SoftwareTimedSampleClock>(
                this->stream_rate
            );
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
/// the lifecycle of this task.
template<typename T>
class ReadTaskSource final : public common::Source {
public:
    /// @brief constructs a source bound to the provided parent read task.
    explicit ReadTaskSource(
        ReadTaskConfig cfg,
        std::unique_ptr<hardware::Reader<T>> hw_reader
    ):
        cfg(std::move(cfg)),
        buf(this->cfg.samples_per_chan * this->cfg.channels.size()),
        hw_reader(std::move(hw_reader)),
        sample_clock(this->cfg.sample_clock()) {}

private:
    /// @brief the raw synnax task configuration.
    /// @brief the parsed configuration for the task.
    const ReadTaskConfig cfg;
    /// @brief the buffer used to read data from the hardware. This vector is
    /// pre-allocated and reused.
    std::vector<T> buf;
    /// @brief interface used to read data from the hardware.
    std::unique_ptr<hardware::Reader<T>> hw_reader;
    /// @brief the timestamp at which the hardware task was started. We use this to
    /// interpolate the correct timestamps of recorded samples.
    std::unique_ptr<common::SampleClock> sample_clock;
    /// @brief the error accumulated from the latest read. Primarily used to
    /// determine whether we've just recovered from an error state.
    xerrors::Error curr_read_err = xerrors::NIL;

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        return this->cfg.sy_channels();
    }

    xerrors::Error start() override {
        this->sample_clock->reset();
        auto err = this->hw_reader->start();
        return err;
    }

    xerrors::Error stop() override { return this->hw_reader->stop(); }

    xerrors::Error restart() {
        if (const auto err = this->hw_reader->stop()) return err;
        if (const auto err = this->hw_reader->start()) return err;
        this->sample_clock->reset();
        return xerrors::NIL;
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        return this->cfg.writer();
    }

    common::ReadResult read(breaker::Breaker &breaker, synnax::Frame &fr) override {
        common::ReadResult res;
        const auto n_channels = this->cfg.channels.size();
        const auto n_samples = this->cfg.samples_per_chan;
        common::initialize_frame(fr, this->cfg.channels, this->cfg.indexes, n_samples);

        auto start = this->sample_clock->wait(breaker);
        const auto hw_res = this->hw_reader->read(n_samples, this->buf);
        // A non-zero skew means that our application cannot keep up with the
        // hardware acquisition rate.
        if (std::abs(hw_res.skew) > this->cfg.skew_warn_on_count)
            res.warning = common::skew_warning(hw_res.skew);

        auto prev_read_err = this->curr_read_err;
        this->curr_read_err = translate_error(hw_res.error);
        res.error = this->curr_read_err;

        if (this->curr_read_err.matches(daqmx::REQUIRES_RESTART)) {
            res.error = translate_error(this->restart());
            this->curr_read_err = res.error;
            return res;
        }

        if (res.error) return res;
        if (prev_read_err) {
            this->sample_clock->reset();
            return res;
        }

        const auto end = this->sample_clock->end();
        common::transfer_buf(this->buf, fr, n_channels, n_samples);
        common::generate_index_data(
            fr,
            this->cfg.indexes,
            start,
            end,
            n_samples,
            n_channels
        );
        return res;
    }
};
}
