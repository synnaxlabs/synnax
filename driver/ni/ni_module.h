// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
#include "driver/pipeline/acqReader.h"
#include "driver/pipeline/ctrl.h"

#pragma once

class NiAnalogReaderTask : public module::Module {
public:
    NiAnalogReaderTask(){};
    void init(const std::shared_ptr<synnax::Synnax> client,
                          std::unique_ptr<daq::AcqReader> daq_reader,
                          synnax::WriterConfig writer_config);

    freighter::Error startAcquisition();
    freighter::Error stopAcquisition();
private:
    Acq acq_pipeline;
};

class NiDigitalReaderTask : public module::Module {
public:
    NiDigitalReaderTask(){};
    void init(const std::shared_ptr<synnax::Synnax> client,
              std::unique_ptr<daq::AcqReader> daq_reader,
              synnax::WriterConfig writer_config);

    freighter::Error startAcquisition();
    freighter::Error stopAcquisition();
private:
    Acq acq_pipeline;
};

// TODO: createDigitalWriterTask
class NiDigitalWriterTask : public module::Module {
public:
    NiDigitalWriterTask(){};
    void init(const std::shared_ptr<synnax::Synnax> client,
              std::unique_ptr<daq::daqWriter> daq_writer,
              synnax::WriterConfig writer_config,
              synnax::StreamerConfig streamer_config);

    freighter::Error startAcquisition();
    freighter::Error stopAcquisition();
private:
    pipeline::Ctrl ctrl_pipeline;
};




class niTaskFactory : public module::Factory {
public:
    niTaskFactory() {};
    std::unique_ptr<module::Module> createModule(TaskHandle taskhandle,
                                                 const std::shared_ptr<synnax::Synnax> &client,
                                                 const json &config,
                                                 bool &valid_config,
                                                 json &config_err);

    std::unique_ptr <NiAnalogReaderTask> createAnalogReaderTask(TaskHandle taskhandle,
                                                                std::shared_ptr<synnax::Synnax> client,
                                                                bool &valid_config,
                                                                const json &config,
                                                                json &config_err);

    std::unique_ptr <NiDigitalReaderTask> createDigitalReaderTask(TaskHandle taskhandle,
                                                                std::shared_ptr<synnax::Synnax> client,
                                                                bool &valid_config,
                                                                const json &config,
                                                                json &config_err);

    std::unique_ptr <NiDigitalWriterTask> createDigitalWriterTask(TaskHandle taskhandle,
                                                                std::shared_ptr<synnax::Synnax> client,
                                                                bool &valid_config,
                                                                const json &config,
                                                                json &config_err);

    bool validChannelConfig(const json &config, json &config_err);

    // TODO: createDigitalReaderTask
    // TODO: createDigitalWriterTask
    
};


