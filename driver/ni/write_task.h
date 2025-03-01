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
#include <map>
#include <set>

/// module
#include "client/cpp/synnax.h"
#include "x/cpp/breaker/breaker.h"
#include "x/cpp/loop/loop.h"
#include "x/cpp/xjson/xjson.h"

/// internal
#include "driver/ni/channel/channels.h"
#include "driver/ni/ni.h"
#include "driver/ni/hardware/hardware.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"

namespace ni {
/// @brief WriteTaskConfig is the configuration for creating an NI Digital or Analog
/// Write Task.
struct WriteTaskConfig {
    /// @brief the key of the device the task is writing to.
    const std::string device_key;
    /// @brief the rate at which the task will publish the states of the outputs
    /// back to the Synnax cluster.
    const telem::Rate state_rate;
    /// @brief whether data saving is enabled for the task.
    const bool data_saving;
    /// @brief a map of command channel keys to the configurations for each output
    /// channel in the task.
    std::map<synnax::ChannelKey, std::unique_ptr<channel::Output>> channels;
    /// @brief the index channel keys for all the state channels. This is used
    /// to make sure we write correct timestamps for each state channel.
    std::set<synnax::ChannelKey> state_indexes;
    /// @brief a map of channel keys to their index positions within the tasks
    /// write buffer. This map is only valid after apply() has been called on the
    /// configuration.
    std::unordered_map<synnax::ChannelKey, std::size_t> buf_indexes;

    /// @brief move constructor to deal with output channel unique pointers.
    WriteTaskConfig(WriteTaskConfig &&other) noexcept:
        device_key(other.device_key),
        state_rate(other.state_rate),
        data_saving(other.data_saving),
        channels(std::move(other.channels)),
        state_indexes(std::move(other.state_indexes)),
        buf_indexes(std::move(other.buf_indexes)) {
    }

    /// @brief delete copy constructor and copy assignment to prevent accidental
    /// copies.
    WriteTaskConfig(const WriteTaskConfig &) = delete;

    const WriteTaskConfig &operator=(const WriteTaskConfig &) = delete;

    /// @brief constructs the configuration from the provided JSON parser, using the
    /// client to fetch any remote data from the Synnax cluster.
    /// @param client - Synnax client used to fetch remote data from the cluster.
    /// @param cfg - The JSON configuration for the task.
    /// @details any errors encountered while parsing the configuration will be
    /// added as field errors to the provided parser. The caller should use
    /// cfg.error() after this constructor in order to check for these errors.
    explicit WriteTaskConfig(
        const std::shared_ptr<synnax::Synnax> &client,
        xjson::Parser &cfg
    ): device_key(cfg.required<std::string>("device")),
       state_rate(telem::Rate(cfg.required<float>("state_rate"))),
       data_saving(cfg.optional<bool>("data_saving", false)) {
        cfg.iter("channels", [&](xjson::Parser &ch_cfg) {
            auto ch = channel::parse_output(ch_cfg);
            if (ch->enabled) this->channels[ch->cmd_ch_key] = std::move(ch);
        });
        if (channels.empty()) {
            cfg.field_err("channels", "task must have at least one enabled channel");
            return;
        }
        std::vector<synnax::ChannelKey> state_keys;
        state_keys.reserve(this->channels.size());
        std::unordered_map<synnax::ChannelKey, synnax::ChannelKey> state_to_cmd;
        size_t index = 0;
        for (const auto &[_, ch]: this->channels) {
            state_keys.push_back(ch->state_ch_key);
            state_to_cmd[ch->state_ch_key] = ch->cmd_ch_key;
            buf_indexes[ch->cmd_ch_key] = index++;
        }
        auto [dev, err] = client->hardware.retrieve_device(this->device_key);
        if (err) {
            cfg.field_err("device", "failed to retrieve device " + err.message());
            return;
        }
        auto [state_channels, ch_err] = client->channels.retrieve(state_keys);
        if (ch_err) {
            cfg.field_err("channels",
                          "failed to retrieve state channels: " + ch_err.message());
            return;
        }
        for (const auto &state_ch: state_channels) {
            auto &ch = this->channels[state_to_cmd[state_ch.key]];
            ch->bind_remote_info(state_ch, dev.location);
            if (state_ch.index != 0) this->state_indexes.insert(state_ch.index);
        }
    }

