//
// Created by Synnax on 1/24/2024.
//

#ifndef DRIVER_NI__MODULE_H
#define DRIVER_NI__MODULE_H

#endif //DRIVER_NI__MODULE_H

#include "synnax/synnax.h"
#include <memory>
#include <utility>
#include "nlohmann/json.hpp"
#include "driver/modules/module.h"

#pragma once

class NiAnalogReader : module {
public:
    std::vector <channel_config> channels;
    Acq acq;
    std::uint64_t acq_rate;
    std::uint64_t stream_rate;
    std::uint64_t batch_size;
    std::uint64_t num_channels;

    NiAnalogReader(channels, acq_rate, stream_rate) : channels(channels), acq_rate(acq_rate), stream_rate(stream_rate) {}

    freighter::Error init(std::uint64_t acq_r, std::uint64_t stream_r, uint64_t num_channels,
                          std::unique_ptr<daq::AcqReader> daq_reader,
                          std::unique_ptr<synnax::WriterConfig> writer_config,
                          std::unique_ptr<synnax::StreamerConfig> streamer_config);

    freighter::Error startAcquisition();
    freighter::Error stopAcquisition();

}