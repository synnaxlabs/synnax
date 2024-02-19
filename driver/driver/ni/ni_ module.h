//
// Created by Synnax on 1/24/2024.
//

#ifndef DRIVER_NI__MODULE_H
#define DRIVER_NI__MODULE_H

#endif //DRIVER_NI__MODULE_H

#include "synnax/synnax.h"
#include <memory>
#include <utility>
#include "nlohmann/json.hpp" // for json parsing
#include "driver/modules/module.h"
#include "driver/pipeline/acq.h"
#include "driver/ni/ni_reader.h" // to get channel config info

#pragma once

class NiAnalogReaderTask : public module {
public:
    std::vector <channel_config> channels;
    Acq acq_pipeline;
    std::uint64_t acq_rate;
    std::uint64_t stream_rate;
    std::uint64_t num_channels;


    NiAnalogReaderTask() = default;

    freighter::Error init(std::unique_ptr<daq::AcqReader> daq_reader,
                          std::unique_ptr<synnax::WriterConfig> writer_config,
                          std::unique_ptr<synnax::StreamerConfig> streamer_config);

    freighter::Error startAcquisition();
    freighter::Error stopAcquisition();
}

// TODO: createDigitalReaderTask
// TODO: createDigitalWriterTask


class niTaskFactory : public module::Factory {
public:
    std::unique_ptr<module::Module> createModule(const std::shared_ptr<synnax::Synnax> &client,
                                                 const json &config,
                                                 bool &valid_config,
                                                 json &config_err);

private:
    std::<unique_ptr<NiAnalogReaderTask>> createAnalogReaderTask(const json &config, json &config_err);

    bool validChannelConfig(const json &config, json &config_err);

    // TODO: createDigitalReaderTask
    // TODO: createDigitalWriterTask
    
};
