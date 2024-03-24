
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
#include <stdio.h>

#pragma once

using json = nlohmann::json;

//using namespace pipeline;

Acq::Acq(){}

Acq::Acq(synnax::WriterConfig writer_config,
         std::shared_ptr<synnax::Synnax> client,
         std::unique_ptr<daq::AcqReader> daq_reader):
         writer_config(writer_config),
         client(client),
         daq_reader(std::move(daq_reader)) {
}

void Acq::start() {
    running = true;
    acq_thread = std::thread(&Acq::run, this);
}

void Acq::stop() {
    running = false;
    acq_thread.join();
}

void Acq::run() {
    // start daq read
    auto dq_err = daq_reader->start();
    if (!dq_err.ok()) { // daq read error
        if (dq_err.type == driver::TYPE_TRANSIENT_HARDWARE_ERROR && breaker->wait()) run();
        else if(dq_err.type == driver::TYPE_PERMANENT_HARDWARE_ERROR) {
            this->error_info = daq_reader->getErrorInfo();
            daq_reader->stop(); // TODO: remove this line? Error Handling
            return;
        }
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
            retry = error.type == driver::TYPE_TRANSIENT_HARDWARE_ERROR;
            if(error.type == driver::TYPE_PERMANENT_HARDWARE_ERROR) {
                this->error_info = daq_reader->getErrorInfo();
                daq_reader->stop(); // TODO: remove this line? Error Handling
                return;
            }
        }

        if (!writer.write(std::move(frame))) { // write frame to channel
            auto err = writer.error();
            if (!err.ok()) {
                retry = error.type == freighter::TYPE_UNREACHABLE;
                std::cout << "failed to write" << std::endl;
                break;
            }
        }
        // synnax commit
        auto now = synnax::TimeStamp::now();
        if (now - last_commit > commit_interval) {
            auto [end, ok] = writer.commit();
            auto err = writer.error();
            if (!ok) {
                retry = error.type == freighter::TYPE_UNREACHABLE;
                std::cout << "committing failed" << std::endl;
                break;
            }
            last_commit = now;
        }
    }
    daq_reader->stop(); // TODO: catch error
    auto err = writer.close();
    if (retry && breaker->wait()) run();
}

void Acq::setStateChannelKey(synnax::ChannelKey state_channel_key, synnax::ChannelKey state_channel_idx_key) {
    this->state_channel_key = state_channel_key;
    this->state_channel_idx_key = state_channel_idx_key;

    this->state_writer_config = synnax::WriterConfig{
        std::vector<synnax::ChannelKey>{state_channel_key, state_channel_idx_key},
        synnax::TimeStamp::now(),
        std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
        synnax::Subject{"state_writer"}
    };
    freighter::Error wErr;
    [state_writer,wErr] = client->telem.openWriter(state_writer_config); // perform error handling for opening stateWriter
}

void Acq::postError() {
    auto frame = synnax::Frame(2);
    frame.add(state_channel_idx_key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));
    frame.add(state_channel_key, synnax::Series(std::vector<string>{this->error_info.dump()}));
    state_writer.write(std::move(frame));
    state_writer.commit();
}