
// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/driver/driver.h"

driver::Heartbeat::Heartbeat(
        RackKey rack,
        std::shared_ptr<Synnax> client,
        breaker::Breaker breaker
) :
        rack_key(rack),
        client(std::move(client)),
        generation(generation),
        version(0),
        breaker(std::move(breaker)),
        running(false),
        exit_err(freighter::NIL) {
}

freighter::Error driver::Heartbeat::start(std::latch &latch) {
    auto [hb_channel, err] = client->channels.retrieve("sy_rack_heartbeat");
    if (err) return err;
    running = true;
    channel = hb_channel;
    exec_thread = std::thread(&Heartbeat::run, this);
    return freighter::NIL;
}

freighter::Error driver::Heartbeat::stop() {
    running = false;
    exec_thread.join();
    return exit_err;
}

void driver::Heartbeat::run() {
    std::vector channels = {channel.key};
    auto [writer, err] = client->telem.openWriter(WriterConfig{.channels = channels});
    if (err) {
        if (err.type == freighter::TYPE_UNREACHABLE && breaker.wait()) run();
        exit_err = err;
        return;
    }

    while (running) {
        auto heartbeat = static_cast<std::uint64_t>(rack_key.value) << 32 | version;
        auto series = Series(std::vector{heartbeat});
        auto fr = Frame(1);
        fr.add(channel.key, std::move(series));
        if (!writer.write(std::move(fr))) {
            auto w_err = writer.error();
            if (w_err.type == freighter::TYPE_UNREACHABLE && breaker.wait()) run();
            exit_err = w_err;
            break;
        }
        version++;
    }
    writer.close();
}