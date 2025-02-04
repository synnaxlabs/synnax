// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <string>
#include <vector>
#include <map>
#include <queue>
#include <utility>
#include <memory>
#include <atomic>
#include <thread>
#include <set>
#include <condition_variable>

#include "nidaqmx/nidaqmx_api.h"
#include "nlohmann/json.hpp"
#include "client/cpp/synnax.h"
#include "driver/ni/channels.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/task/task.h"
#include "driver/breaker/breaker.h"
#include "driver/loop/loop.h"
#include "driver/config/config.h"
#include "driver/ni/ni.h"

namespace ni {
// Forward declarations
class Analog;

///////////////////////////////////////////////////////////////////////////////////
//                                   WriterChannelConfig                         //
///////////////////////////////////////////////////////////////////////////////////
///@brief defines the configuration for a channel in a writer task.
struct WriterChannelConfig {
    ///@brief the name of the channel in the format <device_name>/<port>/<line> for
    /// digital channels or <device_name>/<port> for analog channels
    std::string name;
    ///@brief whether the channel is enabled
    bool enabled = true;
    ///@brief the data type of the channel
    synnax::DataType data_type;
    ///@brief the key of the channel
    uint32_t channel_key;
    ///@brief the key of the state channel
    uint32_t state_channel_key;
    ///@brief the port of the channel on NI device
    std::string port;
    ///@brief the line of the channel on NI device (for digital channels)
    std::string line;
    ///@brief the NI channel object (for analog channels)
    std::shared_ptr<ni::Analog> ni_channel; 
    ///@brief the type of the channel (for analog channels)
    std::string channel_type;

    WriterChannelConfig() = default;

