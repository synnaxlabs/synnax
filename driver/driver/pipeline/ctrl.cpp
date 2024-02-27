// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include "driver/pipeline/ctrl.h"
#pragma once

using namespace pipeline;

Ctrl::Ctrl(){}

Ctrl::Ctrl(synnax::WriterConfig writer_config,
           std::shared_ptr<synnax::Synnax> client,
           std::unique_ptr<daq::Writer> daq_writer):
           writer_config(writer_config),
           client(client),
           daq_writer(std::move(daq_writer)) {
}

void Ctrl::start() {
    running = true;
    ctrl_thread = std::thread(&Ctrl::run, this);
}

void Ctrl::stop() {
    running = false;
    ctrl_thread.join();
}

void Ctrl::run(){

    auto daq_err = daq_writer->start();
    if(!daq_err.ok()) { // daq read error
        if (daq_err.type == freighter::TYPE_TRANSIENT_HARDWARE_ERROR && breaker->wait()) run();
        return;
    }

    // start synnax writer
    auto [writer, wo_err] = client->telem.openWriter(writer_config);
    if (!wo_err.ok()) { // synnax write error
        daq_writer->stop();
        if (wo_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
        return;
    }

    // start synnax streamer
    auto [streamer, so_err] = client->telem.openStreamer(streamer_config);
    if (!so_err.ok()) { // synnax write error
        daq_writer->stop();
        writer->close();
        if (so_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
        return;
    }

    bool retry = false;

    while(running){
        auto [cmd_frame, cmd_err] = streamer->read();
        if(!cmd_err.ok()){
            if(cmd_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
            return;
        }

        //check if we received anything from the streamer


        auto [ack_frame, daq_err] = daq_writer->write(std::move(cmd_frame));
        if(!daq_err.ok()){
            if(daq_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
            return;
        }

        auto write_ok = writer->write(std::move(ack_frame));
        if(!write_ok.ok()){
            if(write_ok.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
            return;
        }

        // synnax commit acknowledgement
        auto now = synnax::TimeStamp::now();
        if (now - last_commit > commit_interval) {
            auto [end, ok] = writer.commit();
            auto err = writer.error();
            if (!ok) {
                retry = error.type == freighter::TYPE_UNREACHABLE;
                std::cout << "committing failed" << std::endl;
            }
            last_commit = now;
        }
    }
    daq_writer->stop();
    auto err = writer.close();
    if (retry && breaker->wait()) run();
}


//void Inbound::execute() {
//    daq_writer->start();
//    while (running) {
//        auto [cmd_frame, cmd_err] = streamer->read();
//        auto [ack_frame, daq_err] = daq_writer->write(std::move(cmd_frame));
//
//
//        auto write_ok = writer->write(std::move(ack_frame));
//    }
//    daq_writer->stop();
//    writer->close();
//}
