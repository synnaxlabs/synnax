
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include <thread>
#include "driver/pipeline/acq.h"
#include "driver/errors/errors.h"
#include "nlohmann/json.hpp"
#include "acq.h"


#pragma once

using json = nlohmann::json;

//using namespace pipeline;

Acq::Acq(){}

void Acq::Acq(synnax::WriterConfig writer_config,
         std::unique_ptr<synnax::Synnax> client
         std::vector<ni::channel_config> channels,
         uint64_t acquisition_rate,
         uint64_t stream_rate,
         Taskhandle taskHandle):
         daq_reader(std::make_unique<niDaqReader>(taskHandle)),
         client(std::move(client)),
         writer_config(writer_config){

    // instantiate the daq_reader
    static_cast <niDaqReader*>(this->daq_reader.get())->init(channels, acquisition_rate, stream_rate);

}

void Acq::start() {
    running = true;
    acq_thread = std::thread(&Acq::run);
}

void Acq::stop() {
    running = false;
//    acq_thread.join(); // FIXME: I dont want to call join im p sure (elham)
}

void Acq::run() {
    // start daq read
    auto dq_err = daq_reader->start();
    if (!dq_err.ok()) { // daq read error
        if (dq_err.type == TYPE_TRANSIENT_HARDWARE_ERROR && breaker->wait()) run();
        return;
    }

    // start synnax writer
    auto [writer, wo_err] = client->telem.openWriter(writer_config);
    if (!wo_err.ok()) { // synnax write error
        daq_reader->stop();
        if (wo_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
        return;
    }

    bool retry = false;
    while (running) {
        // perform a daq read
        auto [frame, error] = daq_reader->read();
        if (!error.ok()) {
            // Any other type means we've encountered a critical hardware failure
            // or configuration error and can't proceed.
            retry = error.type == TYPE_TRANSIENT_HARDWARE_ERROR;
        }
        if (!writer.write(std::move(frame))) { // write frame to channel
            auto err = writer.error();
            if (!err.ok()) {
                retry = error.type == freighter::TYPE_UNREACHABLE;
                break;
            }
        }
        // synnax commit
        auto now = synnax::TimeStamp::now();
        if (now - last_commit > commit_interval) {
            auto ok = writer.commit().second;
            auto err = writer.error();
            if (!ok) {
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



