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
#include <utility>
#include <set>
#include <stdio.h>

#include "LJM_Utilities.h"
#include "LabJackMModbusMap.h"

#include "nlohmann/json.hpp"

#include "client/cpp/synnax.h"

#include "driver/config/config.h"
#include "driver/errors/errors.h"
#include "driver/task/task.h"
#include "driver/pipeline/acquisition.h"
#include "driver/pipeline/middleware.h"
#include "driver/queue/ts_queue.h"
#include "driver/breaker/breaker.h"

namespace labjack {
///////////////////////////////////////////////////////////////////////////////////
//                            Thermocouple Config                                //
///////////////////////////////////////////////////////////////////////////////////
const int SINGLE_ENDED = 199; // default negative channel for single ended signals

///@brief look up table mapping LJM TC Type to TC AIN_EF index
// Thermocouple type:		 B  E  J  K  N  R  S  T  C
const int TC_INDEX_LUT[9] = {28, 20, 21, 22, 27, 23, 25, 24, 30};

struct TCConfig {
    ///@brief The thermocouple type
    // Supported TC types are:
    //     LJM_ttB (val=6001)
    //     LJM_ttE (val=6002)
    //     LJM_ttJ (val=6003)
    //     LJM_ttK (val=6004)
    //     LJM_ttN (val=6005)
    //     LJM_ttR (val=6006)
    //     LJM_ttS (val=6007)
    //     LJM_ttT (val=6008)
    //     LJM_ttC (val=6009)
    // Note that the values above do not align with the AIN_EF index values
    // or order. We use a lookup table provided by labjack to convert our
    // thermocouple constant to the correct index when using the AIN_EF
    // Lookup table: TC_INDNEX_LUT[ x - 60001] = AIN_EF_INDEX
    long type;

    ///@brief locations of the single ended or differential signal
    // For T7s only:
    // For differential signals, pos_chan should be an even num AIN and
    // neg_chan will be pos_chan + 1.
    // For single ended signals, neg_chan should be set to 199
    int pos_chan;
    int neg_chan;

    ///@brief  Modbus Address to read the CJC sensor
    int cjc_addr;

    ///@brief slope of CJC Voltage to temperature conversion (Kelvin/Volts).
    // if using device temp (cjc_addr is TEMPERATURE_DEVICE_K), set to 1
    // If using a LM34 on some AIN, set to 55.56
    float cjc_slope;

    ///@brief OFffset for CJC temp (Kelvin)
    // If cjc_addr = TEMPERATURE_DEVICE_K. set to 0
    // If using InAmp or expansion board, might need to adjust it a few degrees
    // If using LM34 connected to an AIN, set to 255.37
    float cjc_offset;

    ///@brief units for the thermocouple reading
    std::string units;

    explicit TCConfig(
        long type = 0,
        int pos_chan = 0,
        int neg_chan = 0,
        int cjc_addr = 0,
        float cjc_slope = 0.0f,
        float cjc_offset = 0.0f,
        std::string units = "K"
    ) : type(type),
        pos_chan(pos_chan),
        neg_chan(neg_chan),
        cjc_addr(cjc_addr),
        cjc_slope(cjc_slope),
        cjc_offset(cjc_offset),
        units(units) {
    }

