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


void ni::AnalogReadSource::getIndexKeys(){
    std::set<std::uint32_t> index_keys;
    //iterate through channels in reader config
    for (auto &channel : this->reader_config.channels){
        LOG(INFO) << "constructing index channel configs";
        auto [channel_info, err] = this->ctx->client->channels.retrieve(channel.channel_key);
        // TODO handle error with breaker
        if (err != freighter::NIL){
            // Log error
            LOG(ERROR) << "[NI Reader] failed to retrieve channel " << channel.channel_key;
            this->ok_state = false;
            return;
        } else{
            index_keys.insert(channel_info.index);
        }
    }

    // now iterate through the set and add all the index channels as configs
    for (auto it = index_keys.begin(); it != index_keys.end(); ++it){
        auto index_key = *it;
        LOG(INFO) << "constructing index channel configs";
        auto [channel_info, err] = this->ctx->client->channels.retrieve(index_key);
        if (err != freighter::NIL){
            LOG(ERROR) << "[NI Reader] failed to retrieve channel " << index_key;
            this->ok_state = false;
            return;
        } else{
            ni::ChannelConfig index_channel;
            index_channel.channel_key = channel_info.key;
            index_channel.channel_type = "index";
            index_channel.name = channel_info.name;
            this->reader_config.channels.push_back(index_channel);
            LOG(INFO) << "[NI Reader] index channel " << index_channel.channel_key << " and name: " << index_channel.name <<" added to task " << this->reader_config.task_name;
        }
    }
}

// TODO: Code dedup
ni::AnalogReadSource::AnalogReadSource(
    TaskHandle task_handle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task) : task_handle(task_handle), ctx(ctx){
    // Create parser
    auto config_parser = config::Parser(task.config);
    this->reader_config.task_name = task.name;
    this->reader_config.task_key = task.key;


    // Parse configuration and make sure it is valid
    this->parseConfig(config_parser);

    if (!config_parser.ok()){
        // Log error
        LOG(ERROR) << "[NI Reader] failed to parse configuration for " << this->reader_config.task_name;
        this->ctx->setState({.task = task.key,
                             .variant = "error",
                             .details = config_parser.error_json()});
        this->ok_state = false;
        return;
    }
    LOG(INFO) << "[NI Reader] successfully parsed configuration for " << this->reader_config.task_name;

    this->getIndexKeys(); // get index keys for the task
    

    LOG(INFO) << "[NI Reader] index keys retrieved " << this->reader_config.task_name;

    // Create breaker
    auto breaker_config = breaker::Config{
        .name = task.name,
        .base_interval = 1 * SECOND,
        .max_retries = 20,
        .scale = 1.2,
    };
    this->breaker = breaker::Breaker(breaker_config);

    // TODO: make sure you have all the channel info you could possible need
    // Now configure the actual NI hardware
    if (this->init()){
        LOG(ERROR) << "[NI Reader] Failed while configuring NI hardware for task " << this->reader_config.task_name;
        this->ok_state = false;
    }

    // this->start(); // errors are handled in start
}



void ni::AnalogReadSource::parseConfig(config::Parser &parser){
    // Get Acquisition Rate and Stream Rates
    this->reader_config.sample_rate = parser.required<uint64_t>("sample_rate");
    this->reader_config.stream_rate = parser.required<uint64_t>("stream_rate");
    this->reader_config.device_key = parser.required<std::string>("device");
    this->reader_config.timing_source = "none"; // parser.required<std::string>("timing_source"); TODO: uncomment this when ui provides timing source


    auto [dev, err] = this->ctx->client->hardware.retrieveDevice(this->reader_config.device_key);

    if (err != freighter::NIL) {
        LOG(ERROR) << "[NI Reader] failed to retrieve device " << this->reader_config.device_name;
        this->ok_state = false;
        return;
    }
    this->reader_config.device_name = dev.location;

    // now parse the channels
    assert(parser.ok());
    parser.iter("channels",
                [&](config::Parser &channel_builder){
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
                    this->parseCustomScale(channel_builder, config);
                    this->reader_config.channels.push_back(config);
                });
}



