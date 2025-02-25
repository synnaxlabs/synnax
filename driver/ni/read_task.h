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
#include "driver/ni/channels.h"
#include "driver/queue/ts_queue.h"
#include "driver/pipeline/acquisition.h"
#include "driver/task/task.h"

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
    const std::size_t samples_per_channel;
    /// @brief whether the task should be software timed.
    const bool software_timed;
    /// @brief the buffer size to allocate to read from the hardware.
    std::size_t buffer_size = 0;
    /// @brief the indexes of the channels in the task.
    std::set<synnax::ChannelKey> indexes;
    /// @brief the configurations for each channel in the task.
    std::vector<std::unique_ptr<InputChan> > channels;

    /// @brief Move constructor to allow transfer of ownership
    ReadTaskConfig(ReadTaskConfig &&other) noexcept:
        data_saving(other.data_saving),
        device_key(std::move(other.device_key)),
        sample_rate(other.sample_rate),
        stream_rate(other.stream_rate),
        timing_source(std::move(other.timing_source)),
        samples_per_channel(other.samples_per_channel),
        software_timed(other.software_timed),
        buffer_size(other.buffer_size),
        indexes(std::move(other.indexes)),
        channels(std::move(other.channels)) {
    }

    /// @brief delete copy constructor and copy assignment to prevent accidental copies.
    ReadTaskConfig(const ReadTaskConfig &) = delete;

    const ReadTaskConfig &operator=(const ReadTaskConfig &) = delete;

    explicit ReadTaskConfig(
        std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg,
        std::string task_type
    ):
        data_saving(cfg.optional<bool>("data_saving", false)),
        device_key(cfg.optional<std::string>("device", "cross-device")),
        sample_rate(telem::Rate(cfg.required<float>("sample_rate"))),
        stream_rate(telem::Rate(cfg.required<float>("stream_rate"))),
        timing_source(cfg.optional<std::string>("timing_source", "none")),
        samples_per_channel(std::floor(sample_rate.value / stream_rate.value)),
        software_timed(this->timing_source == "none" && task_type == "ni_digital_read"),
        channels(cfg.map<std::unique_ptr<InputChan> >(
            "channels",
            [&](xjson::Parser &ch_cfg) -> std::pair<std::unique_ptr<InputChan>, bool> {
                auto ch = parse_input_chan(ch_cfg, {});
                return {std::move(ch), ch->enabled};
            }
        )) {
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
            auto remote_ch = remote_channels.at(ch->synnax_key);
            auto dev = devices[ch->dev_key];
            ch->bind_remote_info(remote_ch, dev.location);
            this->buffer_size += this->samples_per_channel * remote_ch.data_type.
                    density();
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
        if (!this->software_timed)
            dmx->CfgSampClkTiming(
                handle,
                this->timing_source == "none" ? nullptr : this->timing_source.c_str(),
                this->sample_rate.value,
                DAQmx_Val_Rising,
                DAQmx_Val_ContSamps,
                this->sample_rate.value
            );
        for (const auto &ch: this->channels)
            if (auto err = ch->apply(dmx, handle)) return err;
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

/// @brief a thing shim on top of NI DAQMX that allows us to use different read
/// interfaces for analog and digital tasks. It also allows us to mock the hardware
/// during testing.
template<typename T>
struct HardwareInterface {
    virtual ~HardwareInterface() = default;

    /// @brief starts the task.
    virtual xerrors::Error start() const = 0;

    /// @brief stops the task.
    virtual xerrors::Error stop() const = 0;

    /// @brief reads data from the hardware.
    virtual xerrors::Error read(
        size_t samples_per_channel,
        std::vector<T> &data,
        size_t data_size
    ) = 0;
};

/// @brief a base implementation of the hardware interface that uses the NI DAQMX
/// in the background.
template<typename T>
struct DAQmxHardwareInterface : HardwareInterface<T> {
protected:
    /// @brief the handle for the task.
    TaskHandle task_handle;
    /// @brief the NI DAQmx API.
    std::shared_ptr<SugaredDAQmx> dmx;

    DAQmxHardwareInterface(TaskHandle task_handle, std::shared_ptr<SugaredDAQmx> dmx):
        task_handle(task_handle), dmx(std::move(dmx)) {
    }

public:
    xerrors::Error start() const override {
        return this->dmx->StartTask(task_handle);
    }

    xerrors::Error stop() const override {
        return this->dmx->StopTask(task_handle);
    }
};

/// @brief a hardware interface for digital tasks.
struct DigitalHardwareInterface final : DAQmxHardwareInterface<uint8_t> {
    DigitalHardwareInterface(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ): DAQmxHardwareInterface(task_handle, dmx) {
    }

    xerrors::Error read(
        const size_t samples_per_channel,
        std::vector<uint8_t> &digital_data,
        size_t digital_data_size
    ) override {
        return this->dmx->ReadDigitalLines(
            this->task_handle,
            samples_per_channel,
            -1,
            DAQmx_Val_GroupByChannel,
            digital_data.data(),
            digital_data.size(),
            nullptr,
            nullptr,
            nullptr
        );
    }
};

/// @brief a hardware interface for analog tasks.
struct AnalogHardwareInterface final : DAQmxHardwareInterface<double> {
    AnalogHardwareInterface(
        const std::shared_ptr<SugaredDAQmx> &dmx,
        TaskHandle task_handle
    ): DAQmxHardwareInterface(task_handle, dmx) {
    }

    xerrors::Error read(
        const size_t samples_per_channel,
        std::vector<double> &analog_data,
        size_t analog_data_size
    ) override {
        return this->dmx->ReadAnalogF64(
            this->task_handle,
            samples_per_channel,
            -1,
            DAQmx_Val_GroupByChannel,
            analog_data.data(),
            analog_data.size(),
            nullptr,
            nullptr
        );
    }
};

/// @brief a read task that can pull from both analog and digital channels.
template<typename T>
class ReadTask final : public task::Task {
    /// @brief the raw synnax task configuration.
    const synnax::Task task;
    /// @brief the parsed configuration for the task.
    const ReadTaskConfig cfg;
    /// @brief the task context used to communicate state changes back to Synnax.
    std::shared_ptr<task::Context> ctx;
    /// @brief tare middleware used for taring values.
    std::shared_ptr<pipeline::TareMiddleware> tare_mw;
    /// @brief the current task state.
    task::State state;

public:
    explicit ReadTask(
        synnax::Task task,
        const std::shared_ptr<task::Context> &ctx,
        ReadTaskConfig cfg,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<HardwareInterface<T> > hw_api
    ): task(std::move(task)),
       cfg(std::move(cfg)),
       ctx(ctx),
       tare_mw(std::make_shared<pipeline::TareMiddleware>(
           this->cfg.writer_config().channels)),
       source(std::make_shared<Source>(*this, std::move(hw_api))),
       pipe(
           this->ctx->client,
           this->cfg.writer_config(),
           this->source,
           breaker_cfg
       ) {
        this->pipe.add_middleware(this->tare_mw);
    }

    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key);
        else if (cmd.type == "tare") this->tare_mw->tare(cmd.args);
    }

    void stop() override { this->stop(""); }

    void stop(const std::string &cmd_key) {
        this->state.key = cmd_key;
        this->source->breaker.stop();
        this->source->sample_thread.join();
        this->pipe.stop();
        this->ctx->set_state(this->state);
    }

    void start(const std::string &cmd_key) {
        this->state.key = cmd_key;
        this->pipe.start();
    }

    class Source final : public pipeline::Source {
    public:
        ReadTask &task;

        struct DataPacket {
            std::vector<T> data;
            telem::TimeStamp t0;
            telem::TimeStamp tf;
            int32 samples_read_per_channel;
        };

        /// @brief interface used to read data from the hardware.
        std::unique_ptr<HardwareInterface<T> > hw_api;
        /// @brief a separate thread to acquire samples in.
        std::thread sample_thread;
        /// @brief breaker for hte separate thread.
        breaker::Breaker sample_thread_breaker;
        /// @brief queue used to exchange data between the separate thread and the main thread.
        TSQueue<DataPacket> queue;
        /// @brief a timer that is used in the software timed mode.
        loop::Timer timer;
        /// @brief automatically infer the data type from the template parameter. This
        /// will either be UINT8_T or FLOAT64_T. We use this to appropriately cast
        /// the data read from the hardware.
        const telem::DataType data_type = telem::DataType::infer<T>();

        explicit Source(
            ReadTask &task,
            std::unique_ptr<HardwareInterface<T> > hw
        ): task(task), hw_api(std::move(hw)) {
        }

        void stopped_with_err(const xerrors::Error &err) override {
        };

        void acquire_data() {
            if (const auto err = hw_api->start()) {
                this->task.state.variant = "error";
                this->task.state.details["message"] = err.message();
                this->task.ctx->set_state(this->task.state);
                return;
            }
            this->task.state.variant = "success";
            this->task.state.details["message"] = "Task started successfully";
            while (this->sample_thread_breaker.running()) {
                DataPacket data_packet;
                data_packet.data.resize(this->task.cfg.buffer_size);
                data_packet.t0 = telem::TimeStamp::now();
                if (this->task.cfg.software_timed)
                    this->timer.wait(this->sample_thread_breaker);
                if (const auto err = hw_api->read(
                    this->task.cfg.samples_per_channel,
                    data_packet.data,
                    this->task.cfg.buffer_size
                )) {
                    this->task.state.variant = "error";
                    this->task.state.details["message"] = err.message();
                    break;
                }
                data_packet.tf = telem::TimeStamp::now();
                queue.enqueue(data_packet);
            }
            const auto err = hw_api->stop();
            if (this->task.state.variant == "error") return;
            if (err) {
                this->task.state.variant = "error";
                this->task.state.details["message"] = err.message();
                return;
            }
            this->task.state.variant = "success";
            this->task.state.details["message"] = "task stopped successfully";
        }

        std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
            if (!this->sample_thread_breaker.running()) {
                this->sample_thread_breaker.start();
                sample_thread = std::thread(&Source::acquire_data, this);
            }
            auto [d, ok] = this->queue.dequeue();
            if (!ok) return std::make_pair(synnax::Frame(), xerrors::NIL);

            auto f = synnax::Frame(this->task.cfg.channels.size());

            const size_t count = this->task.cfg.samples_per_channel;
            size_t data_index = 0;

            const auto buf = d.data.data();
            for (const auto &ch: this->task.cfg.channels) {
                auto s = telem::Series(ch->ch.data_type, count);
                const size_t start = data_index * count;
                // Hot path - if the data matches, we can just copy it.
                if (s.data_type == this->data_type) s.write(buf + start, count);
                    // Otherwise we need to cast it.
                else
                    for (int i = 0; i < count; ++i)
                        s.write(s.data_type.cast(buf[start + i]));
                f.emplace(ch->synnax_key, std::move(s));
                data_index++;
            }

            if (!this->task.cfg.indexes.empty()) {
                const auto index_data = telem::Series::linspace(d.t0, d.tf, count);
                for (const auto &idx: this->task.cfg.indexes)
                    f.emplace(idx, std::move(index_data.deep_copy()));
            }

            return std::make_pair(std::move(f), xerrors::NIL);
        }
    };

    std::shared_ptr<Source> source;
    /// @brief the pipeline used to read data from the hardware and pipe it to Synnax.
    pipeline::Acquisition pipe;

    std::string name() override { return task.name; }
};

template<typename T, typename Constructor>
static std::pair<std::unique_ptr<task::Task>, xerrors::Error> configure_read(
    const std::shared_ptr<SugaredDAQmx> &dmx,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task &task
) {
    auto [cfg, cfg_err] = ReadTaskConfig::parse(ctx->client, task);
    if (cfg_err) return {nullptr, cfg_err};
    TaskHandle task_handle;
    if (const auto err = dmx->CreateTask("", &task_handle)) return {nullptr, err};
    if (const auto err = cfg.apply(dmx, task_handle)) return {nullptr, err};
    if (const auto err = cycle_task_to_detect_cfg_errors(dmx, task_handle))
        return {nullptr, err};
    return {
        std::make_unique<ReadTask<T> >(
            task,
            ctx,
            std::move(cfg),
            breaker::default_config(task.name),
            std::make_unique<Constructor>(dmx, task_handle)
        ),
        xerrors::NIL
    };
}
}
