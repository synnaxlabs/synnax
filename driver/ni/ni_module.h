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

#include "client/cpp/synnax.h"
#include <memory>
#include <utility>
#include "nlohmann/json.hpp" // for json parsing
#include "driver/modules/module.h"
#include "driver/pipeline/acq.h"
#include "driver/ni/ni_reader.h" // to get channel config info
#include "driver/pipeline/daq_reader.h"
#include "driver/pipeline/ctrl.h"

#pragma once


// namespace ni{
    
//     ///////////////////////////////////////////////////////////////////////////////////
//     //                                    NiReaderTask                               //
//     ///////////////////////////////////////////////////////////////////////////////////
//     class NiReaderTask final : public task::Task {
//     public:
//         explicit NiReaderTask(  const std::shared_ptr<task::Context> &ctx, 
//                                 synnax::Task task); 

//         static std::unique_ptr<task::Task> configure(
//             const std::shared_ptr<task::Context> &ctx,
//             const synnax::Task &task
//         );
                                
//         void exec(task::Command &cmd) override;

//         void stop() override{};
//     private:
//         pipeline::Acquisition daq_read_pipe; // source is a daqreader 
//         Taskhandle taskhandle;
//     }

//     ///////////////////////////////////////////////////////////////////////////////////
//     //                                    NiWriterTask                               //
//     ///////////////////////////////////////////////////////////////////////////////////
//     class NiWriterTask final : public task::Task {
//     public:
//         explicit NiWriterTask(  const std::shared_ptr<task::Context> &ctx, 
//                                 synnax::Task task); 

        
//         static std::unique_ptr<task::Task> configure(
//             const std::shared_ptr<task::Context> &ctx,
//             const synnax::Task &task
//         );

//         void exec(task::Command &cmd) override;
//         void stop() override{};
//     private:
//         pipeline::Acquisition cmd_read_pipe; // source reads from synnax (TODO: make this source)
//         pipeline::Control state_write_pipe;
//         Taskhandle taskhandle;
//     }


//     //////////////////////////////////////////////// OLD CODE



//     class niTaskFactory : public module::Factory {
//     public:
//         niTaskFactory() {};
//         std::unique_ptr<module::Module> createModule(TaskHandle taskhandle,
//                                                     const std::shared_ptr<synnax::Synnax> &client,
//                                                     const json &config,
//                                                     bool &valid_config,
//                                                     json &config_err);

//         std::unique_ptr <NiAnalogReaderTask> createAnalogReaderTask(TaskHandle taskhandle,
//                                                                     std::shared_ptr<synnax::Synnax> client,
//                                                                     bool &valid_config,
//                                                                     const json &config,
//                                                                     json &config_err);

//         std::unique_ptr <NiDigitalReaderTask> createDigitalReaderTask(TaskHandle taskhandle,
//                                                                     std::shared_ptr<synnax::Synnax> client,
//                                                                     bool &valid_config,
//                                                                     const json &config,
//                                                                     json &config_err);

//         std::unique_ptr <NiDigitalWriterTask> createDigitalWriterTask(TaskHandle taskhandle,
//                                                                     std::shared_ptr<synnax::Synnax> client,
//                                                                     bool &valid_config,
//                                                                     const json &config,
//                                                                     json &config_err);

//         bool validChannelConfig(const json &config, json &config_err);

//         // TODO: createDigitalReaderTask
//         // TODO: createDigitalWriterTask
        
//     };


// }