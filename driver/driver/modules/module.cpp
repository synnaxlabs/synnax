//
// Created by Synnax on 1/24/2024.
//

#include <thread>
#include "driver/pipeline/acq.h"
#include "driver/errors/errors.h"
#include "nlohmann/json.hpp"
#include "driver/modules/module.h"

#pragma once

freighter::Error NiAnalogReader::init(std::uint64_t acq_r, std::uint64_t stream_r, uint64_t num_channels,
                      std::unique_ptr<daq::AcqReader> daq_reader,
                      std::unique_ptr<synnax::WriterConfig> writer_config,
                      std::unique_ptr<synnax::StreamerConfig> streamer_config){

    //TODO: move daq_reader instantiation into this function to expose as little of the underlying imnplementation as possible (elham)
    acq_rate = acq_r;
    stream_rate = stream_r;
    channels.resize(num_channels);
    // instantiate acquistion pipeline
    acq = Acq(std::move(daq_reader), std::move(writer_config), std::move(streamer_config));

    return freighter::TYPE_NIL;
}

// Calls the start function of the acquisition pipeline which instantiates a thread to run
freighter::Error startAcquisition(){
    acq.start();
    return freighter::TYPE_NIL;
}

freighter::Error stopAcquisition(){
    acq.stop();
    return freighter::TYPE_NIL;
}