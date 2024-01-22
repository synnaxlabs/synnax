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
//
//namespace daq {
//class Writer {
//public:
//    std::pair<synnax::Frame, freighter::Error> write(synnax::Frame);
//    void start();
//    void stop();
//};
//}
//
//namespace pipeline {
//class Inbound {
//public:
//    void start();
//    void stop();
//private:
//    /// @brief threading.
//    std::atomic<bool> running;
//    std::thread exec_thread;
//
//    /// @brief synnax IO.
//    std::unique_ptr<synnax::Synnax> client;
//    std::unique_ptr<synnax::Streamer> streamer;
//    synnax::StreamerConfig streamer_config;
//    std::unique_ptr<synnax::Writer> writer;
//    synnax::WriterConfig writer_config;
//
//
//    /// @brief daq interface.
//    std::unique_ptr<daq::Writer> daq_writer;
//
//    void execute();
//};
//}