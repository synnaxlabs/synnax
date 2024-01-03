
#include "synnax/synnax.h"
#include <memory>
#include <thread>
#include <atomic>

namespace hardware {
class Reader {
public:
    virtual freighter::Error start() = 0;
    virtual std::pair<synnax::Frame, freighter::Error> read() = 0;
    virtual freighter::Error stop() = 0;
};
}

namespace pipeline {
class Read {
private:
    // Synnax
    std::unique_ptr<synnax::Synnax> client;
    std::unique_ptr<synnax::Writer> writer;
    synnax::WriterConfig writer_config;

    // DAQ
    std::unique_ptr<hardware::Reader> reader;

    // Threading
    std::atomic<bool> running;
    std::mutex running_mut;
    std::thread thread;

    freighter::Error execute();

public:
    void start();

    void stop();
};

void Read::execute() {
    while (running) {
        auto [frame, err] = reader->read();
        if (!err.ok()) {
            std::cerr << err.message() << std::endl;
            continue;
        }
        auto [err] = writer->write(frame);
        if (!err.ok()) {
            std::cerr << err.message() << std::endl;
            continue;
        }
    }
};

void Read::start() {
    std::thread tmp(&Read::execute, this);
}
}
