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
#include "driver/ni/ni.h"

namespace ni {
static xerrors::Error translate_error(const xerrors::Error &err) {
    if (daqmx::APPLICATION_TOO_SLOW.matches(err))
        return {
            xerrors::Error(
                driver::CRITICAL_HARDWARE_ERROR,
                "the network cannot keep up with the stream rate specified. try making the sample rate a higher multiple of the stream rate"
            )
        };
    return err;
}

/// @brief used to regulate the acquisition speed of a task, and provide timing
/// information for generating timestamps.
struct SampleClock {
    virtual ~SampleClock() = default;
    /// @brief resets the sample clock, making it ready for task startup.
    virtual void reset() {};
    /// @brief waits for the next acquisition loop to begin, returning the timestamp
    /// of the first sample.
    virtual telem::TimeStamp wait(breaker::Breaker &breaker) = 0;
    /// @brief ends the acquisition loop, interpolating an ending timestamp based
    /// on the number of samples read.
    virtual telem::TimeStamp end(size_t n_read) = 0;
};

/// @brief a sample clock that regulates the acquisition rate at the application
/// layer by using a software timer.
class SoftwareTimedSampleClock final : public SampleClock {
    /// @brief the timer used to regulate the acquisition rate.
    loop::Timer timer;
public:
    explicit SoftwareTimedSampleClock(const telem::Rate &stream_rate):
        timer(stream_rate) {
    }

    telem::TimeStamp wait(breaker::Breaker &breaker) override {
        this->timer.wait(breaker);
        return telem::TimeStamp::now();
    }

    telem::TimeStamp end(const size_t _) override {
        return telem::TimeStamp::now();
    }
};

/// @brief a sample clock that relies on an external, steady hardware clock to
/// regulate the acquisition rate. Timestamps are interpolated based on a fixed
/// sample rate.
class HardwareTimedSampleClock final : public SampleClock {
    /// @brief the sample rate of the task.
    const telem::Rate sample_rate;
    /// @brief the high water-mark for the next acquisition loop.
    telem::TimeStamp high_water{};


    explicit HardwareTimedSampleClock(const telem::Rate sample_rate):
        sample_rate(sample_rate) {
    }

    void reset() override {
        this->high_water = telem::TimeStamp::now();
    }

    telem::TimeStamp wait(breaker::Breaker &_) override {
        return this->high_water;
    }

    telem::TimeStamp end(const size_t n_read) override {
        const auto end = this->high_water + (n_read - 1) * this->sample_rate.period();
        this->high_water = end + this->sample_rate.period();
        return end;
    }
};

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
       samples_per_chan(
           static_cast<size_t>(std::floor((sample_rate / stream_rate).hz()))),
       software_timed(this->timing_source.empty() && task_type == "ni_digital_read"),
       channels(cfg.map<std::unique_ptr<channel::Input>>(
           "channels",
           [&](xjson::Parser &ch_cfg) -> std::pair<std::unique_ptr<channel::Input>,
       bool> {
               auto ch = channel::parse_input(ch_cfg, {});
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
        auto remote_channels = channel_keys_map(channel_vec);
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
            devices = device_keys_map(devices_vec);
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

    [[nodiscard]] std::unique_ptr<SampleClock> sample_clock() const {
        return this->software_timed
                   ? std::make_unique<SoftwareTimedSampleClock>(this->stream_rate)
                   : std::make_unique<HardwareTimedSampleClock>(this->sample_rate);
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
    std::unique_ptr<hardware::Reader<T>> hw_reader;
    /// @brief the timestamp at which the hardware task was started. We use this to
    /// interpolate the correct timestamps of recorded samples.
    std::unique_ptr<SampleClock> sample_clock;
    /// @brief handles communicating the task state back to the cluster.
    ni::TaskStateHandler state;

    /// @brief an internal source that we pass to the acquisition pipeline that manages
    /// the lifecycle of this task.
    class Source final : public pipeline::Source {
    public:
        /// @brief constructs a source bound to the provided parent read task.
        explicit Source(ReadTask &parent):
            p(parent),
            timer(parent.cfg.stream_rate),
            buffer(parent.cfg.samples_per_chan * parent.cfg.channels.size()) {
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
            this->p.state.error(err);
            this->p.state.error(this->p.hw_reader->stop());
            this->p.state.send_stop("");
        }

        std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
            auto start = this->p.sample_clock->wait(breaker);
            const auto [n, err] = this->p.hw_reader->read(
                this->p.cfg.samples_per_chan,
                buffer
            );
            if (err) return {Frame(), translate_error(err)};
            auto end = this->p.sample_clock->end(n);
            synnax::Frame f(this->p.cfg.channels.size());
            size_t i = 0;
            for (const auto &ch: this->p.cfg.channels)
                f.emplace(
                    ch->synnax_key,
                    telem::Series::cast(ch->ch.data_type, buffer.data() + i++ * n, n)
                );
            if (!this->p.cfg.indexes.empty()) {
                const auto index_data = telem::Series::linspace(start, end, n);
                for (const auto &idx: this->p.cfg.indexes)
                    f.emplace(idx, std::move(index_data.deep_copy()));
            }
            return std::make_pair(std::move(f), xerrors::NIL);
        }
    };

public:
    /// @brief base constructor that takes in a pipeline writer factory to allow the
    /// caller to stub cluster communication during tests.
    explicit ReadTask(
        synnax::Task task,
        const std::shared_ptr<task::Context> &ctx,
        ReadTaskConfig cfg,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<hardware::Reader<T>> reader,
        const std::shared_ptr<pipeline::WriterFactory> &factory
    ): cfg(std::move(cfg)),
       tare_mw(std::make_shared<pipeline::TareMiddleware>(
           this->cfg.writer().channels
       )),
       pipe(
           factory,
           this->cfg.writer(),
           std::make_shared<Source>(*this),
           breaker_cfg
       ),
       hw_reader(std::move(reader)),
       sample_clock(this->cfg.sample_clock()),
       state(ctx, task) {
        this->pipe.use(this->tare_mw);
    }

    /// @brief primary constructor that uses the task context's Synnax client in order
    /// to communicate with the cluster.
    explicit ReadTask(
        synnax::Task task,
        const std::shared_ptr<task::Context> &ctx,
        ReadTaskConfig cfg,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<hardware::Reader<T>> reader
    ): ReadTask(
        std::move(task),
        ctx,
        std::move(cfg),
        breaker_cfg,
        std::move(reader),
        std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client)
    ) {
    }

    /// @brief executes the given command on the task.
    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key, false);
        else if (cmd.type == "tare") this->tare_mw->tare(cmd.args);
    }

    /// @brief stops the task.
    void stop(const bool will_reconfigure) override {
        this->stop("", will_reconfigure);
    }

    /// @brief stops the task, using the given command key as reference for
    /// communicating success state.
    void stop(const std::string &cmd_key, const bool will_reconfigure) {
        if (const auto was_running = this->pipe.stop(); !was_running) return;
        this->state.error(this->hw_reader->stop());
        if (will_reconfigure) return;
        this->state.send_stop(cmd_key);
    }

    /// @brief starts the task, using the given command key as a reference for
    /// communicating task state.
    void start(const std::string &cmd_key) {
        if (this->pipe.running()) return;
        this->sample_clock.reset();
        if (const auto ok = this->state.error(this->hw_reader->start()); !ok)
            this->pipe.start();
        this->state.send_start(cmd_key);
    }

    /// @brief implements task::Task.
    std::string name() override { return this->state.task.name; }
};
}
