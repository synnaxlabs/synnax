// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include "driver/pipeline/control.h"

#include <utility>
#include "driver/errors/errors.h"
#include <exception>
#include <stdexcept>

using namespace pipeline;

Control::Control(
    std::shared_ptr<task::Context> ctx,
    synnax::StreamerConfig streamer_config,
    std::unique_ptr<pipeline::Sink> sink,
    const breaker::Config &breaker_config
):  ctx(std::move(ctx)),
    thread(nullptr),
    streamer_config(std::move(streamer_config)),
    sink(std::move(sink)),
    breaker(breaker::Breaker(breaker_config)) {
}


void Control::start() {
    if (breaker.running()) return;
    if (this->thread != nullptr && thread->joinable() && std::this_thread::get_id() != thread->get_id())
        thread->join();
    breaker.start();
    auto s_err = sink->start();
    if (s_err) {
        LOG(ERROR) << "[acquisition] Failed to start source: " << s_err.message();
        if (s_err.matches(driver::TYPE_TEMPORARY_HARDWARE_ERROR) && breaker.wait(
                s_err.message()))
            run();
        return;
    }  
    thread = std::make_unique<std::thread>(&Control::run, this);
}

void Control::stop() {
    if(breaker.running()) return;
    breaker.stop(); 
    if (this->thread != nullptr && thread->joinable() && std::this_thread::get_id() != thread->get_id()) {
        thread->join();
    };
    this->streamer->closeSend();
    sink->stop();
    LOG(INFO) << "[control] Control stopped";
}

void Control::runInternal() {
    this->streamer_config.start = synnax::TimeStamp::now();
    auto [test, so_err] = ctx->client->telem.openStreamer(this->streamer_config);
    this->streamer = std::make_unique<synnax::Streamer>(std::move(test));
    if (so_err) {
        if (    so_err.matches(freighter::UNREACHABLE) 
            &&  breaker.wait(so_err.message())) {
            return runInternal();
        }
    }
    while (breaker.running()) {
        auto [cmd_frame, cmd_err] = this->streamer->read();
        if (cmd_err) break;
        auto daq_err = sink->write(std::move(cmd_frame));    
        //    breaker.reset();
    }
    const auto err = this->streamer->close(); // close or closeSend

    if (err.matches(freighter::UNREACHABLE) && breaker.wait()) return runInternal();
    this->stop();
}

void Control::run() {
    try{
        runInternal();
    } catch (const std::exception &e) {
        LOG(ERROR) << "[Control] Unhandled standard exception: " << e.what();
        this->stop();
    } catch (...) {
        LOG(ERROR) << "[Control] Unhandled unknown exception";
        this->stop();
    }
}
