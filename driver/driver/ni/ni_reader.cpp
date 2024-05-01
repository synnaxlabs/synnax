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

#include "driver/driver/ni/ni_reader.h"
#include "nlohmann/json.hpp"
#include "client/cpp/synnax/telem/telem.h"
#include <utility>
#include <chrono>
#include <stdio.h>
#include <cassert>
#include <driver/driver/config/config.h>
#include "glog/logging.h"

using json = nlohmann::json;
// using namespace ni; // TODO; remove

///////////////////////////////////////////////////////////////////////////////////
//                                    niDaqReader                                //
///////////////////////////////////////////////////////////////////////////////////

ni::niDaqReader::niDaqReader(
    TaskHandle taskHandle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task): taskHandle(taskHandle), ctx(ctx)
{
    // Create parser
    auto config_parser = config::Parser(task.config);
    this->reader_config.task_name = task.name;
    this->reader_config.task_key = task.key;

    this->reader_config.reader_type = config_parser.required<std::string>("reader_type");
    this->reader_config.isDigital = (this->reader_config.reader_type == "digitalReader");

    // Parse configuration and make sure it is valid
    if (this->reader_config.isDigital){ this->parse_digital_reader_config(config_parser);}
    else{   this->parse_analog_reader_config(config_parser);}
    if (!config_parser.ok()){
        // Log error
        LOG(ERROR) << "[NI Reader] failed to parse configuration for " << this->reader_config.task_name;
        this->ctx->setState({.task = task.key,
                             .variant = "error",
                             .details = config_parser.error_json()});
        this->ok_state = false;

        //print error json
        std::cout << config_parser.error_json() << std::endl;

        return;
    }
    LOG(INFO) << "[NI Reader] successfully parsed configuration for " << this->reader_config.task_name;

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
    if(this->init()){
        LOG(ERROR) << "[NI Reader] Failed while configuring NI hardware for task " << this->reader_config.task_name;
        this->ok_state = false;
    }
}

void ni::niDaqReader::parse_analog_reader_config(config::Parser & parser)
{
    // Get Acquisition Rate and Stream Rates
    this->reader_config.acq_rate = parser.required<uint64_t>("acq_rate");
    this->reader_config.stream_rate = parser.required<uint64_t>("stream_rate");

    // device name
    this->reader_config.device_name = parser.required<std::string>("device_name");
    //std::cout << "Device name: " << this->reader_config.device_name << std::endl;
    //std::cout << "lul Device name: " << this->reader_config.device_name << std::endl;
    // now parse the channels
    parser.iter("channels",
                [&](config::Parser &channel_builder)
                {
                    ChannelConfig config;
                    config.channel_type = channel_builder.required<std::string>("channel_type");

                    // analog channel names are formatted: <device_name>/ai<port>
                    config.name =   (config.channel_type == "index")  ? (channel_builder.required<std::string>("name")) 
                            : (this->reader_config.device_name +"/ai" + std::to_string(channel_builder.required<std::uint64_t>("port")));

                    config.channel_key = channel_builder.required<uint32_t>("channel_key");

                    if(config.channel_type != "index"){
                        config.min_val = channel_builder.required<float_t>("min_val");  
                        config.max_val = channel_builder.required<std::float_t>("max_val");
                    }
            

                    this->reader_config.channels.push_back(config);
                });
    // assert(parser.ok());

}


