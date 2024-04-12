// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//
// Created by Synnax on 2/18/2024.
//

#include "driver/driver/ni/ni_module.h"
#include "ni_module.h"
#include <cassert>
#include <stdio.h>


/* NiAnalogReaderTask */
void NiAnalogReaderTask::init(const std::shared_ptr <Synnax> client,
                                          std::unique_ptr <daq::AcqReader> daq_reader,
                                          synnax::WriterConfig writer_config) {
    printf("Initializing Analog Reader Task\n");
    this->acq_pipeline = Acq(writer_config, client, std::move(daq_reader));
    printf("Initialized Analog Reader Task\n");
}


freighter::Error NiAnalogReaderTask::startAcquisition(){
    printf("Starting Acq Pipeline\n");
    this->acq_pipeline.start();
    return freighter::TYPE_NIL;
}

freighter::Error NiAnalogReaderTask::stopAcquisition(){
    printf("Stopping Acq Pipeline\n");
    this->acq_pipeline.stop();
    return freighter::TYPE_NIL;
}

/* NiDigitalReaderTask */
void NiDigitalReaderTask::init(const std::shared_ptr <Synnax> client,
                              std::unique_ptr <daq::AcqReader> daq_reader,
                              synnax::WriterConfig writer_config) {
    printf("Initializing Analog Reader Task\n");
    this->acq_pipeline = Acq(writer_config, client, std::move(daq_reader));
    printf("Initialized Analog Reader Task\n");
}


freighter::Error NiDigitalReaderTask::startAcquisition(){
    printf("Starting Acq Pipeline\n");
    this->acq_pipeline.start();
    return freighter::TYPE_NIL;
}

freighter::Error NiDigitalReaderTask::stopAcquisition(){
    printf("Stopping Acq Pipeline\n");
    this->acq_pipeline.stop();
    return freighter::TYPE_NIL;
}


/* NiDigitalWriterTask */
void NiDigitalWriterTask::init(const std::shared_ptr <Synnax> client,
                               std::unique_ptr <daq::daqWriter> daq_writer,
                               synnax::WriterConfig writer_config,
                               synnax::StreamerConfig streamer_config) {
    printf("Initializing Digital Writer Task\n");
    this->ctrl_pipeline = pipeline::Ctrl(streamer_config, writer_config, client, std::move(daq_writer));
    printf("Initialized Digital Writer Task\n");
}

freighter::Error NiDigitalWriterTask::startAcquisition(){
    printf("Starting Ctrl Pipeline\n");
    this->ctrl_pipeline.start();
    return freighter::TYPE_NIL;
}

freighter::Error NiDigitalWriterTask::stopAcquisition(){
    printf("Stopping Ctrl Pipeline\n");
    this->ctrl_pipeline.stop();
    return freighter::TYPE_NIL;
}

/* niTaskFactory */

bool niTaskFactory::validChannelConfig(const json &config, json &config_err){
    printf("Validate Config \n");
    if (config.find("channels") == config.end()){ // TODO: also assert that there is at least one channel entry
        config_err = "Property: channels - not found in config"; // TODO: change these errors
        return false;
    }
    if (config.find("acq_rate") == config.end()){
        config_err = "Property: acq_rate - not found in config";
        return false;
    }
    if (config.find("stream_rate") == config.end()){
        config_err = "Property: stream_rate - not found in config";
        return false;
    }
    if (config.find("hardware") == config.end()){
        config_err = "Property: hardware - not found in config";
        return false;
    }
    return true;
}