    explicit WriterChannelConfig(
        config::Parser &parser,
        std::string device_name,
        bool is_digital,
        TaskHandle task_handle,
        synnax::TaskKey task_key,
        std::shared_ptr<task::Context> ctx
    ) {
        enabled = parser.optional<bool>("enabled", true);
        channel_key = parser.required<uint32_t>("cmd_channel");
        state_channel_key = parser.required<uint32_t>("state_channel");
        channel_type = parser.optional<std::string>("type", "");

        auto port_num = parser.required<std::uint64_t>("port");
        if (is_digital) {
            // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
            auto line_num = parser.required<std::uint64_t>("line");
            port = "port" + std::to_string(port_num);
            line = "line" + std::to_string(line_num);
            name = device_name + "/" + port + "/" + line;
        } else {
            // analog channel names are formatted: <device_name>/ao<port_number>
            port = "ao" + std::to_string(port_num);
            name = device_name + "/" + port;

            ni_channel = AnalogOutputChannelFactory::create_channel(channel_type, parser, task_handle, name);
            if (ni_channel == nullptr) {
                std::string msg = "Channel " + name + " has an unrecognized type: " + channel_type;
                ctx->set_state({
                    .task = task_key,
                    .variant = "error",
                    .details = {
                        {"running", false},
                        {"message", msg}
                    }
                });
                LOG(ERROR) << "[ni.writer] " << msg;
            }
        }
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                   WriterConfig                                //
///////////////////////////////////////////////////////////////////////////////////
///@brief defines the configuration for a writer task.
struct WriterConfig {
    ///@brief the type of the device
    std::string device_type;
    ///@brief the name of the device. 
    std::string device_name;
    ///@brief the key of the device to retrieve from Synnax Server
    std::string device_key;
    ///@brief the name of the task
    std::string task_name;
    ///@brief the rate at which the state channel is written to
    float state_rate = 0;
    ///@brief the key of the task
    synnax::ChannelKey task_key;
    ///@brief the channels in the task
    std::vector<WriterChannelConfig> channels;
    ///@brief the keys of the state channels
    std::vector<synnax::ChannelKey> state_channel_keys;
    ///@brief the keys of the drive command channels
    std::vector<synnax::ChannelKey> drive_cmd_channel_keys;
    ///@brief the key of the state index channel
    synnax::ChannelKey state_index_key;

    ///@brief queues to maintain changes to state channels
    std::queue<synnax::ChannelKey> modified_state_keys;
    std::queue<std::uint8_t> digital_modified_state_values;
    std::queue<double> analog_modified_state_values;

    WriterConfig() = default;

    explicit WriterConfig(
        config::Parser &parser,
        const std::shared_ptr<task::Context> &ctx,
        bool is_digital,
        TaskHandle task_handle,
        synnax::TaskKey task_key
    ) {
        device_key = parser.required<std::string>("device");
        state_rate = parser.required<float>("state_rate");
        task_name = parser.optional<std::string>("task_name", "");

        auto [dev, err] = ctx->client->hardware.retrieveDevice(device_key);
        if (err != freighter::NIL) {
            LOG(ERROR) << "Failed to retrieve device with key " << device_key;
            return;
        }
        device_name = dev.location;

        parser.iter("channels", [&](config::Parser &channel_parser) {
            auto channel = WriterChannelConfig(
                channel_parser,
                device_name,
                is_digital,
                task_handle,
                task_key,
                ctx
            );

            if (!channel.enabled) return;
            if (!channel_parser.ok()) {
                LOG(ERROR) << "Failed to parse channel config: " << channel_parser.error_json().dump(4);
                return;
            }

            channels.push_back(channel);
            drive_cmd_channel_keys.push_back(channel.channel_key);
            state_channel_keys.push_back(channel.state_channel_key);
        });

        if (!parser.ok()) {
            ctx->set_state({
                .task = task_key,
                .variant = "error",
                .details = {
                    {"running", false},
                    {"message", parser.error_json().dump(4)}
                }
            });
            LOG(ERROR) << "Failed to parse channel config: " << parser.error_json().dump(4);
        }
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
///@brief a source that maintains the state of write task ports and writes them back to
/// the Synnax server. Templated to support both digital and analog state channels.
template<typename T>
class StateSource final : public pipeline::Source {
public:
    explicit StateSource() = default;

    explicit StateSource(
        float state_rate,
        synnax::ChannelKey &state_index_key,
        std::vector<synnax::ChannelKey> &state_channel_keys
    );

    std::pair<synnax::Frame, freighter::Error> read(breaker::Breaker &breaker) override;

    synnax::Frame get_state();

    void update_state(
        std::queue<synnax::ChannelKey> &modified_state_keys,
        std::queue<T> &modified_state_values
    );

private:
    std::mutex state_mutex;
    std::condition_variable waiting_reader;
    synnax::Rate state_rate = synnax::Rate(1);
    std::map<synnax::ChannelKey, T> state_map;
    synnax::ChannelKey state_index_key;
    loop::Timer timer;
};

// Type aliases
using DigitalStateSource = StateSource<uint8_t>;
using AnalogStateSource = StateSource<double>;


///////////////////////////////////////////////////////////////////////////////////
//                                    WriteSink                                  //
///////////////////////////////////////////////////////////////////////////////////
// Base class for common functionality
class WriteSink : public pipeline::Sink {
public:
    WriteSink(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    ) : dmx(dmx),
        task_handle(task_handle),
        ctx(ctx),
        task(task),
        err_info({}) {
    }

    virtual ~WriteSink() {
        clear_task();
    }

    freighter::Error cycle();

    freighter::Error start(const std::string &cmd_key);

    freighter::Error stop(const std::string &cmd_key);

    std::vector<synnax::ChannelKey> get_cmd_channel_keys();

    std::vector<synnax::ChannelKey> get_state_channel_keys();

    bool ok();

protected:
    // Keep implementation-specific methods protected
    void get_index_keys();

    void jsonify_error(std::string);

    void stopped_with_err(const freighter::Error &err) override;

    void log_error(std::string err_msg);

    void clear_task();

    int check_err(int32 error, std::string caller);

    // Pure virtual methods that derived classes must implement
    virtual freighter::Error start_ni() = 0;

    virtual freighter::Error stop_ni() = 0;

    virtual int init() = 0;

    virtual freighter::Error format_data(const synnax::Frame &frame) = 0;

    // Protected members accessible to derived classes
    const std::shared_ptr<DAQmx> dmx;
    TaskHandle task_handle = 0;
    int buffer_size = 0;
    int num_samples_per_channel = 0;
    uint64_t num_channels = 0;
    json err_info;
    bool ok_state = true;
    std::shared_ptr<task::Context> ctx;
    WriterConfig writer_config;
    breaker::Breaker breaker;
    synnax::Task task;
    std::map<std::string, std::string> channel_map;
};

///////////////////////////////////////////////////////////////////////////////////
//                                    DigitalWriteSink                           //
///////////////////////////////////////////////////////////////////////////////////
class DigitalWriteSink final : public WriteSink {
public:
    explicit DigitalWriteSink(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    ~DigitalWriteSink();

    freighter::Error write(synnax::Frame frame) override;

    std::shared_ptr<ni::DigitalStateSource> writer_state_source;

private:
    freighter::Error start_ni() override;

    freighter::Error stop_ni() override;

    int init() override;

    freighter::Error format_data(const synnax::Frame &frame) override;

    uint8_t *write_buffer = nullptr;
};

///////////////////////////////////////////////////////////////////////////////////
//                                 AnalogWriteSink                               //
///////////////////////////////////////////////////////////////////////////////////
class AnalogWriteSink final : public WriteSink {
public:
    explicit AnalogWriteSink(
        const std::shared_ptr<DAQmx> &dmx,
        TaskHandle task_handle,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

    ~AnalogWriteSink();

    freighter::Error write(synnax::Frame frame) override;

    std::shared_ptr<ni::AnalogStateSource> writer_state_source;

private:
    freighter::Error start_ni() override;

    freighter::Error stop_ni() override;

    int init() override;

    freighter::Error format_data(const synnax::Frame &frame) override;

    double *write_buffer = nullptr;
};

///////////////////////////////////////////////////////////////////////////////////
//                                    WriterTask                                 //
///////////////////////////////////////////////////////////////////////////////////
class WriterTask final : public task::Task {
public:
    explicit WriterTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        std::shared_ptr<pipeline::Sink> sink,
        std::shared_ptr<WriteSink> ni_sink,
        std::shared_ptr<pipeline::Source> state_source,
        synnax::WriterConfig state_writer_config,
        synnax::StreamerConfig cmd_streamer_config,
        const breaker::Config breaker_config
    );

    void exec(task::Command &cmd) override;

    void stop() override;

    void stop(const std::string &cmd_key);

    void start(const std::string &key);

    bool ok();

    std::string name() override { return task.name; }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<DAQmx> &dmx,
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

private:
    std::atomic<bool> running = false;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    pipeline::Control cmd_write_pipe;
    pipeline::Acquisition state_write_pipe;
    bool ok_state = true;
    std::shared_ptr<WriteSink> sink;
};
}
