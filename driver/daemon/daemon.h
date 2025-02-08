// Copyright 2025 Synnax Labs, Inc.
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
    ERROR_  // ERROR is already a reserved macro in <winerror.h>
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
} // namespace daemon
