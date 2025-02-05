#pragma once

#include <functional>
#include <string>

namespace daemon {

// Callback type for the main application logic
using ApplicationCallback = std::function<void(int argc, char* argv[])>;

// Status codes that can be reported to the system service manager
enum class Status {
    INITIALIZING,
    READY,
    RUNNING,
    STOPPING,
    ERROR
};

// Configuration for the daemon
struct Config {
    // How often to send watchdog notifications (in seconds)
    int watchdog_interval = 10;
    // The application's main logic callback
    ApplicationCallback callback;
};

// Runs the application as a daemon with the given configuration
void run(const Config& config, int argc, char* argv[]);

// Updates the daemon's status
void update_status(Status status, const std::string& message = "");

// Notifies the service manager that the watchdog is still alive
void notify_watchdog();

}  // namespace daemon 