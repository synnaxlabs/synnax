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

#include "driver/ni/ni.h"
#include "nlohmann/json.hpp"
#include "client/cpp/telem/telem.h"
#include <utility>
#include <chrono>
#include <stdio.h>
#include <cassert>
#include "glog/logging.h"

using json = nlohmann::json;


void  ParseFloats(std::vector<float64> vec, double* arr){
    for(int i = 0; i < vec.size(); i++){
        arr[i] = vec[i];
    }
}

void ni::AnalogReadSource::parseChannels(config::Parser &parser){
    LOG(INFO) << "[NI Reader] Parsing Channels for task " << this->reader_config.task_name;
    // now parse the channels
    parser.iter("channels",
                [&](config::Parser &channel_builder){

                    LOG(INFO) << channel_builder.get_json().dump(4);

                    ni::ChannelConfig config;
                    // analog channel names are formatted: <device_name>/ai<port>
                    config.name = (this->reader_config.device_name + "/ai" + std::to_string(channel_builder.required<std::uint64_t>("port")));

                    config.channel_key = channel_builder.required<uint32_t>("channel");

                    config.min_val = channel_builder.required<float_t>("min_val");
                    config.max_val = channel_builder.required<std::float_t>("max_val");

                    auto terminal_config = channel_builder.required<std::string>("terminal_config");
                    config.terminal_config =     (terminal_config == "PseudoDiff") ? DAQmx_Val_PseudoDiff 
                                            :    (terminal_config == "Diff") ? DAQmx_Val_Diff
                                            :    (terminal_config == "NRSE") ? DAQmx_Val_NRSE
                                            :    (terminal_config == "RSE") ? DAQmx_Val_RSE
                                            :    DAQmx_Val_Cfg_Default;
                
                    // check for custom scale
                    std::string scale_name = std::to_string(config.channel_key) + "_scale";
                    auto scale_parser = channel_builder.child("custom_scale");
                    config.scale_config = ScaleConfig(scale_parser, scale_name);
                    if(!scale_parser.ok()){
                        LOG(ERROR) << "[NI Reader] Failed to parse custom scale for channel " << config.name;
                        this->ok_state = false;
                        return;
                    }
                    
                    this->reader_config.channels.push_back(config);
                });
}



int ni::AnalogReadSource::configureTiming(){
    if(this->reader_config.timing_source == "none"){
        LOG(INFO) << "[NI Reader] configuring timing for task " << this->reader_config.task_name;
        if (this->checkNIError(ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
                                                                  "",
                                                                  this->reader_config.sample_rate,
                                                                  DAQmx_Val_Rising,
                                                                  DAQmx_Val_ContSamps,
                                                                  this->reader_config.sample_rate))){
        LOG(ERROR) << "[NI Reader] failed while configuring timing for task " << this->reader_config.task_name;
        this->ok_state = false;
        return -1;
    }
    } else{
        if (this->checkNIError(ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
                                                                    this->reader_config.timing_source.c_str(),
                                                                    this->reader_config.sample_rate,
                                                                    DAQmx_Val_Rising,
                                                                    DAQmx_Val_ContSamps,
                                                                    this->reader_config.sample_rate))){
            LOG(ERROR) << "[NI Reader] failed while configuring timing for task " << this->reader_config.task_name;
            this->ok_state = false;
            return -1;
        }
    }
    this->numSamplesPerChannel = std::floor(this->reader_config.sample_rate / this->reader_config.stream_rate);
    this->bufferSize = this->numAIChannels * this->numSamplesPerChannel;
    return 0;
}

void ni::AnalogReadSource::acquireData(){
    while(this->running){
        DataPacket data_packet;
        data_packet.data = new double[this->bufferSize];
        data_packet.t0 = (uint64_t) ((synnax::TimeStamp::now()).value);
        if (this->checkNIError(ni::NiDAQmxInterface::ReadAnalogF64(
                                                            this->task_handle,
                                                            this->numSamplesPerChannel,
                                                            -1,
                                                            DAQmx_Val_GroupByChannel,
                                                            static_cast<double*>(data_packet.data),
                                                            this->bufferSize,
                                                            &data_packet.samplesReadPerChannel,
                                                            NULL))){
            this->logError("failed while reading analog data for task " + this->reader_config.task_name);
        }
        data_packet.tf = (uint64_t)((synnax::TimeStamp::now()).value);
        data_queue.enqueue(data_packet);
    }
}

std::pair<synnax::Frame, freighter::Error> ni::AnalogReadSource::read(){
    synnax::Frame f = synnax::Frame(numChannels);
    // sleep per stream rate
    std::this_thread::sleep_for(std::chrono::nanoseconds((uint64_t)((1.0 / this->reader_config.stream_rate )* 1000000000)));

    // take data off of queue
    DataPacket d = data_queue.dequeue();
    double* data = static_cast<double*>(d.data);

    // interpolate  timestamps between the initial and final timestamp to ensure non-overlapping timestamps between batched reads
    uint64_t incr = ( (d.tf- d.t0) / this->numSamplesPerChannel );
    
    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(this->numSamplesPerChannel);
    for (uint64_t i = 0; i < d.samplesReadPerChannel; ++i)
        time_index[i] = d.t0 + (std::uint64_t)(incr * i);
    

    // Construct and populate synnax frame
    uint64_t data_index = 0;
    for(int i = 0; i < numChannels; i++){
        if(this->reader_config.channels[i].channel_type == "index") {
            f.add(this->reader_config.channels[i].channel_key, synnax::Series(time_index, synnax::TIMESTAMP));
            continue;
        }
        // copy data into vector
        std::vector<float> data_vec(d.samplesReadPerChannel);
        for (int j = 0; j < d.samplesReadPerChannel; j++)
            data_vec[j] = data[data_index * d.samplesReadPerChannel + j];
        f.add(this->reader_config.channels[i].channel_key, synnax::Series(data_vec, synnax::FLOAT32));
        data_index++;
    }

    //delete data array
    delete[] data;

    return std::make_pair(std::move(f), freighter::NIL);
}


int ni::AnalogReadSource::createChannel(ni::ChannelConfig &channel){
    if(channel.scale_config.type == "none"){
        return  this->checkNIError(ni::NiDAQmxInterface::CreateAIVoltageChan( 
                    this->task_handle, 
                    channel.name.c_str(), 
                    "", 
                    channel.terminal_config, 
                    channel.min_val, 
                    channel.max_val, 
                    DAQmx_Val_Volts, 
                    NULL
                ));
    } else{
        this->checkNIError(channel.scale_config.createNIScale());
        return this->checkNIError(ni::NiDAQmxInterface::CreateAIVoltageChan(    
                this->task_handle, channel.name.c_str(), 
                "", 
                channel.terminal_config, 
                channel.min_val, 
                channel.max_val, 
                DAQmx_Val_FromCustomScale,  
                channel.scale_config.name.c_str()
            ));
    }
}


int ni::AnalogReadSource::createChannels(){
    int err = 0;
    auto channels = this->reader_config.channels;
    for (auto &channel : channels){
        if (channel.channel_type != "index" ){
            err = createChannel(channel);
            this->numAIChannels++;
        } 
        this->numChannels++; 
        if (err < 0){
            LOG(ERROR) << "[NI Reader] failed while configuring channel " << channel.name;
            this->ok_state = false;
            return -1;
        }
    }
    return 0;
}