void ni::niDaqReader::parse_digital_reader_config(config::Parser & parser)
{
    // Get Acquisition Rate and Stream Rates
    this->reader_config.acq_rate = parser.required<uint64_t>("acq_rate");
    this->reader_config.stream_rate = parser.required<uint64_t>("stream_rate"); 

    // device name
    this->reader_config.device_name = parser.required<std::string>("device_name");
    assert(parser.ok());

    // now parse the channels
    parser.iter("channels",
                [&](config::Parser &channel_builder){
                    ChannelConfig config;
                    config.channel_type = channel_builder.required<std::string>("channel_type");

                    // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
                    config.name = (config.channel_type == "index") ? (channel_builder.required<std::string>("name"))
                                                                    : (this->reader_config.device_name 
                                                                        + "/port" 
                                                                        + std::to_string(channel_builder.required<std::uint64_t>("port"))
                                                                        + "/line"  
                                                                        + std::to_string(channel_builder.required<std::uint64_t>("line")));

                    config.channel_key = channel_builder.required<uint32_t>("channel_key");

                    // TODO: there could be more than 2 state logic
                    config.min_val = 0;
                    config.max_val = 1;

                    this->reader_config.channels.push_back(config);
                });
    assert(parser.ok());
}


int ni::niDaqReader::init()
{
    int err = 0;
    auto channels = this->reader_config.channels;
    
    // iterate through channels
    for(auto &channel : channels){
        if(channel.channel_type == "analogVoltageInput"){
            err = this->checkNIError( DAQmxCreateAIVoltageChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_Cfg_Default, channel.min_val, channel.max_val, DAQmx_Val_Volts, NULL ));
        } else if(channel.channel_type == "digitalInput"){
            err = this->checkNIError( DAQmxCreateDIChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine));
        }
        this->numChannels++; // includes index channels TODO: how is this different form jsut channels.size()? 
        if (err < 0){
            LOG(ERROR) << "[NI Reader] failed while configuring channel " << channel.name;
            return -1;
        }
    }

    // Configure timing 
    // TODO: make sure there isnt different cases to handle between analog and digital
    if(this->checkNIError(DAQmxCfgSampClkTiming(  taskHandle,
                                            "",
                                            this->reader_config.acq_rate,
                                            DAQmx_Val_Rising,
                                            DAQmx_Val_ContSamps,
                                            this->reader_config.acq_rate))){
        LOG(ERROR) << "[NI Reader] failed while configuring timing for task " << this->reader_config.task_name;
        return -1;
    }

    // Configure buffer size and read resources
    this->numSamplesPerChannel = std::floor(this->reader_config.acq_rate / this->reader_config.stream_rate);
    this->bufferSize = this->numChannels * this->numSamplesPerChannel;
    if(this->reader_config.isDigital){
        this->digitalData = new uInt8[bufferSize];
    }
    else{
        this->data = new double[bufferSize];
    }

    LOG(INFO) << "[NI Reader] successfully configured NI hardware for task " << this->reader_config.task_name;
    return 0;
}


freighter::Error ni::niDaqReader::start()
{
    // TODO: don't let multiple starts happen (or handle it at least)
    freighter::Error err = freighter::NIL;
    if (this->checkNIError(DAQmxStartTask(taskHandle))){
        LOG(ERROR) << "[NI Reader] failed while starting task " << this->reader_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    } else{
        LOG(INFO) << "[NI Reader] successfully started task " << this->reader_config.task_name;
    }
    return err;
}


