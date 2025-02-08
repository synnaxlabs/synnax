#pragma once

/// std.
#include <functional>

/// internal.
#include "freighter/cpp/freighter.h"

namespace daemond {
// Callback type for the main application logic
using ApplicationCallback = std::function<void(int argc, char *argv[])>;

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

// Service management functions
freighter::Error install_service();

freighter::Error uninstall_service();

// Service control functions
freighter::Error start_service();

freighter::Error stop_service();

freighter::Error restart_service();

// Service status functions.
freighter::Error view_logs();

// Runs the application as a daemon with the given configuration
void run(const Config &config, int argc, char *argv[]);
} // namespace daemond