    explicit TCConfig(config::Parser &parser)
        : pos_chan(parser.required<int>("pos_chan")),
          neg_chan(parser.optional<int>("neg_chan", SINGLE_ENDED)),
          cjc_slope(parser.required<float>("cjc_slope")),
          cjc_offset(parser.required<float>("cjc_offset")),
          units(parser.required<std::string>("units")) {
        auto tc_type = parser.required<std::string>("thermocouple_type");

        if (tc_type == "B")
            this->type = LJM_ttB;
        else if (tc_type == "E")
            this->type = LJM_ttE;
        else if (tc_type == "J")
            this->type = LJM_ttJ;
        else if (tc_type == "K")
            this->type = LJM_ttK;
        else if (tc_type == "N")
            this->type = LJM_ttN;
        else if (tc_type == "R")
            this->type = LJM_ttR;
        else if (tc_type == "S")
            this->type = LJM_ttS;
        else if (tc_type == "T")
            this->type = LJM_ttT;
        else if (tc_type == "C")
            this->type = LJM_ttC;
        else
            LOG(ERROR) << "Invalid thermocouple type: " << tc_type;

        auto cjc_source = parser.required<std::string>("cjc_source");

        if (cjc_source == "TEMPERATURE_DEVICE_K")
            cjc_addr = LJM_TEMPERATURE_DEVICE_K_ADDRESS;
        else if (cjc_source == "TEMPERATURE_AIR_K")
            cjc_addr = LJM_TEMPERATURE_AIR_K_ADDRESS;
        else if (cjc_source.find("AIN") != std::string::npos) {
            // address for modbus register for analog port is port number x 2
            int port_num = std::stoi(cjc_source.substr(3));
            cjc_addr = port_num * 2;
        }
    }
}; // TCConfig

struct ReaderChannelConfig {
    ///@brief The location of the channel on device (e.g. AIN0, FIO4, etc.)
    std::string location;
    ///@brief Whether to read from this channel
    bool enabled = true;
    synnax::DataType data_type;
    ///@brief Synnax channel key
    uint32_t key;
    ///@brief voltage range
    double range = 10.0;
    int neg_chan = 199;
    int pos_chan;
    ///@brief channel type (e.g. AIN, DIN, TC)
    std::string channel_type = "";
    ///@brief Thermocouple configuration if applicable
    TCConfig tc_config;

    ReaderChannelConfig() = default;

    explicit ReaderChannelConfig(config::Parser &parser)
        : enabled(parser.optional<bool>("enabled", true)),
          data_type(parser.optional<std::string>("data_type", "float32")),
          key(parser.required<uint32_t>("channel")),
          range(parser.optional<double>("range", 10.0)),
          neg_chan(parser.optional<double>("neg_chan", SINGLE_ENDED)),
          pos_chan(parser.optional<int>("pos_chan", 0)),
          channel_type(parser.optional<std::string>("type", "")),
          location(parser.optional<std::string>("port", "")) {
        if (!parser.ok())
            LOG(ERROR) << "Failed to parse reader channel config: " << parser.error_json().dump(4);

        if (this->channel_type == "TC") {
            this->tc_config = TCConfig(parser);
            // temparature : AIN#_EF_READ_A register
            // voltage     : AIN#_EF_READ_B register
            // CJC temp    : AIN#_EF_READ_C register
            this->location = "AIN" + std::to_string(this->tc_config.pos_chan) + "_EF_READ_A";
        }
    }
};


///////////////////////////////////////////////////////////////////////////////////
//                                   ReaderConfig                                //
///////////////////////////////////////////////////////////////////////////////////
struct ReaderConfig {
    ///@brief The type of device (e.g. T4, T7, T8, etc.)
    std::string device_type;
    ///@brief Key of device on synnax server
    std::string device_key;
    std::vector<ReaderChannelConfig> channels;
    synnax::Rate sample_rate = synnax::Rate(1);
    synnax::Rate stream_rate = synnax::Rate(1);
    synnax::ChannelKey task_key;
    std::set<uint32_t> index_keys;
    std::string serial_number; // used to open devices
    ///@brief The type of connection (i.e. USB, Ethernet, or WIFI)
    std::string connection_type; // used to open devices
    ///@brief map of locations on device to synnax channel keys
    std::map<std::string, uint32_t> channel_map;
    std::vector<std::string> phys_channels;
    std::vector<ReaderChannelConfig> tc_channels;
    std::set<uint32_t> tc_index_keys;
    ///@brief whether to persist data to disk
    bool data_saving;

    ReaderConfig() = default;

