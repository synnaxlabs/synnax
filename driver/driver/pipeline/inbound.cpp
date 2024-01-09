// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/pipeline/inbound.h"

using namespace pipeline;

void Inbound::start() {
    running = true;
    exec_thread = std::thread(&Inbound::execute, this);
}

void Inbound::stop() {
    running = false;
    exec_thread.join();
}

void Inbound::execute() {
    daq_writer->start();
    while (running) {
        auto [cmd_frame, cmd_err] = streamer->read();
        auto [ack_frame, daq_err] = daq_writer->write(std::move(cmd_frame));


        auto write_ok = writer->write(std::move(ack_frame));
    }
    daq_writer->stop();
    writer->close();
}
