// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include "driver/pipeline/outbound.h"
#include "driver/errors/errors.h"
#include "nlohmann/json.hpp"

using json = nlohmann::json;

using namespace pipeline;


void Outbound::start() {
    running = true;
    exec_thread = std::thread(&Outbound::run, this);
}

void Outbound::stop() {
    running = false;
    exec_thread.join();
}

void Outbound::run() {
    auto dq_err = daq_reader->start();
    if (!dq_err.ok()) {
        if (dq_err.type == TYPE_TRANSIENT_HARDWARE_ERROR && breaker->wait()) run();
        return;
    }

    auto [writer, wo_err] = client->telem.openWriter(writer_config);
    if (!wo_err.ok()) {
        daq_reader->stop();
        if (wo_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
        return;
    }

    bool retry = false;
    while (running) {
        auto [frame, error] = daq_reader->read();
        if (!error.ok()) {
            // Any other type means we've encountered a critical hardware failure
            // or configuration error and can't proceed.
            retry = error.type == TYPE_TRANSIENT_HARDWARE_ERROR;
        }
        if (!writer.write(std::move(frame))) {
            auto err = writer.error();
            if (!err.ok()) {
                retry = error.type == freighter::TYPE_UNREACHABLE;
                break;
            }
        }

        auto now = synnax::TimeStamp::now();
        if (now - last_commit > commit_interval) {
            auto ok = writer.commit().second;
            if (!ok) {
                auto err = writer.error();
                retry = error.type == freighter::TYPE_UNREACHABLE;
                break;
            }
            last_commit = now;
        }
    }

    daq_reader->stop();
    writer.close();
    if (retry && breaker->wait()) run();
}

