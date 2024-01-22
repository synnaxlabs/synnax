// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Emiliano Bonilla on 1/3/24.
//

#include "ni_reader.h"
#include "nlohmann/json.hpp"

using json = nlohmannnnniii::json;
using namespace ni;





//
//typedef freighter::Error (*DAQmxCreateChannel) (TaskHandle taskHandle, ChannelConfig config);
//
//freighter::Error create_ai_voltage_channel(TaskHandle taskHandle, json config) {
//    auto physical_channel = config["physical_channel"].get<std::string>();
//    auto max_val = config["max_val"].get<float>();
//    auto min_val = config["min_val"].get<float>();
//    DAQmxCreateAIVoltageChan(
//            taskHandle,
//            physical_channel.c_str(),
//            NULL,
//            DAQmx_Val_Cfg_Default,
//            min_val,
//            max_val,
//            DAQmx_Val_Volts,
//            NULL
//    );
//}
//
//static std::map<std::string, DAQmxCreateChannel> create_channel_map = {
//        {"ai_voltage", create_ai_voltage_channel}
//};
//
//freighter::Error Reader::configure(synnax::Module config) {
//    // Step 1 is parsing the configuration
//    json j = json::parse(config.config);
//
//    // Grab the sample rate key
//    auto sample_rate_val = j.find("sample_rate");
//    if (sample_rate_val == j.end()) return freighter::NIL;
//    if (!sample_rate_val->is_number_float()) return freighter::NIL;
//    auto sample_rate = synnax::Rate(sample_rate_val->get<std::float_t>());
//
//    // Grab the transfer rate key
//    auto transfer_rate_val = j.find("transfer_rate");
//    if (transfer_rate_val == j.end()) return freighter::NIL;
//    if (!transfer_rate_val->is_number_float()) return freighter::NIL;
//    auto transfer_rate = synnax::Rate(transfer_rate_val->get<std::float_t>());
//
//    // Grab the channels key
//    auto channels_val = j.find("channels");
//    if (channels_val == j.end()) return freighter::NIL;
//    if (!channels_val->is_array()) return freighter::NIL;
//    auto channels = channels_val->get<std::vector<json>>();
//    for (auto &channel: channels) {
//        if (!channel.is_object()) return freighter::NIL;
//        auto type_val = channel.find("type");
//        if (type_val == channel.end()) return freighter::NIL;
//        if (!type_val->is_string()) return freighter::NIL;
//        auto type = type_val->get<std::string>();
//    }
//}
//
//
//
//
//
//
//
//
//
//
//    int32 daqmx_err = DAQmxCreateTask(config.name.c_str(), &task);
//    uInt64 samples_per_chan = uInt64(config.sample_rate.value / config.transfer_rate.value);
//
//    for (auto &channel: config.channels) {
//        auto create_channel = create_channel_map[channel.type];
//        auto err = create_channel(task, channel);
//    }
//
//    daqmx_err = DAQmxCfgSampClkTiming(
//            task,
//            NULL,
//            config.sample_rate.value,
//            DAQmx_Val_Rising,
//            DAQmx_Val_ContSamps,
//            samples_per_chan
//    );
//
//    daqmx_err = DAQmxStartTask(task);
//}
//
//freighter::Error Reader::stop() {
//    int32 daqmx_err = DAQmxStopTask(task);
//    daqmx_err = DAQmxClearTask(task);
//}
//
//std::pair<synnax::Frame, freighter::Error> Reader::read() {
//    std::byte data[64];
//    int32 samples_read;
//    int32 daqmx_err = DAQmxReadAnalogF64(
//            task,
//            -1,
//            0,
//            DAQmx_Val_GroupByChannel,
//            reinterpret_cast<float64 *>(data),
//            64,
//            &samples_read,
//            NULL
//    );
//
//    return std::make_pair(std::move(frame), error);
//}