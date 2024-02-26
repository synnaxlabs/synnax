//
// Created by Synnax on 2/18/2024.
//

#include "driver/ni/ni_module.h"
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

/* NiDigitalWriterTask */

/* niTaskFactory */

bool niTaskFactory::validChannelConfig(const json &config, json &config_err){
    printf("Validate Config \n");
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
    if (type == "analogVoltageInput"){
        return createAnalogReaderTask(taskhandle, client, config, config_err); // TODO: implict cast from unique_ptr of NiAnalogReaderTask to unique_ptr of module::Module?
    }
   else {
        valid_config = false;
        config_err = "Invalid module type";
        return nullptr;
    }
}


std::unique_ptr <NiAnalogReaderTask> niTaskFactory::createAnalogReaderTask(TaskHandle taskhandle,
                                                                           std::shared_ptr<synnax::Synnax> client,
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
    std::string devName = config["device"];
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
    daq_reader->init(config, acq_rate, stream_rate);
    //create module

    auto module = std::make_unique<NiAnalogReaderTask>();
    module->init(client, std::move(daq_reader), writer_config);
    std::cout << "Creating Analog Reader Task" << std::endl;
    return module;

}

