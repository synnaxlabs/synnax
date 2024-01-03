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
