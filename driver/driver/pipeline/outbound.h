// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "synnax/synnax.h"
#include <atomic>
#include <memory>
#include <thread>

#pragma once

namespace daq {
class Reader {
public:
    virtual std::pair<synnax::Frame, freighter::Error> read() = 0;

    virtual freighter::Error start() = 0;

    virtual freighter::Error stop() = 0;
};
}

namespace pipeline {
class Breaker {
public:
    virtual bool wait() = 0;
};


class Outbound {
public:
    void start();

    void stop();

private:
    /// @brief threading.
    std::atomic<bool> running;
    std::thread exec_thread;

    /// @brief synnax writer.
    std::unique_ptr<synnax::Synnax> client;
    synnax::WriterConfig writer_config;

    /// @brief commit tracking;
    synnax::TimeSpan commit_interval;
    synnax::TimeStamp last_commit;

    /// @brief driver comms.
    synnax::ChannelKey comms_channel_key;

    /// @brief breaker
    std::unique_ptr<Breaker> breaker;

    /// @brief daq interface.
    std::unique_ptr<daq::Reader> daq_reader;

    void run();

    void runInternal();
};
}