std::unique_ptr<module::Module> niTaskFactory::createModule(TaskHandle taskhandle,
                                                            const std::shared_ptr<synnax::Synnax> &client,
                                                            const json &config,
                                                            bool &valid_config,
                                                            json &config_err){
    printf("createModule \n");
    // validate config
    auto err = validChannelConfig(config, config_err);
    if (!err){
        valid_config = false;
        return nullptr;
    }

    // create module
    auto type = config["channels"][0]["type"];

    // TODO: change this to a ternary operator
    if (type == "analogVoltageInput"){
        return createAnalogReaderTask(taskhandle, client, config, config_err); // TODO: implict cast from unique_ptr of NiAnalogReaderTask to unique_ptr of module::Module?
    }
    else if(type == "digitalInput"){
        return createDigitalReaderTask(taskhandle, client, config, config_err); // TODO: implict cast from unique_ptr of NiDigitalReaderTask to unique_ptr of module::Module?
    } else if(type == "digitalOutput"){
        return createDigitalWriterTask(taskhandle,client, config, config_err); // TODO: createAnalogWriterTask
    }
    else {
        valid_config = false;
        config_err = "Invalid module type";
        return nullptr;
    }
}


std::unique_ptr <NiAnalogReaderTask> niTaskFactory::createAnalogReaderTask(TaskHandle taskhandle,
                                                                           std::shared_ptr<synnax::Synnax> client,
                                                                           bool &valid_config,
                                                                           const json &config,
                                                                           json &config_err){
    std::uint64_t acq_rate;
    std::uint64_t stream_rate;
    std::uint64_t num_channels;

    // parse config

    json channels = config["channels"];
    acq_rate = uInt64(config["acq_rate"]);
    stream_rate = uInt64(config["stream_rate"]);

    //print acq and stream rate
    printf("Acq Rate: %d\n", acq_rate);         //TODO: remove
    printf("Stream Rate: %d\n", stream_rate);   //TODO: remove


    // create vector of channel keys to construct writer
    std::vector<synnax::ChannelKey> channel_keys;
    std::vector<synnax::Authority> authorities;
    for (auto &channel : channels){
        //convert channel key to synnax::ChannelKey
        uint64_t channel_key = uInt64(channel["channel"]);
        channel_keys.push_back(channel_key);
        authorities.push_back(synnax::ABSOLUTTE); // TODO: can diff channels for analog reader  task have diff authorities?
    }


    // Concatenate analog_reader  with device name
    std::string devName = config["hardware"];
    std::string writerName = devName + "_analog_reader"; //TODO:  Is this the right convention?

    //create writer config
    auto writer_config = synnax::WriterConfig{
        channel_keys,
        synnax::TimeStamp::now(),
        authorities,
        synnax::Subject{writerName}
    };

    // create daq_reader and init
    auto daq_reader = std::make_unique<ni::niDaqReader>(taskhandle);
    auto [err_info, err] = daq_reader->init(config, acq_rate, stream_rate);
    if(err < 0){
        config_err = err_info;
        valid_config = false;
        return nullptr;
    }

    //create module
    auto module = std::make_unique<NiAnalogReaderTask>();
    module->init(client, std::move(daq_reader), writer_config);
    std::cout << "Creating Analog Reader Task" << std::endl;
    return module;

}

std::unique_ptr <NiDigitalReaderTask> niTaskFactory::createDigitalReaderTask(TaskHandle taskhandle,
                                                                           std::shared_ptr<synnax::Synnax> client,
                                                                           bool &valid_config,
                                                                           const json &config,
                                                                           json &config_err){
    std::uint64_t acq_rate;
    std::uint64_t stream_rate;
    std::uint64_t num_channels;

    // parse config

    json channels = config["channels"];
    acq_rate = uInt64(config["acq_rate"]);
    stream_rate = uInt64(config["stream_rate"]);

    //print acq and stream rate
    printf("Acq Rate: %d\n", acq_rate);
    printf("Stream Rate: %d\n", stream_rate);


    // create vector of channel keys to construct data writer
    std::vector<synnax::ChannelKey> channel_keys;
    std::vector<synnax::Authority> authorities;
    for (auto &channel : channels){
        //convert channel key to synnax::ChannelKey
        uint64_t channel_key = uInt64(channel["channel"]);
        channel_keys.push_back(channel_key);
        authorities.push_back(synnax::ABSOLUTTE); // TODO: can diff channels for analog reader  task have diff authorities?
    }


    // Concatenate analog_reader  with device name
    std::string devName = config["hardware"];
    std::string writerName = devName + "_digital_reader"; //TODO:  Is this the right convention?

    //create writer config
    auto writer_config = synnax::WriterConfig{
            channel_keys,
            synnax::TimeStamp::now(),
            authorities,
            synnax::Subject{writerName}
    };

    // create daq_reader and init
    auto daq_reader = std::make_unique<ni::niDaqReader>(taskhandle);
    auto [err_info, err] = daq_reader->init(config, acq_rate, stream_rate);
    if(err < 0){
        config_err = err_info;
        valid_config = false;
        return nullptr;
    }
    //create module
    auto module = std::make_unique<NiDigitalReaderTask>();
    module->init(client, std::move(daq_reader), writer_config);
    std::cout << "Creating Analog Reader Task" << std::endl;
    return module;
}