void ni::AnalogReadSource::parseCustomScale(config::Parser & parser, ni::ChannelConfig & config){
    json j = parser.get_json();
    if(j.contains("scale")){
        config.custom_scale = true;
        auto scale_parser = parser.child("scale");
        config.scale_type = scale_parser.required<std::string>("variant");

         // get the scaled and prescaled units
        auto prescaled_units = scale_parser.required<std::string>("prescaled_units");
        auto scaled_units = scale_parser.required<std::string>("scaled_units");

        // now handle the different variants
        // Scale custom_scale;
        if(config.scale_type == "LinScale"){

            auto slope = scale_parser.required<double>("slope");
            auto offset = scale_parser.required<double>("offset");
            config.scale->linear = {slope, offset, prescaled_units, scaled_units};

        } else if(config.scale_type == "MapScale"){
            
            auto prescaled_min = scale_parser.required<double>("prescaled_min");
            auto prescaled_max = scale_parser.required<double>("prescaled_max");
            auto scaled_min = scale_parser.required<double>("scaled_min");
            auto scaled_max = scale_parser.required<double>("scaled_max");
            config.scale->map = {prescaled_min, prescaled_max, scaled_min, scaled_max, prescaled_units, scaled_units};

        } else if(config.scale_type == "PolyScale"){
            // get forward coeffs (prescale -> scale)
            json j = scale_parser.get_json();
            if(!j.contains("forward_coeffs")){
                return;
            }
            std::vector<double> forward_coeffs_vec = j["forward_coeffs"]; 
            if(scale_parser.ok()){
                auto min_x = scale_parser.required<double>("min_x");
                auto max_x = scale_parser.required<double>("max_x");
                uint32_t num_forward_coeffs = scale_parser.required<uint32_t>("num_coeffs");
                auto poly_order = scale_parser.required<int32>("poly_order");
                int32_t num_points = scale_parser.required<uint32_t>("num_points");

                float64* forward_coeffs = new double[num_forward_coeffs];
                float64* reverse_coeffs = new double[num_forward_coeffs]; // TODO: reverse coeffs can be less than forward_coeffs depending on the order of the function (only equal if order is)      
                ParseFloats(forward_coeffs_vec, forward_coeffs);


                // get reverse coeffs (scale -> prescale)
                ni::NiDAQmxInterface::CalculateReversePolyCoeff(forward_coeffs, num_forward_coeffs, min_x, max_x, num_points, -1,  reverse_coeffs); // FIXME: reversePoly order should be user inputted?
                config.scale->polynomial = {forward_coeffs, reverse_coeffs, num_forward_coeffs, min_x, max_x, num_points, poly_order, prescaled_units, scaled_units};
            }
   
        } else if(config.scale_type == "TableScale"){
            json j = scale_parser.get_json();
            if(!j.contains("prescaled") || !j.contains("scaled")){
                return;
            }
            std::vector<double> prescaled_vec = j["prescaled"];
            std::vector<double> scaled_vec = j["scaled"];
            uint32_t num_points = scale_parser.required<uint32_t>("num_points");
            if(scale_parser.ok()){
                uint32_t num_points = prescaled_vec.size();
                float64* prescaled_arr = new double[num_points];
                float64* scaled_arr = new double[num_points];
                ParseFloats(prescaled_vec, prescaled_arr);
                ParseFloats(scaled_vec, scaled_arr);
                config.scale->table = {prescaled_arr, scaled_arr, num_points, prescaled_units, scaled_units};
            }
        } else{ //invalid scale type return error
            json err;
            err["errors"] = nlohmann::json::array();
            err["errors"].push_back({
                {"path", "scale->variant"},
                {"message", "Invalid scale type"}
            });
            LOG(ERROR) << "[NI Reader] failed to parse custom scale configuration for " << this->reader_config.task_name;
            this->ctx->setState({.task = this->reader_config.task_key,
                                .variant = "error",
                                .details = err});
            this->ok_state = false;
            return;
        }
        
        if(!scale_parser.ok()){ 
            LOG(ERROR) << "[NI Reader] failed to parse custom scale configuration for " << this->reader_config.task_name;
            this->ctx->setState({.task = this->reader_config.task_key,
                                .variant = "error",
                                .details = scale_parser.error_json()});
            this->ok_state = false;
            return;
        } 

    } else {
        config.custom_scale = false;
    }
}





