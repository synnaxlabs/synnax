// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <functional>

#include "x/cpp/errors/errors.h"

namespace driver::daemon {
/// @brief the application's main logic. Returns the process exit code.
using ApplicationCallback = std::function<int()>;

/// @brief status codes reported to the system service manager.
enum class Status {
    INITIALIZING,
    READY,
    RUNNING,
    STOPPING,
    ERROR_ // ERROR is already a reserved macro in <winerror.h>
};

struct Config {
    /// @brief how often to send watchdog notifications, in seconds.
    int watchdog_interval = 10;
    /// @brief the application's main logic callback.
    ApplicationCallback callback;
};

/// @brief returns true when the Driver service is installed on this computer.
bool installed();

x::errors::Error install_service();

x::errors::Error uninstall_service();

x::errors::Error start_service();

x::errors::Error stop_service();

x::errors::Error restart_service();

x::errors::Error view_logs();

x::errors::Error status();

/// @brief sets the variables in the Driver's environment file into the process
/// environment, so login reads the same variables as the service. A variable already in
/// the environment is kept. Does nothing where the service reads no environment file:
/// on Windows, and on macOS while the service is not installed.
x::errors::Error load_env();

/// @brief runs the application as a daemon with the given configuration. Returns the
/// exit code of the callback, or 1 when it throws or the daemon cannot start.
int run(const Config &config);
}
