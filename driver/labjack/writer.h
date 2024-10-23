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
#include <thread>
#include <stdio.h>

#include "LJM_Utilities.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"


#include "driver/errors/errors.h"
#include "driver/task/task.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/control.h"
#include "driver/breaker/breaker.h"
#include "driver/loop/loop.h"

namespace labjack{
struct out_state{
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
        const synnax::Rate state_rate, // TODO: make this synnax::Rate?
        const synnax::ChannelKey &state_index_key,
        const std::map<synnax::ChannelKey, labjack::out_state> state_map
    );

    std::pair<synnax::Frame, freighter::Error> read(breaker::Breaker &breaker) override;

    synnax::Frame get_state(); // TODO: maybe i don't need this

    void update_state(synnax::Frame frame);


private:
    std::mutex state_mutex;
    std::condition_variable waiting_reader;
    synnax::Rate state_rate = synnax::Rate(1);
    std::map<synnax::ChannelKey, labjack::out_state> state_map; // alll values are
    synnax::ChannelKey state_index_key;
    loop::Timer timer;
};  // class StateSource

///////////////////////////////////////////////////////////////////////////////////
//                                   WriterChannelConfig                         //
///////////////////////////////////////////////////////////////////////////////////
struct WriterChannelConfig{
    std::string location;
    bool enabled = true;
    synnax::DataType data_type;
    uint32_t cmd_key;  // TODO: change channel type to synanx::channelKEY or whatever it is
    uint32_t state_key;
    std::string channel_type = "";
    int port;

    WriterChannelConfig() = default;

    explicit WriterChannelConfig(config::Parser &parser)
        : enabled(parser.optional<bool>("enabled", true)),
          data_type(parser.required<std::string>("data_type")),
          cmd_key(parser.required<uint32_t>("cmd_key")),
          state_key(parser.required<uint32_t>("state_key")),
          channel_type(parser.optional<std::string>("type", "")),
          port(parser.optional<int>("port", 0)){
        this->location = this->channel_type + std::to_string(this->port);
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                   WriterConfig                                //
///////////////////////////////////////////////////////////////////////////////////
struct WriterConfig{
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
    synnax::ChannelKey state_index_key;

    WriterConfig() = default;

    explicit WriterConfig(config::Parser &parser)
        :  device_type(parser.required<std::string>("device_type")),
           device_key(parser.required<std::string>("device_key")),
           state_rate(synnax::Rate(parser.optional<int>("state_rate", 1))),
           serial_number(parser.optional<std::string>("serial_number", "")),
           connection_type(parser.optional<std::string>("connection_type", "")),
           data_saving(parser.optional<bool>("data_saving", false)
       ){
        // Parse the channels
        parser.iter("channels", [this](config::Parser &channel_parser){
            channels.emplace_back(WriterChannelConfig(channel_parser));

            double initial_val = 0.0;
            if(channels.back().data_type == synnax::SY_UINT8){
                initial_val = 1.0;
            }
            initial_state_map[channels.back().cmd_key] = labjack::out_state{
                .location = channels.back().location,
                .state = initial_val,
                .data_type = channels.back().data_type,
                .state_key = channels.back().state_key
            };
        });
    }
}; // struct WriterConfig

///////////////////////////////////////////////////////////////////////////////////
//                                   WriteSink                                   //
///////////////////////////////////////////////////////////////////////////////////
class WriteSink final : public pipeline::Sink{
public:

    explicit WriteSink(
            const std::shared_ptr<task::Context> &ctx,
            const synnax::Task &task,
            const labjack::WriterConfig &writer_config
        );

    ~WriteSink();

    void init();

    freighter::Error write(synnax::Frame frame) override;

    freighter::Error stop(const std::string &cmd_key);

    freighter::Error start(const std::string &cmd_key);

    std::vector<synnax::ChannelKey> get_cmd_channel_keys();

    std::vector<synnax::ChannelKey> get_state_channel_keys();

    std::shared_ptr<labjack::StateSource> state_source;

    void get_index_keys();



private:
    int handle;
    std::shared_ptr<task::Context> ctx;
    WriterConfig writer_config;
    breaker::Breaker breaker;
    synnax::Task task;

}; // class DigitalWriteSink


} // namespace labjack
// TODO: add a cycle function to catch errors before hand?