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
#include "driver/queue/ts_queue.h"
#include "driver/breaker/breaker.h"


namespace labjack{

    typedef enum { DEGK ='K', DEGC = 'C', DEGF = 'F' } TCUnits;

    ///@brief look up table mapping LJM TC Type to TC AIN_EF index
    // Thermocouple type:		 B  E  J  K  N  R  S  T  C
    const int TC_INDEX_LUT[9] = {28,20,21,22,27,23,25,24,30};

    struct {
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

        const TCUnits units;
    }tc_config;

    struct ReaderChannelConfig {
        std::string location;
        bool enabled = true;
        synnax::DataType data_type;
        uint32_t channel_key;
        double range = 10.0;
        std::string channel_type = "";
        int port;

        ReaderChannelConfig() = default;

        explicit ReaderChannelConfig(config::Parser &parser)
                : enabled(parser.optional<bool>("enabled", true)),
                  data_type(parser.required<std::string>("data_type")),
                  channel_key(parser.required<uint32_t>("channel")),
                  range(parser.optional<double>("range", 10.0)),
                  channel_type(parser.optional<std::string>("type", "")),
                  port(parser.optional<int>("port", 0)){
            this->location = this->channel_type + std::to_string(this->port);
            LOG(INFO) << parser.get_json().dump(4);
        }
    };

    struct ReaderConfig {
        std::string device_type;
        std::string device_key;
        std::vector<ReaderChannelConfig> channels;
        synnax::Rate sample_rate = synnax::Rate(1);
        synnax::Rate stream_rate = synnax::Rate(1);
        synnax::ChannelKey task_key;
        std::set<uint32_t> index_keys;
        std::string serial_number; // used to open devices
        std::string connection_type; // used to open devices
        std::map<std::string, uint32_t> channel_map; // move this into class instead of reader config
        std::vector<std::string> phys_channels;
        bool data_saving;

        ReaderConfig() = default;

        explicit ReaderConfig(config::Parser &parser)
                : device_type(parser.optional<std::string>("type", "")),
                  device_key(parser.required<std::string>("device_key")),
                  sample_rate(synnax::Rate(parser.optional<int>("sample_rate", 1))),
                  stream_rate(synnax::Rate(parser.optional<int>("stream_rate", 1))),
                  index_keys(parser.optional<std::set<uint32_t>>("index_keys", {})),
                  serial_number(parser.optional<std::string>("serial_number", "")),
                  connection_type(parser.optional<std::string>("connection_type", "")),
                  data_saving(parser.optional<bool>("data_saving", false)
          ) {

            // Parse the channels
            parser.iter("channels", [this](config::Parser &channel_parser) {
                channels.emplace_back(ReaderChannelConfig(channel_parser));
                this->channel_map[channels.back().location] = channels.back().channel_key;
                LOG(INFO) << "channel: " << channels.back().location;
                if(channels.back().enabled)
                    this->phys_channels.push_back(channels.back().location);

            });
        }
    };

///////////////////////////////////////////////////////////////////////////////////
//                                   Source                                      //
///////////////////////////////////////////////////////////////////////////////////
class Source : public pipeline::Source{
public:
    explicit Source (
            const std::shared_ptr<task::Context> &ctx,
            const synnax::Task task,
            const ReaderConfig &reader_config
        ) : ctx(ctx),
            task(task),
            reader_config(reader_config) {
        // TODO: default construct breaker?
        auto breaker_config = breaker::Config{
                .name = task.name,
                .base_interval = 1 * SECOND,
                .max_retries = 20,
                .scale = 1.2,
        };
        this->breaker = breaker::Breaker(breaker_config);

    }

    ~Source();


    std::vector<synnax::ChannelKey> get_channel_keys();

    void stopped_with_err(const freighter::Error &err);

    std::pair<Frame, freighter::Error> read(breaker::Breaker &breaker);

    void init();

    freighter::Error stop(const std::string &cmd_key);

    freighter::Error start(const std::string &cmd_key);

    void init_stream();

    void acquire_data();

    void write_to_series(
            synnax::Series &series,
            double &data,
            synnax::DataType data_type
        );

    void stop();

    int check_err(int err);

    void configure_tc_ain_ef(TCData tc_data);

private:
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
};
}