int ni::AnalogReadSource::init(){
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

    // Configure buffer size and read resources
    if(this->reader_config.sample_rate < this->reader_config.stream_rate){
        this->err_info["error type"] = "Configuration Error";
        this->err_info["error details"] = "Stream rate is greater than sample rate";
        
        this->ctx->setState({.task = this->reader_config.task_key,
                             .variant = "error",
                             .details = err_info});
        LOG(ERROR) << "[NI Reader] stream rate is greater than sample rate " << this->reader_config.task_name;
        this->ok_state = false;
        return -1;
    }

    
    if (this->configureTiming()){
        LOG(ERROR) << "[NI Reader] Failed while configuring timing for NI hardware for task " << this->reader_config.task_name;
        this->ok_state = false;
    }
    

    LOG(INFO) << "[NI Reader] successfully configured NI hardware for task " << this->reader_config.task_name;
    return 0;
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
        LOG(INFO) << "[NI Reader] configuring special timing for task " << this->reader_config.task_name;
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
    // }r
}





freighter::Error ni::AnalogReadSource::start(){
    if(this->running){
        return freighter::NIL;
    }
    
    if (this->checkNIError(ni::NiDAQmxInterface::StartTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while starting reader for task " << this->reader_config.task_name;
        return freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }else{
        this->running = true;
        this->sample_thread = std::thread(&AnalogReadSource::acquireData, this);
        LOG(INFO) << "[NI Reader] successfully started reader for task " << this->reader_config.task_name;
    }
    return freighter::NIL;;
}


freighter::Error ni::AnalogReadSource::stop(){ 
    if(!this->running){
        return freighter::NIL;
    }
    this->running = false;
    this->sample_thread.join();

    if (this->checkNIError(ni::NiDAQmxInterface::StopTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while stopping reader for task " << this->reader_config.task_name;
        return freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    
    LOG(INFO) << "[NI Reader] successfully stopped  reader for task " << this->reader_config.task_name;
    data_queue.reset();
    return  freighter::NIL;
}


void ni::AnalogReadSource::deleteScales(){
    for(auto &channel : this->reader_config.channels){
        if(channel.custom_scale){
            if(channel.scale_type == "polyScale"){
                delete[] channel.scale->polynomial.forward_coeffs;
                delete[] channel.scale->polynomial.reverse_coeffs;
            } else if(channel.scale_type == "tableScale"){
                delete[] channel.scale->table.prescaled;
                delete[] channel.scale->table.scaled;
            }
        }
    }
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
                                                                data_packet.data,
                                                                this->bufferSize,
                                                                &data_packet.samplesReadPerChannel,
                                                                NULL))){
            LOG(ERROR) << "[NI Reader] failed while reading analog data for task " << this->reader_config.task_name;
            // return; // TODO: handle differently?
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
    std::optional<DataPacket> d = data_queue.dequeue();
    if(!d.has_value()) return std::make_pair(std::move(f), freighter::Error(driver::TYPE_TEMPORARY_HARDWARE_ERROR, "no data available to read"));
    double* data = d.value().data;
    
    // interpolate  timestamps between the initial and final timestamp to ensure non-overlapping timestamps between batched reads
    uint64_t incr = ( (d.value().tf- d.value().t0) / this->numSamplesPerChannel );
    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(this->numSamplesPerChannel);
    for (uint64_t i = 0; i < d.value().samplesReadPerChannel; ++i)
        time_index[i] = d.value().t0 + (std::uint64_t)(incr * i);
    

    // Construct and populate synnax frame
    uint64_t data_index = 0;
    for(int i = 0; i < numChannels; i++){
        if(this->reader_config.channels[i].channel_type == "index") continue;
        // copy data into vector
        std::vector<float> data_vec(d.value().samplesReadPerChannel);
        for (int j = 0; j < d.value().samplesReadPerChannel; j++)
            data_vec[j] = data[data_index * d.value().samplesReadPerChannel + j];
        f.add(this->reader_config.channels[i].channel_key, synnax::Series(data_vec, synnax::FLOAT32));
        data_index++;
    }

    for(int i = 0; i < numChannels; i++){
        if(this->reader_config.channels[i].channel_type != "index") continue;
        f.add(this->reader_config.channels[i].channel_key, synnax::Series(time_index, synnax::TIMESTAMP));
    }
    
    if(d.has_value()) delete[] d.value().data;

    return std::make_pair(std::move(f), freighter::NIL);
}


int ni::AnalogReadSource::createChannel(ni::ChannelConfig &channel){
    if(!channel.custom_scale){
        return this->checkNIError(ni::NiDAQmxInterface::CreateAIVoltageChan(this->task_handle, channel.name.c_str(), "", channel.terminal_config, channel.min_val, channel.max_val, DAQmx_Val_Volts, NULL));
    } else{
        // name scale
         channel.scale_name = channel.name + "_scale";
        // create scale
        if(channel.scale_type == "LinScale"){
        
            this->checkNIError( 
                ni::NiDAQmxInterface::CreateLinScale(
                    channel.scale_name.c_str(), 
                    channel.scale->linear.slope, 
                    channel.scale->linear.offset, 
                    ni::UNITS_MAP.at(channel.scale->linear.prescaled_units), 
                    channel.scale->linear.scaled_units.c_str()
            ));

        } else if(channel.scale_type == "MapScale"){

            this->checkNIError(ni::NiDAQmxInterface::CreateMapScale(
                channel.scale_name.c_str(), 
                channel.scale->map.prescaled_min, 
                channel.scale->map.prescaled_max, 
                channel.scale->map.scaled_min, 
                channel.scale->map.scaled_max, 
                ni::UNITS_MAP.at(channel.scale->map.prescaled_units), 
                channel.scale->map.scaled_units.c_str()
            ));

        } else if(channel.scale_type == "PolyScale"){

            // create forward and reverse coeffs inputs
            float64 forward_coeffs_in[1000];
            float64 reverse_coeffs_in[1000];
            for(int i = 0; i < channel.scale->polynomial.num_coeffs; i++){
                forward_coeffs_in[i] = channel.scale->polynomial.forward_coeffs[i];
                reverse_coeffs_in[i] = channel.scale->polynomial.reverse_coeffs[i];
            }

            this->checkNIError(ni::NiDAQmxInterface::CreatePolynomialScale(
                channel.scale_name.c_str(), 
                forward_coeffs_in, 
                channel.scale->polynomial.num_coeffs, 
                reverse_coeffs_in, 
                channel.scale->polynomial.num_coeffs,
                ni::UNITS_MAP.at(channel.scale->polynomial.prescaled_units), 
                channel.scale->polynomial.scaled_units.c_str()
            
            ));

        } else if(channel.scale_type == "TableScale"){
            // create prescaled and scaled inputs
            float64 prescaled[1000];
            float64 scaled[1000];
            for(int i = 0; i < channel.scale->table.num_points; i++){
                prescaled[i] = channel.scale->table.prescaled[i];
                scaled[i] = channel.scale->table.scaled[i];
            }
            this->checkNIError(ni::NiDAQmxInterface::CreateTableScale(
                channel.scale_name.c_str(), 
                prescaled, 
                channel.scale->table.num_points, 
                scaled,
                channel.scale->table.num_points, 
                ni::UNITS_MAP.at(channel.scale->table.prescaled_units), 
                channel.scale->table.scaled_units.c_str()
            ));
        }
        // create channel
        return this->checkNIError(ni::NiDAQmxInterface::CreateAIVoltageChan(this->task_handle, channel.name.c_str(), "", channel.terminal_config, channel.min_val, channel.max_val, DAQmx_Val_Volts, channel.scale_name.c_str()));
    }
    return -1;
}

bool ni::AnalogReadSource::ok(){ 
    return this->ok_state;
}


ni::AnalogReadSource::~AnalogReadSource(){
    LOG(INFO) << "[NI Reader] clearing reader for task " << this->reader_config.task_name;
    if (this->checkNIError(ni::NiDAQmxInterface::ClearTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while clearing reader for task " << this->reader_config.task_name;
    }
    this->deleteScales();
}

int ni::AnalogReadSource::checkNIError(int32 error){
    if (error < 0){
        char errBuff[2048] = {'\0'};

        ni::NiDAQmxInterface::GetExtendedErrorInfo(errBuff, 2048);

        this->err_info["error type"] = "Vendor Error";
        this->err_info["error details"] = errBuff;
        
        this->ctx->setState({.task = this->reader_config.task_key,
                             .variant = "error",
                             .details = err_info});

        LOG(ERROR) << "[NI Reader] Vendor Error: " << this->err_info["error details"];
        this->ok_state = false;

        return -1;
    }
    return 0;
}


std::vector<synnax::ChannelKey> ni::AnalogReadSource::getChannelKeys(){
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel : this->reader_config.channels){
        keys.push_back(channel.channel_key);
    }
    return keys;
}



