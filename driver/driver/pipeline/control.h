// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>
#include <thread>
#include <latch>
#include "client/cpp/synnax/synnax.h"
#include "driver/driver/task/task.h"
#include "driver/driver/breaker/breaker.h"

namespace pipeline {
class Sink {
public:
    virtual freighter::Error write(synnax::Frame frame) = 0;
};

class Control {
public:
    void start();

    void stop();

    Control(
        std::shared_ptr<task::Context> ctx,
        synnax::StreamerConfig streamer_config,
        synnax::WriterConfig writer_config,
        std::unique_ptr<Sink> sink,
        breaker::Breaker breaker,
    std::shared_ptr<task::Context> ctx;

    /// @brief writer thread.
    bool cmd_running = false;
    std::thread cmd_thread;

    /// @brief acks thread.
    bool state_running = false;
    std::thread acks_thread;

    std::mutex state_mutex;
    synnax::Rate state_rate;
    synnax::Frame curr_state;
    synnax::Writer state_writer;


    /// @brief synnax writer
    std::unique_ptr<synnax::Streamer> streamer;
    synnax::StreamerConfig streamer_config;

    /// @brief synnax writer
    synnax::WriterConfig writer_config;

    /// @brief daq interface
    std::unique_ptr<Sink> sink;

    /// @brief breaker
    breaker::Breaker state_breaker;
    breaker::Breaker cmd_breaker;

    void runCommands(std::latch &latch);

    void runStateUpdates(std::latch &latch);
};
}
