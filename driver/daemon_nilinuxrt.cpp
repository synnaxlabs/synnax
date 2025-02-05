#include "driver/daemon.h"
#include "glog/logging.h"

namespace daemon {

void update_status(Status status, const std::string& message) {
    // NILinuxRT doesn't have a service manager, so we just log the status
    std::string status_str;
    switch (status) {
        case Status::INITIALIZING: status_str = "Initializing"; break;
        case Status::READY: status_str = "Ready"; break;
        case Status::RUNNING: status_str = "Running"; break;
        case Status::STOPPING: status_str = "Stopping"; break;
        case Status::ERROR: status_str = "Error"; break;
    }
    
    if (!message.empty()) {
        LOG(INFO) << "[daemon] Status: " << status_str << " - " << message;
    } else {
        LOG(INFO) << "[daemon] Status: " << status_str;
    }
}

void notify_watchdog() {
    // No-op for NILinuxRT
}

void run(const Config& config, int argc, char* argv[]) {
    update_status(Status::INITIALIZING);
    update_status(Status::READY);
    
    try {
        config.callback(argc, argv);
    } catch (const std::exception& e) {
        update_status(Status::ERROR, e.what());
        LOG(ERROR) << "Application error: " << e.what();
    }

    update_status(Status::STOPPING);
}

}  // namespace daemon 