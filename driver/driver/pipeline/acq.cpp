
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

using json = nlohmann::json;

//using namespace pipeline;

Acq::Acq(std::unique_ptr<daq::AcqReader> daq_reader,
        synnax::WriterConfig writer_config,
        synnax::StreamerConfig streamer_config) :
        daq_reader(std::move(daq_reader)), writer_config(writer_config), streamer_config(streamer_config) {}

void Acq::start() {
    running = true;
    exec_thread = std::thread(&Acq::run, this);
}

void Acq::stop() {
    running = false;
    exec_thread.join();
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
        if (!writer.write(std::move(frame))) {
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



//
//std::string Acq::runInternal(){
//
//    running = true;
//    while (running) {
//        auto [frame, error] = daq_reader->read();
//
//        if (!error.ok()) { // daq read error
//            // Any other type means we've encountered a critical hardware failure
//            // or configuration error and can't proceed.
//            return TYPE_TRANSIENT_HARDWARE_ERROR;
//        }
//
//        if (!writer.write(std::move(frame))) { // synnax write error
//            auto err = writer.error();
//            if (!err.ok()) {
//                return freighter::TYPE_UNREACHABLE;
//            }
//        }
//
//        auto now = synnax::TimeStamp::now();
//
//        if (now - last_commit > commit_interval) {
//            auto ok = writer.commit().second;
//            auto err = writer.error();
//            if (!ok) { // synnax commit error
//                return freighter::TYPE_UNREACHABLE;
//            }
//            last_commit = now;
//        }
//    }
//}
//
//void Acq::run(){
//    bool retry = false;
//
//    // attempt to open daq specific reader
//    auto dq_err = daq_reader->start();
//    if (!dq_err.ok()) {
//        if (dq_err.type == TYPE_TRANSIENT_HARDWARE_ERROR && breaker->wait()) runInternal();
//        return;
//    }
//
//    // attempt open synnax writer
//    auto [writer, wo_err] = client->telem.openWriter(writer_config);
//    if (!wo_err.ok()) {
//        daq_reader->stop();
//        if (wo_err.type == freighter::TYPE_UNREACHABLE && breaker->wait()) runInternal();
//        return;
//    }
//
//    runInternal();
//
//    daq_reader->stop(); // close daq readeer
//    writer.close(); // close synnax writer
//
//    if(retry && breaker->wait()) runInternal();
//}