    /// @brief returns the configuration necessary for opening the writer
    /// to communicate state values back to Synnax.
    [[nodiscard]] synnax::WriterConfig writer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size() + state_indexes.size());
        for (const auto &[_, ch]: channels) keys.push_back(ch->state_ch_key);
        for (const auto &idx: state_indexes) keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .start = telem::TimeStamp::now(),
            .mode = synnax::data_saving_writer_mode(this->data_saving),
            .enable_auto_commit = true
        };
    }

    /// @brief returns the configuration necessary for opening a streamer to
    /// receive values form Synnax.
    [[nodiscard]] synnax::StreamerConfig streamer_config() const {
        std::vector<synnax::ChannelKey> keys;
        keys.reserve(channels.size());
        for (const auto &[_, ch]: channels) keys.push_back(ch->cmd_ch_key);
        return synnax::StreamerConfig{.channels = keys};
    }

    /// @brief parses the task from the given configuration, returning an error
    /// if the task could not be parsed.
    static std::pair<WriteTaskConfig, xerrors::Error> parse(
        const std::shared_ptr<synnax::Synnax> &client,
        const synnax::Task &task
    ) {
        auto parser = xjson::Parser(task.config);
        return {WriteTaskConfig(client, parser), parser.error()};
    }

    /// @brief applies the configuration to the given DAQmx task.
    xerrors::Error apply(
        const std::shared_ptr<daqmx::SugaredAPI> &dmx,
        TaskHandle task_handle
    ) {
        for (const auto &[_, ch]: channels)
            if (const auto err = ch->apply(dmx, task_handle)) return err;
        return xerrors::NIL;
    }
};

template<typename T>
class WriteTask final : public task::Task {
    /// @brief the configuration for the task.
    const WriteTaskConfig cfg;
    /// @brief the pipeline used to receive commands from Synnax and write them to
    /// the device.
    pipeline::Control cmd_write_pipe;
    /// @brief the pipeline used to receive state changes from the device and write
    /// to Synnax.
    pipeline::Acquisition state_write_pipe;
    /// @brief the hardware interface for writing data
    std::unique_ptr<hardware::Writer<T>> hw_writer;
    /// @brief the current state of all the outputs. This is shared between
    /// the command sink and state source.
    std::unordered_map<synnax::ChannelKey, telem::SampleValue> chan_state;
    /// @brief used to lock concurrent access to the channel state.
    std::mutex chan_state_lock;
    /// @brief the current state of the task.
    ni::TaskStateHandler state;


    /// @brief StateSource is passed to the state pipeline in order to continually
    /// communicate the current output states to Synnax.
    class StateSource final : public pipeline::Source {
    public:
        /// @brief constructs a StateSource bound to the parent WriteTask.
        explicit StateSource(WriteTask &parent):
            p(parent), state_timer(parent.cfg.state_rate) {
        }

    private:
        /// @brief the parent write task.
        WriteTask &p;
        /// @brief a timer that is used to control the rate at which the state is
        /// propagated.
        loop::Timer state_timer;

        /// @brief called when the write() function returns a critical hardware error
        /// or when the pipeline fails to read samples from the cluster. In any case,
        /// we register the error in state, and then stop the entire task by calling
        /// the parent stop() method. This ensures that we stop the acquisition pipeline
        /// used for state values as well.
        void stopped_with_err(const xerrors::Error &err) override {
            this->p.state.error(err);
            this->p.stop("", false);
        }

        /// @brief implements pipeline::Source to return the current state of the
        /// outputs in teh task.
        std::pair<Frame, xerrors::Error> read(breaker::Breaker &breaker) override {
            this->state_timer.wait(breaker);
            std::lock_guard lock{this->p.chan_state_lock};
            auto fr = synnax::Frame(
                this->p.chan_state,
                this->p.chan_state.size() + this->p.cfg.state_indexes.size()
            );
            if (!this->p.cfg.state_indexes.empty()) {
                const auto idx_ser = telem::Series(telem::TimeStamp::now());
                for (const auto idx: this->p.cfg.state_indexes)
                    fr.emplace(idx, idx_ser.deep_copy());
            }
            return {std::move(fr), xerrors::NIL};
        }
    };

    /// @brief sink is passed to the command pipeline in order to receive incoming
    /// data from Synnax, write it to the device, and update the state.
    class CommandSink final : public pipeline::Sink {
    public:
        /// @brief constructs a CommandSink bound to the provided parent WriteTask.
        explicit CommandSink(WriteTask &parent):
            p(parent),
            buf(parent.cfg.channels.size()) {
        }

    private:
        /// @brief automatically infer the data type from the template parameter. This
        /// will either be UINT8_T or FLOAT64_T. We use this to appropriately cast
        /// the data read from the hardware.
        const telem::DataType data_type = telem::DataType::infer<T>();
        /// @brief the parent write task.
        WriteTask &p;
        /// @brief a pre-allocated write buffer that is flushed to the device every
        /// time a command is provided.
        std::vector<T> buf;

