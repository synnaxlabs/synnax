// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once
#include "synnax/synnax.h"
#include <atomic>
#include <memory>
#include <thread>
#include "driver/breaker/breaker.h"
#include "acqReader.h"


class Acq { // Acquisition Pipeline Class
public:
    void start();
    void stop();
    Acq(std::unique_ptr<daq::AcqReader> daq_reader,
        synnax::WriterConfig writer_config,
        synnax::StreamerConfig streamer_config);

private:
    /// @brief threading.
    std::atomic<bool> running;
    std::thread exec_thread;

    /// @brief synnax IO.
    std::unique_ptr<synnax::Synnax> client;

    /// @brief synnax streamer.
    std::unique_ptr<synnax::Streamer> streamer;
    synnax::StreamerConfig streamer_config;

    /// @brief synnax writer
    std::unique_ptr<synnax::Writer> writer;
    synnax::WriterConfig writer_config;

    /// @brief commit tracking;
    synnax::TimeSpan commit_interval = synnax::TimeSpan(1); // TODO: comeback to and move to constructor?
    synnax::TimeStamp last_commit;

    /// @brief driver comms.
    synnax::ChannelKey comms_channel_key;

    /// @brief daq interface
    std::unique_ptr<daq::AcqReader> daq_reader;

    /// @brief breaker
    std::unique_ptr<breaker::Breaker> breaker;

    /// @brief mutex for shared variables
    static std::mutex mut;

    void run();
    void runInternal();
};
