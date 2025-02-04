// Copyright 2025 Synnax Labs, Inc.
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
#include <thread>
#include <stdio.h>
#include <set>
#include <utility>

#include "LJM_Utilities.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"


#include "driver/errors/errors.h"
#include "driver/task/task.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/breaker/breaker.h"
#include "driver/loop/loop.h"
#include "driver/config/config.h"
#include "driver/labjack/util.h"

namespace labjack {
struct out_state {
    std::string location = "";
    double state = 0.0;
    synnax::DataType data_type = synnax::FLOAT64;
    synnax::ChannelKey state_key = 0;
};

///////////////////////////////////////////////////////////////////////////////////
//                                    StateSource                                //
///////////////////////////////////////////////////////////////////////////////////
class StateSource final : public pipeline::Source {
public:
    explicit StateSource() = default;

    explicit StateSource(
        const synnax::Rate state_rate,
        std::vector<synnax::ChannelKey> state_index_keys,
        const std::map<synnax::ChannelKey, labjack::out_state> state_map
    );

    std::pair<synnax::Frame, freighter::Error> read(breaker::Breaker &breaker) override;

    synnax::Frame get_state();

    void update_state(synnax::Frame frame);

private:
    std::mutex state_mutex;
    std::condition_variable waiting_reader;
    synnax::Rate state_rate = synnax::Rate(1);
    std::map<synnax::ChannelKey, labjack::out_state> state_map;
    std::vector<synnax::ChannelKey> state_index_keys;
    loop::Timer timer;
}; // class StateSource

///////////////////////////////////////////////////////////////////////////////////
//                                   WriterChannelConfig                         //
///////////////////////////////////////////////////////////////////////////////////
struct WriterChannelConfig {
    std::string location;
    bool enabled = true;
    synnax::DataType data_type;
    uint32_t cmd_key;
    uint32_t state_key;
    std::string channel_type = "";

    WriterChannelConfig() = default;

    explicit WriterChannelConfig(config::Parser &parser)
        : enabled(parser.optional<bool>("enabled", true)),
          data_type(parser.optional<std::string>("data_type", "uint8")),
          cmd_key(parser.required<uint32_t>("cmd_key")),
          state_key(parser.required<uint32_t>("state_key")),
          channel_type(parser.optional<std::string>("type", "")),
          location(parser.optional<std::string>("port", "")) {
        if (!parser.ok())
            LOG(ERROR) << "Failed to parse writer channel config: " << parser.error_json().dump(4);
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                   WriterConfig                                //
///////////////////////////////////////////////////////////////////////////////////
struct WriterConfig {
    std::string device_type;
    std::string device_key;
    std::vector<WriterChannelConfig> channels;
    synnax::Rate state_rate = synnax::Rate(1);
    std::string serial_number; // used to open devices
    std::string connection_type;
    bool data_saving;
    std::string task_name;
    synnax::ChannelKey task_key;
    std::map<synnax::ChannelKey, labjack::out_state> initial_state_map;
    std::vector<synnax::ChannelKey> state_index_keys;

    WriterConfig() = default;

    explicit WriterConfig(
        config::Parser &parser,
        const std::shared_ptr<task::Context> &ctx
    )
        : device_type(parser.optional<std::string>("type", "")),
          device_key(parser.required<std::string>("device")),
          state_rate(synnax::Rate(parser.optional<int>("state_rate", 1))),
          serial_number(parser.required<std::string>("device")),
          connection_type(parser.optional<std::string>("connection_type", "")),
          data_saving(parser.optional<bool>("data_saving", false)
          ) {
        if (!parser.ok())
            LOG(ERROR) << "Failed to parse writer config: " << parser.error_json().dump(4);

        parser.iter("channels", [this, ctx](config::Parser &channel_parser) {
            auto channel = WriterChannelConfig(channel_parser);

            if (channel.enabled) channels.emplace_back(channel);
            else return;

            auto [channel_info, err] = ctx->client->channels.retrieve(channel.cmd_key);
            if (err) {
                LOG(ERROR) << "Failed to retrieve channel info for key " << channel.cmd_key;
                return;
            }
            channel.data_type = channel_info.data_type;


            /// digital outputs start active high
            double initial_val = 0.0;

            initial_state_map[channel.cmd_key] = labjack::out_state{
                .location = channel.location,
                .state = initial_val,
                .data_type = channel.data_type,
                .state_key = channel.state_key
            };
        });
    }
}; // struct WriterConfig

///////////////////////////////////////////////////////////////////////////////////
//                                   WriteSink                                   //
///////////////////////////////////////////////////////////////////////////////////
class WriteSink final : public pipeline::Sink {
public:
    explicit WriteSink(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        const labjack::WriterConfig &writer_config,
        std::shared_ptr<labjack::DeviceManager> device_manager
    );

    ~WriteSink();

    void init();

    freighter::Error write(synnax::Frame frame) override;

    freighter::Error stop(const std::string &cmd_key);

    freighter::Error start(const std::string &cmd_key);

    std::vector<synnax::ChannelKey> get_cmd_channel_keys();

    std::vector<synnax::ChannelKey> get_state_channel_keys();

    std::shared_ptr<labjack::StateSource> state_source;

    std::vector<synnax::ChannelKey> get_index_keys();

    int check_err(int err, std::string caller);

    bool ok();

    void open_device();

private:
    void log_err(std::string err_msg);

    int handle;
    std::shared_ptr<task::Context> ctx;
    WriterConfig writer_config;
    breaker::Breaker breaker;
    synnax::Task task;
    bool ok_state = true;
    std::shared_ptr<labjack::DeviceManager> device_manager;
}; // class WriteSink

///////////////////////////////////////////////////////////////////////////////////
//                                    WriterTask                                 //
///////////////////////////////////////////////////////////////////////////////////
class WriterTask final : public task::Task {
public:
    explicit WriterTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        std::shared_ptr<pipeline::Sink> sink,
        std::shared_ptr<labjack::WriteSink> labjack_sink,
        std::shared_ptr<pipeline::Source> state_source,
        synnax::WriterConfig writer_config,
        synnax::StreamerConfig streamer_config,
        const breaker::Config breaker_config
    );

    void exec(task::Command &cmd) override;

    void stop() override;

    void stop(const std::string &cmd_key);

    void start(const std::string &cmd_key);

    std::string name() override { return task.name; }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task,
        std::shared_ptr<labjack::DeviceManager> device_manager
    );

private:
    std::atomic<bool> running = false;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    pipeline::Control cmd_pipe;
    pipeline::Acquisition state_pipe;
    std::shared_ptr<labjack::WriteSink> sink;
};
} // namespace labjack
