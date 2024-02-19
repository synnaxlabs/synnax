//
// Created by Synnax on 2/18/2024.
//

#include "driver/ni/ni_module.h"

/* NiAnalogReaderTask */

freighter::Error NiAnalogReaderTask::init(std::unique_ptr<daq::AcqReader> daq_reader,
                                          std::unique_ptr<synnax::WriterConfig> writer_config){

    acq_pipeline = Acq(std::move(writer_config), std::move(client), std::move(daq_reader)));
    return freighter::TYPE_NIL;
}

freighter::Error startAcquisition(){
    acq_pipeline.start();
    return freighter::TYPE_NIL;
}

freighter::Error stopAcquisition(){
    acq_pipeline.stop();
    return freighter::TYPE_NIL;
}

/* NiDigitalReaderTask */

/* NiDigitalWriterTask */

/* niTaskFactory */

bool niTaskFactory::validChannelConfig(const json &config, json &config_err){
    if (config.find("channels") == config.end()){ // TODO: also assert that there is at least one channel entry
        config_err = "Property: channels - not found in config";
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
    if (config.find("device") == config.end()){
        config_err = "Property: device - not found in config";
        return false;
    }
    return true;
}

std::unique_ptr<module::Module> niTaskFactory::createModule(const std::shared_ptr<synnax::Synnax> &client,
                                                            const json &config,
                                                            bool &valid_config,
                                                            json &config_err){
    // validate config
    auto err = validChannelConfig(&config, &config_err);
    if (!err){
        valid_config = false;
        return nullptr;
    }

    // create module
    auto type = conifg["channels"][0]["type"]
    if (type == "analogVoltageInput"){
        return createAnalogReaderTask(config, config_err); // TODO: implict cast from unique_ptr of NiAnalogReaderTask to unique_ptr of module::Module?
    }
   else {
        valid_config = false;
        config_err = "Invalid module type";
        return nullptr;
    }
}


std::<unique_ptr<NiAnalogReaderTask>> niTaskFactory::createAnalogReaderTask(const json &config, json &config_err){
    std::vector<channel_config> channels;
    std::uint64_t acq_rate;
    std::uint64_t stream_rate;
    std::uint64_t num_channels;

    // parse config
    json channels = config["channels"];
    acq_rate = config["acq_rate"];
    stream_rate = config["stream_rate"];

    // create vector of channel keys to construct writer
    std::vector<synnax::ChannelKey> channel_keys;
    for (auto &channel : channels){
        channel_keys.push_back(channel["key"]);
    }

    // Concatenate analog_reader  with device name
    std::string writerName = config["device"] + "_analog_reader"; //TODO:  Is this the right convention?

    // create writer config
    auto writer_config = std::make_unique<synnax::WriterConfig>(channel_keys,
                                                                synnax::TimeStamp::now(),
                                                                synnax::ABSOLUTTE,
                                                                synnax::Subject{writerName});


    // create module
    auto daq_reader = std::make_unique<ni::niDaqReader>();
    daq_reader->init(channels, acq_rate, stream_rate);


}

