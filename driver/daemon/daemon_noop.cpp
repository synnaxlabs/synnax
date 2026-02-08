// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/// @brief noop implementation of daemon functions that do nothing on macOS and Windows.

#include "x/cpp/os/os.h"

#include "driver/daemon/daemon.h"

namespace driver::daemon {
void run(const Config &config, const int argc, char *argv[]) {
    config.callback(argc, argv);
}

const auto NOT_SUPPORTED = x::errors::Error(
    x::errors::NOT_SUPPORTED,
    "running the driver as a daemon is not supported on " + x::os::get() +
        ". Use the -s flag to start in standalone mode"
);

x::errors::Error install_service() {
    return NOT_SUPPORTED;
}
x::errors::Error uninstall_service() {
    return NOT_SUPPORTED;
}
x::errors::Error start_service() {
    return NOT_SUPPORTED;
}
x::errors::Error stop_service() {
    return NOT_SUPPORTED;
}
x::errors::Error restart_service() {
    return NOT_SUPPORTED;
}
x::errors::Error view_logs() {
    return NOT_SUPPORTED;
}
x::errors::Error status() {
    return NOT_SUPPORTED;
}
}
