// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/heartbeat/heartbeat.h"
#include "glog/logging.h"

heartbeat::Heartbeat::Heartbeat(
    const RackKey rack_key,
    const std::shared_ptr<Synnax> &client,
    const breaker::Config &breaker_config
) : rack_key(rack_key),
    client(client),
    version(0),
    breaker(breaker_config),
    running(false) {
}

freighter::Error heartbeat::Heartbeat::start(std::atomic<bool> &done) {
    if(running) return freighter::NIL;
    LOG(INFO) << "[heartbeat] starting up";
    if (const auto err = startGuarded(); err) {
        if (err.matches(freighter::UNREACHABLE) && breaker.wait(err)) start(done);
        done = true;
        return err;
    }
    running = true;
    run_thread = std::thread(&Heartbeat::run, this, std::ref(done));
    return freighter::NIL;
}

const std::string RACK_HEARTBEAT_CHANNEL = "sy_rack_heartbeat";

freighter::Error heartbeat::Heartbeat::startGuarded() {
    auto [hb_channel, err] = client->channels.retrieve(RACK_HEARTBEAT_CHANNEL);
    channel = hb_channel;
    return err;
}

freighter::Error heartbeat::Heartbeat::stop() {
    LOG(INFO) << "[heartbeat] stop called";
    if (!running) return freighter::NIL;
    running = false;
    run_thread.join();
    LOG(INFO) << "[heartbeat] shut down";
    return run_err;
}

void heartbeat::Heartbeat::run(std::atomic<bool> &done) {
    const auto err = runGuarded();
    if (err.matches(freighter::UNREACHABLE) && breaker.wait(err)) return run(done);
    done = true;
    done.notify_all();
    run_err = err;
    LOG(INFO) << "[heartbeat] run thread exiting";
}

freighter::Error heartbeat::Heartbeat::runGuarded() {
    const std::vector channels = {channel.key};
    LOG(INFO) << "[heartbeat] opening writer";
    auto [writer, err] = client->telem.openWriter(WriterConfig{.channels = channels});
    if (err) return err;
    LOG(INFO) << "[heartbeat] operational";
    breaker.reset();
    while (running) {
        // The first 32 bits of the heartbeat are the rack key, while the second 32
        // bits are the current version.
        const auto heartbeat = static_cast<std::uint64_t>(rack_key) << 32 | version;
        if (!writer.write(Frame(channel.key, Series(heartbeat)))) break;
        breaker.reset();
        std::this_thread::sleep_for(std::chrono::seconds(1));
        version++;
    }
    return writer.close();
}
