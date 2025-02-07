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

namespace daemond {
freighter::Error install_service() { return freighter::NIL; }
freighter::Error uninstall_service() { return freighter::NIL; }
freighter::Error start_service() { return freighter::NIL; }
freighter::Error stop_service() { return freighter::NIL; }
freighter::Error restart_service() { return freighter::NIL; }
void run(const Config &config, int argc, char *argv[]) { config.callback(argc, argv); }
freighter::Error view_logs() { return freighter::NIL; }
} // namespace daemond
