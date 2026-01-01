// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

// IT IS ABSOLUTELY MISSION CRITICAL THAT THIS BLOCK IS THE FIRST INCLUDE IN THIS FILE.
// Otherwise, you will see a bunch of linker errors.
#ifdef _WIN32
#ifndef WIN32_LEAN_AND_MEAN
#define WIN32_LEAN_AND_MEAN
#endif
#include <winsock2.h>
#endif
// END OF MISSION CRITICAL CODE BLOCK.

#include <iostream>
#include <string>

#include "glog/logging.h"

#include "client/cpp/synnax.h"
#include "x/cpp/xlog/xlog.h"
#include "x/cpp/xshutdown/xshutdown.h"

#include "core/pkg/version/version.h"
#include "driver/cmd/cmd.h"
#include "driver/daemon/daemon.h"
#include "driver/rack/rack.h"

/// @brief the configuration for opening a connection to the driver.
namespace cmd {
/// @brief exec runs the CLI command.
int exec(int argc, char **argv);

std::string version();

namespace sub {
/// @brief logs the user into a Synnax cluster.
int login(xargs::Parser &args);

/// @brief starts the driver process.
int start(xargs::Parser &args);

/// @brief returns driver version info.
int version(xargs::Parser &args);

/// @brief starts the driver as a background daemon.
int service_start(xargs::Parser &args);

/// @brief stops the driver background daemon.
int service_stop(xargs::Parser &args);

/// @brief restarts the driver background daemon.
int service_restart(xargs::Parser &args);

/// @brief installs the driver as a background daemon.
int service_install(xargs::Parser &args);

/// @brief uninstalls the driver background daemon.
int service_uninstall(xargs::Parser &args);

/// @brief views the status of the driver background daemon.
int service_status(xargs::Parser &args);

/// @brief views the logs of the driver background daemon.
int service_view_logs(xargs::Parser &args);

/// @brief clears the driver persisted state.
int clear(xargs::Parser &args);
}
}
