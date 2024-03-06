// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "synnax/synnax.h"
#include "driver/pipeline/acq.h"
#include <atomic>
#include <memory>
#include <thread>

#pragma once
namespace pipeline {
    class Ctrl {
    public:
        void start();
        void stop();
        Ctrl(synnax::StreamerConfig streamer_config,
             synnax::WriterConfig writer_config,
             std::shared_ptr<synnax::Synnax> client,
             std::unique_ptr<daq::daqWriter> daq_writer);
        Ctrl();

    private:
/// @brief threading.
        bool running = false;
        std::thread ctrl_thread;

        /// @brief synnax IO.
        std::shared_ptr <synnax::Synnax> client;

        /// @brief synnax writer
        std::unique_ptr <synnax::Streamer> streamer;
        synnax::StreamerConfig streamer_config;

        /// @brief synnax writer
        std::unique_ptr <synnax::Writer> writer; //TODO: dont use of get rid of ?
        synnax::WriterConfig writer_config;

        /// @brief daq interface
        std::unique_ptr <daq::daqWriter> daq_writer;

        /// @brief breaker
        std::unique_ptr <breaker::Breaker> breaker; // What am I using this for
        void run();

        /// @brief commit tracking;
        synnax::TimeSpan commit_interval = synnax::TimeSpan(1); // TODO: comeback to and move to constructor?
        synnax::TimeStamp last_commit;



    };
}