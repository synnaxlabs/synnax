// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std.
#include <functional>

/// internal.
#include "x/cpp/errors/errors.h"

namespace driver::daemon {
// Callback type for the main application logic
using ApplicationCallback = std::function<void(int argc, char *argv[])>;

// Status codes that can be reported to the system service manager
enum class Status {
    INITIALIZING,
    READY,
    RUNNING,
    STOPPING,
    ERROR_ // ERROR is already a reserved macro in <winerror.h>
};

// Configuration for the daemon
struct Config {
    // How often to send watchdog notifications (in seconds)
    int watchdog_interval = 10;
    // The application's main logic callback
    ApplicationCallback callback;
};

// Service management functions
x::errors::Error install_service();

x::errors::Error uninstall_service();

// Service control functions
x::errors::Error start_service();

x::errors::Error stop_service();

x::errors::Error restart_service();

// Service status functions.
x::errors::Error view_logs();

x::errors::Error status();

// Runs the application as a daemon with the given configuration
void run(const Config &config, int argc, char *argv[]);
}
