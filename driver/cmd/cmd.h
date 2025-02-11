// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// std
#include <string>

/// @brief the configuration for opening a connection to the driver.
namespace cmd {
/// @brief exec runs the CLI command.
int exec(int argc, char **argv);

std::string version();

namespace sub {
/// @brief logs the user into a Synnax cluster.
int login(int argc, char **argv);

/// @brief starts the driver process.
int start(int argc, char **argv);

/// @brief returns driver version info.
int version(int argc, char **argv);

/// @brief starts the driver as a background daemon.
int service_start(int argc, char **argv);

/// @brief stops the driver background daemon.
int service_stop(int argc, char **argv);

/// @brief restarts the driver background daemon.
int service_restart(int argc, char **argv);

/// @brief installs the driver as a background daemon.
int service_install(int argc, char **argv);

/// @brief uninstalls the driver background daemon.
int service_uninstall(int argc, char **argv);

/// @brief views the status of the driver background daemon.
int service_status(int argc, char **argv);

/// @brief views the logs of the driver background daemon.
int service_view_logs(int argc, char **argv);

/// @brief clears the driver persisted state.
int clear(int argc, char **argv);
}}