freighter::Error ni::niDaqReader::stop()
{ // TODO: don't let multiple stops happen (or handle it at least)
    freighter::Error err = freighter::NIL;

    if (this->checkNIError(DAQmxStopTask(taskHandle))){
        LOG(ERROR) << "[NI Reader] failed while stopping task " << this->reader_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else {
        if (this->checkNIError(DAQmxClearTask(taskHandle))){
            LOG(ERROR) << "[NI Reader] failed while clearing task " << this->reader_config.task_name;
            err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
        }
    }

    if(this->reader_config.isDigital){
        delete[] digitalData; 
    }
    else{
        delete[] data;
    }

    if(err == freighter::NIL){
        LOG(INFO) << "[NI Reader] successfully stopped and cleared task " << this->reader_config.task_name;
    }
    
    return err;
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::readAnalog()
{
    signed long samplesRead = 0;
    float64 flush[1000]; // to flush buffer before performing a read
    signed long flushRead = 0;    
    synnax::Frame f = synnax::Frame(numChannels);

    // initial read to flush buffer
    if(this->checkNIError(DAQmxReadAnalogF64(this->taskHandle,
                                                -1, // reads all available samples in buffer
                                                10.0,
                                                DAQmx_Val_GroupByChannel,
                                                flush,
                                                1000,
                                                &flushRead,
                                                NULL))){
        LOG(ERROR) << "[NI Reader] failed while flushing buffer for task " << this->reader_config.task_name;
        return std::make_pair(std::move(f), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "error reading analog data"));
    }
    
    // actual read of analog lines
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    if (this->checkNIError(DAQmxReadAnalogF64(this->taskHandle,
                                                this->numSamplesPerChannel,
                                                -1,
                                                DAQmx_Val_GroupByChannel,
                                                this->data,
                                                this->bufferSize,
                                                &samplesRead,
                                                NULL))) { 
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



std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::readDigital()
{
    signed long samplesRead;
    char errBuff[2048] = {'\0'};
    uInt8 flushBuffer[10000]; // to flush buffer before performing a read
    uInt8 dataBuffer[10000];
    signed long flushRead;
    synnax::Frame f = synnax::Frame(numChannels);
    int32 numBytesPerSamp ; //TODO do i need this?
    int err = 0;

    // initial read to flush buffer
    if(this->checkNIError(DAQmxReadDigitalLines(this->taskHandle, // TODO: come back to and make sure this call to flush will be fine at any scale (elham)
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

    // actual read to of digital lines
    std::uint64_t initial_timestamp = (synnax::TimeStamp::now()).value;
    if(this->checkNIError(DAQmxReadDigitalLines(this->taskHandle,           // task handle
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
    for (int i = 0; i < samplesRead; ++i)
    { 
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

std::pair<synnax::Frame, freighter::Error> ni::niDaqReader::read()
{

    if(this->reader_config.isDigital){
        return readDigital();
    }
    else{
        return readAnalog();
    }
}


int ni::niDaqReader::checkNIError(int32 error) 
{
    if (error < 0)
    {
        char errBuff[2048] = {'\0'};
        DAQmxGetExtendedErrorInfo(errBuff, 2048);
        this->err_info["error type"] = "Vendor Error";
        this->err_info["error details"] = errBuff;
        this->ok_state = false;
        this->ctx->setState({.task = this->reader_config.task_key,
                             .variant = "error",
                             .details = err_info});

        LOG(ERROR) << "[NI Reader] Vendor Error: " << this->err_info["error details"];

        return -1;
    }
    return 0;
}

bool ni::niDaqReader::ok() //TODO: make this inline?
{
    return this->ok_state;
}




///////////////////////////////////////////////////////////////////////////////////
//                                    niDaqWriter                                //
///////////////////////////////////////////////////////////////////////////////////

ni::niDaqWriter::niDaqWriter(
    TaskHandle taskHandle,
    const std::shared_ptr<task::Context> &ctx,
    const synnax::Task task)
    : taskHandle(taskHandle),
      ctx(ctx)
{
    // Create parser
    auto config_parser = config::Parser(task.config);
    this->writer_config.task_name = task.name;

    // Parse configuration and make sure it is valid
    this->parse_digital_writer_config(config_parser);
    if (!config_parser.ok()){
        // Log error
        LOG(ERROR) << "[NI Writer] failed to parse configuration for " << this->writer_config.task_name;
        this->ctx->setState({.task = task.key,
                             .variant = "error",
                             .details = config_parser.error_json()});
        this->ok_state = false;
        return;
    }
    LOG(INFO) << "[NI Writer] successfully parsed configuration for " << this->writer_config.task_name;

    
    // TODO: get device proprties for things like authentication

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
    if(this->init()){
        LOG(ERROR) << "[NI Writer] Failed while configuring NI hardware for task " << this->writer_config.task_name;
        this->ok_state = false;
    }
    return;
}


void ni::niDaqWriter::parse_digital_writer_config(config::Parser &parser){

    // device name
    this->writer_config.device_name = parser.required<std::string>("device_name");
    // now parse the channels
    parser.iter("channels",
                [&](config::Parser &channel_builder)
                {
                    ChannelConfig config;

                    config.channel_type = channel_builder.required<std::string>("channel_type");

                    // digital channel names are formatted: <device_name>/port<port_number>/line<line_number>
                    config.name = (config.channel_type == "index" || config.channel_type == "driveStateIndex") ? (channel_builder.required<std::string>("name"))
                                                                    : (this->writer_config.device_name + "/port" + std::to_string(channel_builder.required<std::uint64_t>("port")) + "/line" + std::to_string(channel_builder.required<std::uint64_t>("line")));


                    config.channel_key = channel_builder.required<uint32_t>("channel_key");

                    if((config.channel_type != "index") && (config.channel_type != "driveStateIndex")){
                        this->writer_config.drive_state_channel_keys.push_back(channel_builder.required<uint32_t>("drive_state_key"));
                        this->writer_config.drive_cmd_channel_keys.push_back(config.channel_key); 
                    }
         

                    // TODO: there could be more than 2 state  
                    config.min_val = 0;
                    config.max_val = 1;

                    this->writer_config.channels.push_back(config);

                    if(config.channel_type == "driveStateIndex"){
                        this->writer_config.drive_state_index_key = config.channel_key;
                    }
                });

    assert(this->writer_config.drive_state_index_key != 0);
    assert(this->writer_config.drive_state_channel_keys.size() > 0);
    assert(this->writer_config.drive_cmd_channel_keys.size() > 0);
    assert(this->writer_config.drive_cmd_channel_keys.size() == this->writer_config.drive_state_channel_keys.size());
}



int ni::niDaqWriter::init(){
    int err = 0;
    auto channels = this->writer_config.channels;
    
    // iterate through channels
    for(auto &channel : channels){
        if(channel.channel_type == "digitalOutput"){
            err = this->checkNIError(DAQmxCreateDOChan(taskHandle, channel.name.c_str(), "", DAQmx_Val_ChanPerLine));
        }
        this->numChannels++; // includes index channels TODO: how is this different form jsut channels.size()? 
        if (err < 0){
            LOG(ERROR) << "[NI Writer] failed while configuring channel " << channel.name;
            return -1;
        }
    }

    // Configure timing 
    // TODO: make sure there isnt different cases to handle between analog and digital

    // Configure buffer size and read resources
    this->bufferSize = this->numChannels;
    this->writeBuffer = new uint8_t[this->bufferSize];

    for (int i = 0; i < this->bufferSize; i++){
        writeBuffer[i] = 0;
    }

    LOG(INFO) << "[NI Writer] successfully configured NI hardware for task " << this->writer_config.task_name;
    return 0;
}

freighter::Error ni::niDaqWriter::start(){
    // TODO: don't let multiple starts happen (or handle it at least)
    freighter::Error err = freighter::NIL;
    if (this->checkNIError(DAQmxStartTask(this->taskHandle))){
        LOG(ERROR) << "[NI Writer] failed while starting task " << this->writer_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    } else{
        LOG(INFO) << "[NI Writer] successfully started task " << this->writer_config.task_name;
    }
    return err;
}

freighter::Error ni::niDaqWriter::stop(){ 
    // TODO: don't let multiple closes happen (or handle it at least)

    freighter::Error err = freighter::NIL;

    if (this->checkNIError(DAQmxStopTask(taskHandle))){
        LOG(ERROR) << "[NI Writer] failed while stopping task " << this->writer_config.task_name;
        err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
    }
    else {
        if (this->checkNIError(DAQmxClearTask(taskHandle))){
            LOG(ERROR) << "[NI Writer] failed while clearing task " << this->writer_config.task_name;
            err = freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR);
        }
    }

    delete[] writeBuffer;

    if(err == freighter::NIL){
        LOG(INFO) << "[NI Writer] successfully stopped and cleared task " << this->writer_config.task_name;
    }
    
    return err;
}

// Here to modify as we add more writing options
std::pair<synnax::Frame, freighter::Error> ni::niDaqWriter::write(synnax::Frame frame){ 
    // TODO: should this function get a Frame or a bit vector of setpoints instead?
    return writeDigital(std::move(frame));
}

std::pair<synnax::Frame, freighter::Error> ni::niDaqWriter::writeDigital(synnax::Frame frame){
    char errBuff[2048] = {'\0'};
    signed long samplesWritten = 0;
    formatData(std::move(frame));

    // Write digital data
    if(this->checkNIError(DAQmxWriteDigitalLines(this->taskHandle,
                                                    1,                        // number of samples per channel
                                                    1,                        // auto start
                                                    10.0,                     // timeout
                                                    DAQmx_Val_GroupByChannel, // data layout
                                                    writeBuffer,              // data
                                                    &samplesWritten,          // samples written
                                                    NULL))){
        LOG(ERROR) << "[NI Writer] failed while writing digital data for task " << this->writer_config.task_name;
        return std::make_pair(synnax::Frame(0), freighter::Error(driver::TYPE_CRITICAL_HARDWARE_ERROR, "Error reading digital data"));
    }


    // Construct acknowledgement frame (can only do this after a successful write to keep consistent over failed writes)
    auto ack_frame = synnax::Frame(this->drive_state_queue.size() + 1);
    ack_frame.add(this->writer_config.drive_state_index_key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));
    while (!this->drive_state_queue.empty())
    {
        auto ack_key = this->drive_state_queue.front();
        ack_frame.add(ack_key, synnax::Series(std::vector<uint8_t>{this->drive_queue.front()}));
        this->drive_state_queue.pop();
        this->drive_queue.pop();
    }

    // return acknowledgements frame to write to the ack channel
    return {std::move(ack_frame), freighter::NIL};
}

freighter::Error ni::niDaqWriter::formatData(synnax::Frame frame)
{
    uint32_t frame_index = 0;
    uint32_t cmd_channel_index = 0;
    for (auto key : *(frame.channels))
    { // the order the keys are in is the order the data is written
        auto it = std::find(this->writer_config.drive_cmd_channel_keys.begin(),this->writer_config.drive_cmd_channel_keys.end(), key);
        if (it != this->writer_config.drive_cmd_channel_keys.end())
        {
            cmd_channel_index = std::distance(this->writer_config.drive_cmd_channel_keys.begin(), it);
            auto series = frame.series->at(frame_index).uint8(); // used to be auto &series
            writeBuffer[cmd_channel_index] = series[0];
            this->drive_state_queue.push(this->writer_config.drive_state_channel_keys[cmd_channel_index]);
            this->drive_queue.push(series[0]);
        }
        frame_index++;
    }
    return freighter::NIL;
}

int ni::niDaqWriter::checkNIError(int32 error) //TODO: make this inline?
{
    if (error < 0)
    {
        char errBuff[2048] = {'\0'};
        DAQmxGetExtendedErrorInfo(errBuff, 2048);
        this->err_info["error type"] = "Vendor Error";
        this->err_info["error details"] = errBuff;
        this->ok_state = false;
        this->ctx->setState({.task = this->writer_config.task_key,
                             .variant = "error",
                             .details = err_info});
        LOG(ERROR) << "[NI Reader] Vendor Error: " << this->err_info["error details"];
        return -1;
    }
    return 0;
}

bool ni::niDaqWriter::ok() //TODO: make this inline?
{
    return this->ok_state;
}

// TODO create a helper function that takes in a frame and formats into the data to pass into writedigital
// TODO: create a helper function to parse digital data configuration of wehtehr its a port to r line