std::unique_ptr <NiDigitalWriterTask> niTaskFactory::createDigitalWriterTask(TaskHandle taskhandle,
                                                                             std::shared_ptr<synnax::Synnax> client,
                                                                             bool &valid_config,
                                                                             const json &config,
                                                                             json &config_err){
    std::uint64_t acq_rate; //TODO: I dont need acq_rate and stream_rate here?
    std::uint64_t stream_rate;
    std::uint64_t num_channels;

    // parse config

    json channels = config["channels"];
    acq_rate = uInt64(config["acq_rate"]);
    stream_rate = uInt64(config["stream_rate"]);

    //print acq and stream rate
    printf("Acq Rate: %d\n", acq_rate);
    printf("Stream Rate: %d\n", stream_rate);

    // create vector of channel keys to construct ack writer
    std::vector<synnax::ChannelKey> ack_channel_keys;
    std::vector<synnax::Authority> ack_authorities;
    std::vector<synnax::ChannelKey> cmd_channel_keys;
    synnax::ChannelKey ack_idx_key;

    for (auto &channel : channels){
        //convert channel key to synnax::ChannelKey
        if(channel["type"] == "digitalOutput"){
            uint64_t ack_key = uInt64(channel["ack_key"]);
            ack_channel_keys.push_back(ack_key);
            uint64_t cmd_key = uInt64(channel["cmd_key"]);
            cmd_channel_keys.push_back(cmd_key);
            ack_authorities.push_back(synnax::ABSOLUTTE); // TODO: can diff channels for analog reader  task have diff authorities?
        }else if( channel["type"] == "index"){
            uint64_t channel_key = uInt64(channel["channel"]);
            ack_channel_keys.push_back(channel_key);
            ack_authorities.push_back(synnax::ABSOLUTTE); // TODO: can diff channels for analog reader  task have diff authorities?
        } else if(channel["type"] == "ackIndex"){ // to give to the daqWriter
            uint64_t channel_key = uInt64(channel["channel"]);
            ack_idx_key = channel_key;
            ack_channel_keys.push_back(channel_key);
        }
    }
    assert(ack_channel_keys.size() > 0);
    assert(ack_authorities.size() > 0);
    // TODO: assert that ack_idx is found

    // concatenate digital_writer with device name
    std::string devName = config["hardware"];
    std::string writerName = devName + "_digital_writer"; //TODO:  Is this the right convention?

    // create a writer config to writer ack channels
    auto ack_writer_config = synnax::WriterConfig{
            ack_channel_keys,
            synnax::TimeStamp::now(),
            ack_authorities,
            synnax::Subject{writerName}
    };

    // create a streamer config to stream incoming cmds
    auto cmd_streamer_config = synnax::StreamerConfig{
            cmd_channel_keys,
            synnax::TimeStamp::now(),
    };

    // instatiate daq_writer and init
    auto daq_writer = std::make_unique<ni::niDaqWriter>(taskhandle);
    auto [err_info, err] = daq_writer->init(config, ack_idx_key);
    if(err < 0){
        config_err = err_info;
        valid_config = false;
        return nullptr;
    }

    // create module
    auto module = std::make_unique<NiDigitalWriterTask>();
    module->init(client, std::move(daq_writer), ack_writer_config, cmd_streamer_config);
    std::cout << "Creating Digital Writer Task" << std::endl;
    return module;
}