        /// @brief called when the write() function returns a critical hardware error
        /// or when the pipeline fails to read samples from the cluster. In any case,
        /// we register the error in state, and then stop the entire task by calling
        /// the parent stop() method. This ensures that we stop the acquisition pipeline
        /// used for state values as well.
        void stopped_with_err(const xerrors::Error &err) override {
            this->p.state.error(err);
            this->p.stop("", false);
        }

        /// @brief implements pipeline::Sink to write the incoming frame to the
        /// underlying hardware. If the values are successfully written, updates
        /// the write tasks state to match the output values.
        xerrors::Error write(const synnax::Frame &frame) override {
            if (frame.empty()) return xerrors::NIL;
            for (const auto &[key, series]: frame) {
                auto it = this->p.cfg.buf_indexes.find(key);
                if (it != this->p.cfg.buf_indexes.end())
                    buf[it->second] = telem::cast<T>(series.at(-1));
            }
            if (const auto err = this->p.hw_writer->write(buf)) {
                if (daqmx::ANALOG_WRITE_OUT_OF_BOUNDS.matches(err)) {
                    this->p.state.send_warning(err.message());
                    return xerrors::NIL;
                }
                return err;
            }

            std::lock_guard lock{this->p.chan_state_lock};
            for (const auto &[key, series]: frame) {
                const auto it = this->p.cfg.channels.find(key);
                if (it != this->p.cfg.channels.end()) {
                    this->p.chan_state[it->second->state_ch_key] = it->second->state_ch.
                            data_type.cast(series.at(-1));
                }
            }
            return xerrors::NIL;
        }
    };

public:
    /// @brief base constructor that takes in pipeline factories to allow the
    /// caller to stub cluster communication during tests.
    explicit WriteTask(
        const synnax::Task &task,
        const std::shared_ptr<task::Context> &ctx,
        WriteTaskConfig cfg,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<hardware::Writer<T>> hw_writer,
        const std::shared_ptr<pipeline::WriterFactory> &writer_factory,
        const std::shared_ptr<pipeline::StreamerFactory> &streamer_factory
    ):
        cfg(std::move(cfg)),
        cmd_write_pipe(
            streamer_factory,
            this->cfg.streamer_config(),
            std::make_shared<CommandSink>(*this),
            breaker_cfg
        ),
        state_write_pipe(
            writer_factory,
            this->cfg.writer_config(),
            std::make_shared<StateSource>(*this),
            breaker_cfg
        ),
        hw_writer(std::move(hw_writer)),
        state(ctx, task) {
        chan_state.reserve(this->cfg.channels.size());
        for (const auto &[_, ch]: this->cfg.channels)
            chan_state[ch->state_ch_key] = ch->state_ch.data_type.cast(0);
    }

    /// @brief primary constructor that uses the task context's Synnax client for
    /// cluster communication.
    explicit WriteTask(
        synnax::Task task,
        const std::shared_ptr<task::Context> &ctx,
        WriteTaskConfig cfg,
        const breaker::Config &breaker_cfg,
        std::unique_ptr<hardware::Writer<T>> hw_writer
    ): WriteTask(
        std::move(task),
        ctx,
        std::move(cfg),
        breaker_cfg,
        std::move(hw_writer),
        std::make_shared<pipeline::SynnaxWriterFactory>(ctx->client),
        std::make_shared<pipeline::SynnaxStreamerFactory>(ctx->client)
    ) {
    }

    /// @brief implements task::Task to execute teh provided command on the task.
    void exec(task::Command &cmd) override {
        if (cmd.type == "start") this->start(cmd.key);
        else if (cmd.type == "stop") this->stop(cmd.key, false);
    }

    /// @brief implements task::Task to stop the task.
    void stop(const bool will_reconfigure) override {
        this->stop("", will_reconfigure);
    }

    /// @brief stops the task.
    /// @param cmd_key - A reference to the command key used to execute the stop. Will
    /// be used internally to communicate the task state.
    /// @param will_reconfigure whether the task will be reconfigured after it was stopped.
    void stop(const std::string &cmd_key, const bool will_reconfigure) {
        this->cmd_write_pipe.stop();
        this->state_write_pipe.stop();
        this->state.error(this->hw_writer->stop());
        if (will_reconfigure) return;
        this->state.send_stop(cmd_key);
    }

    /// @brief starts the task.
    /// @param cmd_key - A reference to the command key used to execute the start. Will
    /// be used internally to communicate the task state.
    void start(const std::string &cmd_key) {
        if (!this->state.error(this->hw_writer->start())) {
            this->cmd_write_pipe.start();
            this->state_write_pipe.start();
        }
        this->state.send_start(cmd_key);
    }

    /// @brief implements task::Task to return the task's name.
    std::string name() override { return this->state.task.name; }
};
}
