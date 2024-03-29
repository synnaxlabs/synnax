// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#pragma once

#include "driver/driver/pipeline/control.h"

#include <latch>
#include <utility>
#include "driver/driver/errors/errors.h"

using namespace pipeline;

Control::Control(
    std::shared_ptr<task::Context> ctx,
    synnax::StreamerConfig streamer_config,
    synnax::WriterConfig writer_config,
    std::unique_ptr<pipeline::Sink> sink,
    breaker::Breaker breaker,
    synnax::Rate state_rate
): ctx(std::move(ctx)),
   streamer_config(std::move(streamer_config)),
   writer_config(std::move(writer_config)),
   sink(std::move(sink)),
   state_breaker(std::move(breaker)),
   cmd_breaker(std::move(cmd_breaker)),
   state_rate(state_rate) {
}


void Control::start() {
    std::latch l = std::latch(2);
    cmd_running = true;
    cmd_thread = std::thread(&Control::runCommands, this, std::ref(l));
    state_running = true;
    acks_thread = std::thread(&Control::runStateUpdates, this, std::ref(l));
}

void Control::stop() {
    cmd_running = false;
    cmd_thread.join();
    state_running = false;
    acks_thread.join();
}

const std::vector RETRY_ON = {
    freighter::UNREACHABLE,
    freighter::STREAM_CLOSED,
};

void Control::runCommands(std::latch& latch) {
    auto [streamer, so_err] = ctx->client->telem.openStreamer(streamer_config);
    if (so_err) {
        if (so_err.matches(RETRY_ON) && cmd_breaker.wait()) return runCommands(latch);
        return latch.count_down();
    }

    while (cmd_running) {
        auto [cmd_frame, cmd_err] = streamer.read();
        if (cmd_err) break;
        auto [state_frame, daq_err] = sink->write(std::move(cmd_frame));
        state_mutex.lock();
        curr_state = std::move(state_frame);
        state_writer.write(curr_state);
        state_mutex.unlock();
    }
    if (
        const auto err = streamer.close();
        err.matches(RETRY_ON) && cmd_breaker.wait()
    )
        return runCommands(latch);

    latch.count_down();
}

void Control::runStateUpdates(std::latch& latch) {
    auto [sw, so_err] = ctx->client->telem.openWriter(writer_config);
    if (so_err) {
        if (so_err.matches(RETRY_ON) && state_breaker.wait())
            return
                    runStateUpdates(latch);
        return latch.count_down();
    }
    state_writer = std::move(sw);

    while (state_running) {
        std::this_thread::sleep_for(state_rate.period().nanoseconds());
        state_mutex.lock();
        if (!state_writer.write(curr_state)) break;
        state_mutex.unlock();
    }

    auto err = state_writer.close();
    if (err.matches(RETRY_ON) && state_breaker.wait()) runStateUpdates(latch);
}
