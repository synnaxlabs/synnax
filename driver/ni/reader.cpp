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



///////////////////////////////////////////////////////////////////////////////////
//                             Helper Functions                                  //
///////////////////////////////////////////////////////////////////////////////////
void ni::DaqAnalogReader::getIndexKeys(){
    std::set<std::uint32_t> index_keys;
    //iterate through channels in reader config
    for (auto &channel : this->reader_config.channels){
        auto [channel_info, err] = this->ctx->client->channels.retrieve(channel.channel_key);
        // TODO handle error with breaker
        if (err != freighter::NIL){
            // Log error
            LOG(ERROR) << "[NI Reader] failed to retrieve channel " << channel.channel_key;
            this->ok_state = false;
            return;
        } else{
            // add key to set
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


void ni::DaqDigitalReader::getIndexKeys(){
    std::set<std::uint32_t> index_keys;
    //iterate through channels in reader config
    for (auto &channel : this->reader_config.channels){
        auto [channel_info, err] = this->ctx->client->channels.retrieve(channel.channel_key);
        // TODO handle error with breaker
        if (err != freighter::NIL){
            // Log error
            LOG(ERROR) << "[NI Reader] failed to retrieve channel " << channel.channel_key;
            this->ok_state = false;
            return;
        } else{
            // add key to set
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


// TODO: fix this to avoid code duplication:


uint32_t parseFloats(std::vector<float64> vec, double* arr){
    for(int i = 0; i < vec.size(); i++){
        arr[i] = vec[i];
    }
    return vec.size();
}




///////////////////////////////////////////////////////////////////////////////////
//                                   daqAnalogReader                             //
///////////////////////////////////////////////////////////////////////////////////

// TODO: Code dedup
ni::DaqAnalogReader::DaqAnalogReader(
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

    this->start(); // errors are handled in start
}



void ni::DaqAnalogReader::parseConfig(config::Parser &parser){
    // Get Acquisition Rate and Stream Rates
    this->reader_config.acq_rate = parser.required<uint64_t>("sample_rate");
    this->reader_config.stream_rate = parser.required<uint64_t>("stream_rate");
    this->reader_config.device_key = parser.required<std::string>("device");


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



void ni::DaqAnalogReader::parseCustomScale(config::Parser & parser, ni::ChannelConfig & config){
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
                auto num_points = scale_parser.required<int32>("num_points");
                auto poly_order = scale_parser.required<int32>("poly_order");

                float64* forward_coeffs = new double[num_points];
                float64* reverse_coeffs = new double[num_points]; // TODO: reverse coeffs can be less than forward_coeffs depending on the order of the function (only equal if order is)      
                uint32_t num_coeffs = parseFloats(forward_coeffs_vec, forward_coeffs);


                // get reverse coeffs (scale -> prescale)
                ni::NiDAQmxInterface::CalculateReversePolyCoeff(forward_coeffs, num_coeffs, min_x, max_x, num_points, -1,  reverse_coeffs); // FIXME: reversePoly order should be user inputted?
                config.scale->polynomial = {forward_coeffs, reverse_coeffs, num_coeffs, min_x, max_x, num_points, poly_order, prescaled_units, scaled_units};
            }
   
        } else if(config.scale_type == "TableScale"){
            std::vector<double> prescaled_vec = {}; //  scale_parser.required<std::vector<double>>("prescaled"); //FIXME
            std::vector<double> scaled_vec =  {}; //scale_parser.required<std::vector<double>>("scaled"); //FIXME
            if(scale_parser.ok()){
                uint32_t num_points = prescaled_vec.size();
                float64* prescaled_arr;
                float64* scaled_arr;
                uint32_t num_prescaled = parseFloats(prescaled_vec, prescaled_arr);
                uint32_t num_scaled = parseFloats(scaled_vec, scaled_arr);
                if(num_prescaled != num_scaled){
                    // error in parsing scale need to handle error here !!!
                    return;
                }
                config.scale->table = {prescaled_arr, scaled_arr, num_points, prescaled_units, scaled_units};
            }
        } else{ //invalid scale type return error
            // Log error
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
            // print error json
            std::cout << scale_parser.error_json() << std::endl; // TODO: remove
            return;
        }
        
        if(!scale_parser.ok()){ // error in parsing scale need to handle error here !!!
            // Log error
            LOG(ERROR) << "[NI Reader] failed to parse custom scale configuration for " << this->reader_config.task_name;
            this->ctx->setState({.task = this->reader_config.task_key,
                                .variant = "error",
                                .details = scale_parser.error_json()});
            this->ok_state = false;
            // print error json
            std::cout << scale_parser.error_json() << std::endl; // TODO: remove
            return;
        } 

    } else {
        config.custom_scale = false;
    }
}





int ni::DaqAnalogReader::init(){
    int err = 0;
    auto channels = this->reader_config.channels;

    for (auto &channel : channels){
        if (channel.channel_type != "index" ){
            err = createChannel(channel);
        } 
        this->numChannels++; 
        if (err < 0){
            LOG(ERROR) << "[NI Reader] failed while configuring channel " << channel.name;
            this->ok_state = false;
            return -1;
        }
    }

    //TODO: assert that at least 1 channel has been configured here:

    // Configure timing
    // TODO: make sure there isnt different cases to handle between analog and digital
    if (this->checkNIError(ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
                                                                  "",
                                                                  this->reader_config.acq_rate,
                                                                  DAQmx_Val_Rising,
                                                                  DAQmx_Val_ContSamps,
                                                                  this->reader_config.acq_rate))){
        LOG(ERROR) << "[NI Reader] failed while configuring timing for task " << this->reader_config.task_name;
        this->ok_state = false;
        return -1;
    }

    // Configure buffer size and read resources
    if(this->reader_config.acq_rate < this->reader_config.stream_rate){
        this->err_info["error type"] = "Configuration Error";
        this->err_info["error details"] = "Stream rate is greater than sample rate";
        
        this->ctx->setState({.task = this->reader_config.task_key,
                             .variant = "error",
                             .details = err_info});
        LOG(ERROR) << "[NI Reader] stream rate is greater than sample rate " << this->reader_config.task_name;
        this->ok_state = false;
        return -1;
    }

    
    this->numSamplesPerChannel = std::floor(this->reader_config.acq_rate / this->reader_config.stream_rate);
    this->bufferSize = this->numChannels * this->numSamplesPerChannel;
    
    this->data = new double[bufferSize];

    LOG(INFO) << "[NI Reader] successfully configured NI hardware for task " << this->reader_config.task_name;
    return 0;
}





freighter::Error ni::DaqAnalogReader::start(){
    if(this->running){
        LOG(INFO) << "[NI Reader] attempt to start an already running NI task for task " << this->reader_config.task_name;
        return freighter::NIL; // TODO: change return value?
    }

    freighter::Error err = freighter::NIL;
    this->running = true;
    if (this->checkNIError(ni::NiDAQmxInterface::StartTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while starting reader for task " << this->reader_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else{
        LOG(INFO) << "[NI Reader] successfully started reader for task " << this->reader_config.task_name;
    }
    return err;
}


freighter::Error ni::DaqAnalogReader::stop(){ 
    if(!this->running){
        LOG(INFO) << "[NI Reader] attempt to stop an already stopped NI task for task " << this->reader_config.task_name;
        return freighter::NIL; // TODO: change return value?
    }

    freighter::Error err = freighter::NIL;
    this->running = false;
    if (this->checkNIError(ni::NiDAQmxInterface::StopTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while stopping reader for task " << this->reader_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else{
        if (this->checkNIError(ni::NiDAQmxInterface::ClearTask(this->task_handle))){
            LOG(ERROR) << "[NI Reader] failed while clearing reader for task " << this->reader_config.task_name;
            err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
        }
    }

    if (err == freighter::NIL){
        LOG(INFO) << "[NI Reader] successfully stopped and cleared reader for task " << this->reader_config.task_name;
    }

    return err;
}


void ni::DaqAnalogReader::deleteScales(){
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



std::pair<synnax::Frame, freighter::Error> ni::DaqAnalogReader::read(){
    int32 samplesRead = 0;
    float64 flush[100000]; // to flush buffer before performing a read
    int32 flushRead = 0;
    synnax::Frame f = synnax::Frame(numChannels);

    // initial read to flush buffer
    if (this->checkNIError(ni::NiDAQmxInterface::ReadAnalogF64(
                                            this->task_handle,
                                            -1, // reads all available samples in buffer
                                            10.0,
                                            DAQmx_Val_GroupByChannel,
                                            flush,
                                            100000,
                                            &flushRead,
                                            NULL))){
        LOG(ERROR) << "[NI Reader] failed while flushing buffer for task " << this->reader_config.task_name;
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "error reading analog data"));
    }

    // actual read of analog lines
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    if (this->checkNIError(ni::NiDAQmxInterface::ReadAnalogF64(
                                                            this->task_handle,
                                                            this->numSamplesPerChannel,
                                                            -1,
                                                            DAQmx_Val_GroupByChannel,
                                                            this->data,
                                                            this->bufferSize,
                                                            &samplesRead,
                                                            NULL))){
        LOG(ERROR) << "[NI Reader] failed while reading analog data for task " << this->reader_config.task_name;
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "Error reading analog data"));
    }
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

    // we interpolate the timestamps between the initial and final timestamp to ensure non-overlapping timestamps between read iterations
    uint64_t diff = final_timestamp - initial_timestamp;
    uint64_t incr = diff / this->numSamplesPerChannel;

    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(this->numSamplesPerChannel);
    for (uint64_t i = 0; i < samplesRead; ++i){
        time_index[i] = initial_timestamp + (std::uint64_t)(incr * i);
    }

    // Construct and populate synnax frame
    std::vector<float> data_vec(samplesRead);
    uint64_t data_index = 0; // TODO: put a comment explaining the function of data_index
    for (int i = 0; i < numChannels; i++){
        if (this->reader_config.channels[i].channel_type == "index"){
            // LOG(INFO) << "Index channel found: " << this->reader_config.channels[i].channel_key << " name: " << this->reader_config.channels[i].name;
            f.add(this->reader_config.channels[i].channel_key, synnax::Series(time_index, synnax::TIMESTAMP));
        }
        else{
            for (int j = 0; j < samplesRead; j++){
                data_vec[j] = data[data_index * samplesRead + j];
            }
            f.add(this->reader_config.channels[i].channel_key, synnax::Series(data_vec));
            data_index++;
        }
    }
    
    // return synnax frame
    return std::make_pair(std::move(f), freighter::NIL);
}


int ni::DaqAnalogReader::createChannel(ni::ChannelConfig &channel){
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

bool ni::DaqAnalogReader::ok(){ 
    return this->ok_state;
}


ni::DaqAnalogReader::~DaqAnalogReader(){
    this->stop();
    this->deleteScales();
    delete[] this->data;
}

int ni::DaqAnalogReader::checkNIError(int32 error){
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

///////////////////////////////////////////////////////////////////////////////////
//                             DaqDigitalReader                                  //
///////////////////////////////////////////////////////////////////////////////////

// TODO: Code dedup
ni::DaqDigitalReader::DaqDigitalReader(
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

    //TODO: 
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

    this->start(); // errors are handled in start
}


void ni::DaqDigitalReader::parseConfig(config::Parser &parser){
    // Get Acquisition Rate and Stream Rates
    this->reader_config.acq_rate = parser.required<uint64_t>("sample_rate");
    this->reader_config.stream_rate = parser.required<uint64_t>("stream_rate");
    this->reader_config.device_key = parser.required<std::string>("device");
    this->reader_config.timing_source = "none"; // parser.required<std::string>("timing_source"); TODO: uncomment this when ui provides timing source

    // TODO: add a parser ok check here


    auto [dev, err] = this->ctx->client->hardware.retrieveDevice(this->reader_config.device_key);

    if (err != freighter::NIL) {
        LOG(ERROR) << "[NI Reader] failed to retrieve device " << this->reader_config.device_name;
        this->ok_state = false;
        return;
    }
    this->reader_config.device_name = dev.location;
    assert(parser.ok());

    // now parse the channels
    parser.iter("channels",
                [&](config::Parser &channel_builder){
                    ni::ChannelConfig config;

                    // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
                    config.name = (this->reader_config.device_name + "/port" + std::to_string(channel_builder.required<std::uint64_t>("port")) + "/line" + std::to_string(channel_builder.required<std::uint64_t>("line")));

                    config.channel_key = channel_builder.required<uint32_t>("channel");

                    // TODO: there could be more than 2 state logic
                    config.min_val = 0;
                    config.max_val = 1;

                    this->reader_config.channels.push_back(config);
                });
    assert(parser.ok());
}




int ni::DaqDigitalReader::init(){
    int err = 0;
    auto channels = this->reader_config.channels;

    for (auto &channel : channels){
        if (channel.channel_type != "index" ){
            err = this->checkNIError(ni::NiDAQmxInterface::CreateDIChan(task_handle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine));
            LOG(INFO) << "Channel name: " << channel.name;
        } 
        LOG(INFO) << "Index channel added to task: " << channel.name;
        this->numChannels++; 
        if (err < 0){
            LOG(ERROR) << "[NI Reader] failed while configuring channel " << channel.name;
            this->ok_state = false;
            return -1;
        }
    }

    // Configure buffer size and read resources
     if(this->reader_config.acq_rate < this->reader_config.stream_rate){
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
    
    LOG(INFO) << "[NI Reader] successfully configured timing NI hardware for task " << this->reader_config.task_name;

    LOG(INFO) << "[NI Reader] successfully configured NI hardware for task " << this->reader_config.task_name;
    return 0;
}

int ni::DaqDigitalReader::configureTiming(){

        //TODO: assert that at least 1 channel has been configured here:

    if(this->reader_config.timing_source == "none"){ // if timing is not enabled, implement timing in software
        this->reader_config.period = (uint32_t)((1.0 / this->reader_config.acq_rate) * 1000000); // convert to microseconds

        this->numSamplesPerChannel = 1;//std::floor(this->reader_config.acq_rate / this->reader_config.stream_rate);
        this->bufferSize = this->numChannels * this->numSamplesPerChannel;
    
        this->data = new double[bufferSize];

    } else{
        // Configure timing
        if (this->checkNIError(ni::NiDAQmxInterface::CfgSampClkTiming(this->task_handle,
                                                                    this->reader_config.timing_source.c_str(),
                                                                    this->reader_config.acq_rate,
                                                                    DAQmx_Val_Rising,
                                                                    DAQmx_Val_ContSamps,
                                                                    this->reader_config.acq_rate))){
            LOG(ERROR) << "[NI Reader] failed while configuring timing for task " << this->reader_config.task_name;
            this->ok_state = false;
            return -1;
        }

        this->numSamplesPerChannel = std::floor(this->reader_config.acq_rate / this->reader_config.stream_rate);
        this->bufferSize = this->numChannels * this->numSamplesPerChannel;
        // this->data = new double[bufferSize];
    }
    return 0;
}

freighter::Error ni::DaqDigitalReader::start(){
    if(this->running){
        LOG(INFO) << "[NI Reader] attempt to start an already running NI task for task " << this->reader_config.task_name;
        return freighter::NIL; // TODO: change return value?
    }

    freighter::Error err = freighter::NIL;
    this->running = true;
    if (this->checkNIError(ni::NiDAQmxInterface::StartTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while starting reader for task " << this->reader_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else{
        LOG(INFO) << "[NI Reader] successfully started reader for task " << this->reader_config.task_name;
    }
    return err;
}


freighter::Error ni::DaqDigitalReader::stop(){ 
    if(!this->running){
        LOG(INFO) << "[NI Reader] attempt to stop an already stopped NI task for task " << this->reader_config.task_name;
        return freighter::NIL; // TODO: change return value?
    }

    freighter::Error err = freighter::NIL;
    this->running = false;
    if (this->checkNIError(ni::NiDAQmxInterface::StopTask(this->task_handle))){
        LOG(ERROR) << "[NI Reader] failed while stopping reader for task " << this->reader_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else{
        if (this->checkNIError(ni::NiDAQmxInterface::ClearTask(this->task_handle))){
            LOG(ERROR) << "[NI Reader] failed while clearing reader for task " << this->reader_config.task_name;
            err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
        }
    }


    if (err == freighter::NIL){
        LOG(INFO) << "[NI Reader] successfully stopped and cleared reader for task " << this->reader_config.task_name;
    }

    return err;
}


std::pair<synnax::Frame, freighter::Error> ni::DaqDigitalReader::read(){
    int32 samplesRead;
    char errBuff[2048] = {'\0'};
    uInt8 flushBuffer[10000]; // to flush buffer before performing a read
    uInt8 dataBuffer[10000]; // TODO fix this
    int32 flushRead;
    synnax::Frame f = synnax::Frame(numChannels);
    int32 numBytesPerSamp; // TODO do i need this?
    int err = 0;

    // initial read to flush buffer
    if (this->checkNIError(ni::NiDAQmxInterface::ReadDigitalLines(this->task_handle, // TODO: come back to and make sure this call to flush will be fine at any scale (elham)
                                                                  -1,               // reads all available samples in the buffer
                                                                  -1,
                                                                  DAQmx_Val_GroupByChannel,
                                                                  flushBuffer,
                                                                  1000,
                                                                  &samplesRead,
                                                                  &numBytesPerSamp,
                                                                  NULL))){
        LOG(ERROR) << "[NI Reader] failed while flushing buffer for task " << this->reader_config.task_name;
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "error reading digital data"));
    }

    // sleep if period is not 0
    if(this->reader_config.period != 0){
        std::this_thread::sleep_for(std::chrono::microseconds(this->reader_config.period));
    }

    // actual read to of digital lines
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    if (this->checkNIError(ni::NiDAQmxInterface::ReadDigitalLines(this->task_handle,           // task handle
                                                                  this->numSamplesPerChannel, // numSampsPerChan
                                                                  -1,                         // timeout
                                                                  DAQmx_Val_GroupByChannel,   // dataLayout
                                                                  dataBuffer,                 // readArray
                                                                  10000,                      // arraySizeInSamps
                                                                  &samplesRead,               // sampsPerChanRead
                                                                  NULL,                       // numBytesPerSamp
                                                                  NULL))){
        LOG(ERROR) << "[NI Reader] failed while reading digital data for task " << this->reader_config.task_name;
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "error reading digital data"));
    }
    std::uint64_t final_timestamp = (synnax::TimeStamp::now()).value;

    // we interpolate the timestamps between the initial and final timestamp to ensure non-overlapping timestamps between read iterations
    uint64_t diff = final_timestamp - initial_timestamp;
    uint64_t incr = diff / this->numSamplesPerChannel;

    // Construct and populate index channel
    std::vector<std::uint64_t> time_index(numSamplesPerChannel);
    for (int i = 0; i < samplesRead; ++i){
        time_index[i] = initial_timestamp + (std::uint64_t)(incr * i);
    }
    
    // Construct and populate synnax frame
    std::vector<uint8_t> data_vec(samplesRead);
    uint64_t data_index = 0; // TODO: put a comment explaining the function of data_index
    for (int i = 0; i < numChannels; i++){
        if (this->reader_config.channels[i].channel_type == "index"){
            f.add(this->reader_config.channels[i].channel_key, synnax::Series(time_index, synnax::TIMESTAMP));
        }
        else{
            for (int j = 0; j < samplesRead; j++){
                data_vec[j] = dataBuffer[data_index * samplesRead + j];
            }
            f.add(this->reader_config.channels[i].channel_key, synnax::Series(data_vec));
            data_index++;
        }
    }

    // return synnax frame
    return std::make_pair(std::move(f), freighter::NIL);
}

bool ni::DaqDigitalReader::ok(){ 
    return this->ok_state;
}

ni::DaqDigitalReader::~DaqDigitalReader(){
    this->stop();
    delete[] this->data;
}


int ni::DaqDigitalReader::checkNIError(int32 error){
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


std::vector<synnax::ChannelKey> ni::DaqAnalogReader::getChannelKeys(){
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel : this->reader_config.channels){
        keys.push_back(channel.channel_key);
    }
    return keys;
}

std::vector<synnax::ChannelKey> ni::DaqDigitalReader::getChannelKeys(){
    std::vector<synnax::ChannelKey> keys;
    for (auto &channel : this->reader_config.channels){
        keys.push_back(channel.channel_key);
    }
    return keys;
}