    explicit ReaderConfig(config::Parser &parser)
        : device_type(parser.optional<std::string>("type", "")),
          device_key(parser.required<std::string>("device")),
          sample_rate(synnax::Rate(parser.optional<int>("sample_rate", 1))),
          stream_rate(synnax::Rate(parser.optional<int>("stream_rate", 1))),
          serial_number(parser.required<std::string>("device")),
          connection_type(parser.optional<std::string>("connection_type", "")),
          data_saving(parser.optional<bool>("data_saving", false)
          ) {

        LOG(INFO) << "ReaderConfig: " << parser.error_json().dump(4);
        if (!parser.ok())
            LOG(ERROR) << "Failed to parse reader channel config: " << parser.error_json().dump(4);

        parser.iter("channels", [this](config::Parser &channel_parser) {
            auto channel = labjack::ReaderChannelConfig(channel_parser);

            if (channel.enabled && channel.channel_type != "TC") {
                channels.emplace_back(channel);
                this->phys_channels.push_back(channel.location);
            } else if (channel.enabled && channel.channel_type == "TC") {
                tc_channels.emplace_back(channel);
            }

            this->channel_map[channel.location] = channel.key;
        });
    }
};

///////////////////////////////////////////////////////////////////////////////////
//                                   ReaderSource                                //
///////////////////////////////////////////////////////////////////////////////////
class ReaderSource : public pipeline::Source {
public:
    explicit ReaderSource(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task task,
        const ReaderConfig &reader_config
    ) : ctx(ctx),
        task(task),
        reader_config(reader_config) {
        auto breaker_config = breaker::Config{
            .name = task.name,
            .base_interval = 1 * SECOND,
            .max_retries = 20,
            .scale = 1.2,
        };
        this->breaker = breaker::Breaker(breaker_config);
    }

    ~ReaderSource() override;

    std::vector<synnax::ChannelKey> get_channel_keys();

    std::vector<synnax::ChannelKey> get_ai_channel_keys();

    void stopped_with_err(const freighter::Error &err) override;

    std::pair<Frame, freighter::Error> read(breaker::Breaker &breaker);

    freighter::Error stop(const std::string &cmd_key);

    freighter::Error start(const std::string &cmd_key);

    bool ok();

private:
    void init();

    void init_stream();

    void init_tcs();

    void configure_tc_ain_ef(TCConfig tc_config);

    int check_err(int err, std::string caller);

    void write_to_series(
        synnax::Series &series,
        double &data,
        synnax::DataType data_type
    );

    void acquire_data();

    std::pair<Frame, freighter::Error> read_stream(breaker::Breaker &breaker);

    std::pair<Frame, freighter::Error> read_cmd_response(breaker::Breaker &breaker);

    int handle;
    ReaderConfig reader_config;
    std::shared_ptr<task::Context> ctx;
    breaker::Breaker breaker;
    synnax::Task task;

    /// @brief shared resources between daq sampling thread and acquisition thread
    struct DataPacket {
        std::vector<double> data;
        uint64_t t0; // initial timestamp
        uint64_t tf; // final timestamp
    };

    TSQueue<DataPacket> data_queue;
    std::thread sample_thread;
    std::vector<int> port_addresses;
    int buffer_size = 0;
    int num_samples_per_chan = 0;
    bool ok_state = true;
    std::mutex mutex;
};

///////////////////////////////////////////////////////////////////////////////////
//                                    ReaderTask                                 //
///////////////////////////////////////////////////////////////////////////////////
class ReaderTask final : public task::Task {
public:
    explicit ReaderTask(
        const std::shared_ptr<task::Context> &ctx,
        synnax::Task task,
        std::shared_ptr<labjack::ReaderSource> labjack_source,
        std::shared_ptr<pipeline::Source> source,
        synnax::WriterConfig writer_config,
        const breaker::Config breaker_config);

    void exec(task::Command &cmd) override;

    void stop() override;

    void stop(const std::string &cmd_key);

    void start(const std::string &cmd_key);

    std::string name() override { return task.name; }

    static std::unique_ptr<task::Task> configure(
        const std::shared_ptr<task::Context> &ctx,
        const synnax::Task &task
    );

private:
    std::atomic<bool> running = false;
    std::shared_ptr<task::Context> ctx;
    synnax::Task task;
    pipeline::Acquisition read_pipe;
    std::shared_ptr<labjack::ReaderSource> source;
    std::shared_ptr<pipeline::TareMiddleware> tare_mw;
}; // class ReaderTask
}
