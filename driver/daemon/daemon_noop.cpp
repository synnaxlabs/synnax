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
void run(const Config &config, int argc, char *argv[]) { config.callback(argc, argv); }

xerrors::Error install_service() {
    return xerrors::Error("install_service not supported on " + xos::get());
}
xerrors::Error uninstall_service() {
    return xerrors::Error("uninstall_service not supported on " + xos::get());
}
xerrors::Error start_service() {
    return xerrors::Error("start_service not supported on " + xos::get());
}
xerrors::Error stop_service() {
    return xerrors::Error("stop_service not supported on " + xos::get());
}
xerrors::Error restart_service() {
    return xerrors::Error("restart_service not supported on " + xos::get());
}

xerrors::Error view_logs() {
    return xerrors::Error("view_logs not supported on " + xos::get());
}
xerrors::Error status() {
    return xerrors::Error("status not supported on " + xos::get());
}
} // namespace daemond
