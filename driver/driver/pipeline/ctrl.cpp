// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


#include "driver/driver/pipeline/ctrl.h"
#include "driver/driver/errors/errors.h"
#pragma once

using namespace pipeline;

Ctrl::Ctrl(){}

Ctrl::Ctrl(synnax::StreamerConfig streamer_config,
           synnax::WriterConfig writer_config,
           std::shared_ptr<synnax::Synnax> client,
           std::unique_ptr<daq::daqWriter> daq_writer):
           streamer_config(streamer_config),
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
    ctrl_thread.detach();
//    ctrl_thread.join(); wont work if blocked by streamer.read
}

void Ctrl::run(){
    // start daq writer
    auto daq_err = daq_writer->start();
    if(!daq_err.ok()) { // daq read error
        if (daq_err.type == driver::TYPE_TRANSIENT_HARDWARE_ERROR && breaker->wait()) run();
        else if(daq_err.type == driver::TYPE_CRITICAL_HARDWARE_ERROR) {
            this->error_info = daq_writer->getErrorInfo();
            this->postError();
            daq_writer->stop(); // TODO: remove this line? Error Handling
            return;
        }
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
    //print streamer config keys
//    for (auto const& x : streamer_config.channels) {
//        std::cout << "Channel :" << x << std::endl;
//    }
    auto [streamer, so_err] = client->telem.openStreamer(streamer_config);
    if (!so_err.ok()) { // synnax write error
        daq_writer->stop();
        writer.close();
        if (so_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
        return;
    }

    bool retry = false;
    int count = 0;
    while(running){
        std::cout << "Running" << std::endl;
        //check if we received anything from the streamer
        auto [cmd_frame, cmd_err] = streamer.read(); // blocks until we receive a frame from the streamer
//        std::cout << "command frame found" << std::endl;
        if(!cmd_err.ok()){
            if(cmd_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
            return;
        }
        // write to daq
//        std::cout << "Writing to daq" << std::endl;
        // print out value of cmd_frame
//        std::cout << "Command Frame: " << cmd_frame.series << std::endl;
        auto [ack_frame, daq_err] = daq_writer->write(std::move(cmd_frame));
        if(!daq_err.ok()){
            if(daq_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) run();
            else if(daq_err.type == driver::TYPE_CRITICAL_HARDWARE_ERROR) {
                this->error_info = daq_writer->getErrorInfo();
                this->postError();
                daq_writer->stop(); // TODO: remove this line? Error Handling
                writer.close();
                return;
            }
            return;
        }
        // write ack_frame to server
        if(!writer.write(std::move(ack_frame))){
            auto err = writer.error();
            if(!err.ok()){
                retry = daq_err.type == freighter::TYPE_UNREACHABLE; // TODO: come back to make sense of this
                std::cout << "failed to write" << std::endl;
                break;
            }
        }
        // commit acknowledgement
        auto now = synnax::TimeStamp::now();
        if (now - last_commit > commit_interval) {
            auto [end, ok] = writer.commit();
            auto err = writer.error();
            if (!ok) {
                retry = daq_err.type == freighter::TYPE_UNREACHABLE;
                std::cout << "committing failed" << std::endl;
            }
            last_commit = now;
        }
    }
    daq_writer->stop();
    auto err = writer.close();
    if (retry && breaker->wait()) run();
}

void Ctrl::setStateChannelKey(synnax::ChannelKey state_channel_key, synnax::ChannelKey state_channel_idx_key) {
    this->state_channel_key = state_channel_key;
    this->state_channel_idx_key = state_channel_idx_key;

    this->state_writer_config = synnax::WriterConfig{
            std::vector<synnax::ChannelKey>{state_channel_key, state_channel_idx_key},
            synnax::TimeStamp::now(),
            std::vector<synnax::Authority>{synnax::ABSOLUTTE, synnax::ABSOLUTTE},
            synnax::ControlSubject{"state_writer"}
    };

    auto [writer,wErr] = client->telem.openWriter(state_writer_config); // perform error handling for opening stateWriter
    this->state_writer = std::unique_ptr<synnax::Writer>(&writer);
}

void Ctrl::postError() {
    auto frame = synnax::Frame(2);
    frame.add(state_channel_idx_key, synnax::Series(std::vector<uint64_t>{synnax::TimeStamp::now().value}, synnax::TIMESTAMP));
    frame.add(state_channel_key, synnax::Series(std::vector<std::string>{this->error_info.dump()}));
    state_writer->write(std::move(frame));
    state_writer->commit();
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
