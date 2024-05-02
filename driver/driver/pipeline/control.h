
#pragma once

#include <memory>
#include <thread>
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
        const breaker::Config &breaker_config
    );
private:
    std::shared_ptr<task::Context> ctx;

    /// @brief writer thread.
    bool cmd_running = false;
    std::thread cmd_thread;

    /// @brief synnax writer
    std::unique_ptr<synnax::Streamer> streamer;
    synnax::StreamerConfig streamer_config;

    /// @brief synnax writer
    synnax::WriterConfig writer_config;

    /// @brief daq interface
    std::unique_ptr<Sink> sink;

    /// @brief breaker
    breaker::Breaker cmd_breaker;

    void run();

};
}