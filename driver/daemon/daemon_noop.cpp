// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// @brief noop implementation of daemon functions that do nothing on MacOS and Windows.

/// internal
#include "driver/daemon/daemon.h"
#include "x/cpp/xos/xos.h"

namespace daemond {
void run(const Config &config, const int argc, char *argv[]) {
    config.callback(argc, argv);
}

const auto NOT_SUPPORTED = xerrors::Error(
    "running the driver as a daemon is not supported on" + xos::get() +
    ". Use the -s flag to start in standalone mode");

xerrors::Error install_service() { return NOT_SUPPORTED; }
xerrors::Error uninstall_service() { return NOT_SUPPORTED; }
xerrors::Error start_service() { return NOT_SUPPORTED; }
xerrors::Error stop_service() { return NOT_SUPPORTED; }
xerrors::Error restart_service() { return NOT_SUPPORTED; }
xerrors::Error view_logs() { return NOT_SUPPORTED; }
xerrors::Error status() { return NOT_SUPPORTED; }
} // namespace daemond
