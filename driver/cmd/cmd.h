// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

/// @brief the configuration for opening a connection to the driver.
namespace cmd {
/// @brief exec runs the CLI command.
int exec(int argc, char **argv);

namespace priv {
int login(int argc, char **argv);

int start(int argc, char **argv);

int version(int argc, char **argv);

int service_start(int argc, char **argv);

int service_stop(int argc, char **argv);

int service_restart(int argc, char **argv);

int service_install(int argc, char **argv);

int service_uninstall(int argc, char **argv);

int service_view_logs(int argc, char **argv);
}}
