// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#pragma once

#include "driver/pipeline/control.h"

#include <utility>
#include "driver/errors/errors.h"

using namespace pipeline;

Control::Control(
    std::shared_ptr<task::Context> ctx,
    synnax::StreamerConfig streamer_config,
    std::unique_ptr<pipeline::Sink> sink,
    const breaker::Config &breaker_config
): ctx(std::move(ctx)),
   streamer_config(std::move(streamer_config)),
   sink(std::move(sink)),
   cmd_breaker(breaker::Breaker(breaker_config)) {
}


void Control::start() {
    cmd_running = true;
    cmd_thread = std::thread(&Control::run, this);
}

void Control::stop() {
    cmd_running = false;
    cmd_thread.join();
}

void Control::run() {
    auto [streamer, so_err] = ctx->client->telem.openStreamer(streamer_config);
    if (so_err) {
        if (    so_err.matches(freighter::UNREACHABLE) 
            &&  cmd_breaker.wait(so_err.message())) {
            return run();
        }
    }

    while (cmd_running) {
        auto [cmd_frame, cmd_err] = streamer.read();
        if (cmd_err) break;
        auto daq_err = sink->write(std::move(cmd_frame));    
    }

    const auto err = streamer.close();
    if (err.matches(freighter::UNREACHABLE) && cmd_breaker.wait()){        
        return run();
    }
}
