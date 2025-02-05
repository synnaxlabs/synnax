#include "driver/daemon.h"
#include <systemd/sd-daemon.h>
#include <thread>
#include <mutex>
#include <condition_variable>
#include "glog/logging.h"

namespace daemon {

std::mutex mtx;
std::condition_variable cv;
bool should_stop = false;

void update_status(Status status, const std::string& message) {
    std::string status_msg = "STATUS=";
    switch (status) {
        case Status::INITIALIZING:
            status_msg += "Initializing";
            break;
        case Status::READY:
            status_msg += "Ready";
            break;
        case Status::RUNNING:
            status_msg += "Running";
            break;
        case Status::STOPPING:
            status_msg += "Stopping";
            break;
        case Status::ERROR:
            status_msg += "Error";
            break;
    }

    if (!message.empty()) {
        status_msg += ": " + message;
    }

    if (status == Status::READY) {
        status_msg += "\nREADY=1";
    } else if (status == Status::STOPPING) {
        status_msg += "\nSTOPPING=1";
    }

    sd_notify(0, status_msg.c_str());
}

void notify_watchdog() {
    sd_notify(0, "WATCHDOG=1");
}

void run(const Config& config, int argc, char* argv[]) {
    update_status(Status::INITIALIZING);

    // Start watchdog thread
    std::thread watchdog([&]() {
        while (!should_stop) {
            notify_watchdog();
            std::this_thread::sleep_for(std::chrono::seconds(config.watchdog_interval));
        }
    });

    update_status(Status::READY);
    
    // Run the main application logic
    try {
        config.callback(argc, argv);
    } catch (const std::exception& e) {
        update_status(Status::ERROR, e.what());
        LOG(ERROR) << "Application error: " << e.what();
    }

    // Cleanup
    update_status(Status::STOPPING);
    {
        std::lock_guard<std::mutex> lock(mtx);
        should_stop = true;
    }
    watchdog.join();
}

}  // namespace daemon
